package markov

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// Partition represents a set partition of state indices {0, …, n-1} into m
// disjoint, non-empty equivalence classes (blocks) B₀, …, Bₘ₋₁.
//
// A Partition is valid for n states when:
//   - len(p) > 0
//   - every block is non-empty
//   - every index is in [0, n)
//   - every index appears in exactly one block
//   - the union of all blocks equals {0, …, n-1}
type Partition [][]int

// Validate reports whether p is a valid partition of {0, …, n-1}.
// Returns ErrInvalidPartition on any violation.
func (p Partition) Validate(n int) error {
	if n <= 0 {
		return fmt.Errorf("%w: invalid state count %d", ErrInvalidPartition, n)
	}
	if len(p) == 0 {
		return fmt.Errorf("%w: empty partition", ErrInvalidPartition)
	}
	seen := make([]bool, n)
	covered := 0
	for bIdx, block := range p {
		blockCount, err := markBlockStates(block, bIdx, n, seen)
		if err != nil {
			return err
		}
		covered += blockCount
	}
	if covered != n {
		return fmt.Errorf("%w: covers %d of %d states", ErrInvalidPartition, covered, n)
	}
	return nil
}

// IndicatorMatrix constructs the n×m collector matrix V (Kemeny & Snell, 1960 §6.3):
//
//	V[s, j] = 1   if s ∈ Bⱼ
//	V[s, j] = 0   otherwise
//
// Right-multiplying P by V sums transition probabilities into each macro-block:
// (PV)[s, j] = Σ_{t∈Bⱼ} P[s,t].
func (p Partition) IndicatorMatrix(n int) (*mat.Dense, error) {
	if err := p.Validate(n); err != nil {
		return nil, err
	}
	m := len(p)
	V := mat.NewDense(n, m, nil)
	for j, block := range p {
		for _, s := range block {
			V.Set(s, j, 1.0)
		}
	}
	return V, nil
}

// DistributorMatrix constructs the m×n distributor matrix U (Kemeny & Snell, 1960 §6.3):
//
//	U[i, s] ≥ 0,   U[i, s] = 0 for s ∉ Bᵢ,   Σ_{s∈Bᵢ} U[i,s] = 1
//
// U is a left-inverse of V: U·V = Iₘ.
//
// When weights is nil, uniform block weights U[i,s] = 1/|Bᵢ| are used.
// When weights is non-nil (len = n), each row restricts and normalizes weights
// onto Bᵢ: U[i,s] = weights[s] / Σ_{r∈Bᵢ} weights[r].
func (p Partition) DistributorMatrix(n int, weights Distribution, tol float64) (*mat.Dense, error) {
	if err := p.Validate(n); err != nil {
		return nil, err
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", ErrInvalidDistribution, tol)
	}
	if weights != nil && len(weights) != n {
		return nil, fmt.Errorf("%w: weights length %d does not match n=%d", ErrInvalidDistribution, len(weights), n)
	}
	U := mat.NewDense(len(p), n, nil)
	for i, block := range p {
		if err := setDistributorRow(U, i, block, weights, tol); err != nil {
			return nil, err
		}
	}
	return U, nil
}

// ValidatePartition is the top-level form of Partition.Validate.
func ValidatePartition(p Partition, n int) error {
	if p == nil {
		return fmt.Errorf("%w: nil partition", ErrInvalidPartition)
	}
	return p.Validate(n)
}

// IndicatorMatrix is the top-level form of Partition.IndicatorMatrix.
func IndicatorMatrix(p Partition, n int) (*mat.Dense, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: nil partition", ErrInvalidPartition)
	}
	return p.IndicatorMatrix(n)
}

// DistributorMatrix is the top-level form of Partition.DistributorMatrix.
func DistributorMatrix(p Partition, n int, weights Distribution, tol float64) (*mat.Dense, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: nil partition", ErrInvalidPartition)
	}
	return p.DistributorMatrix(n, weights, tol)
}

// ── internal helpers ──────────────────────────────────────────────────────────

// markBlockStates verifies that block bIdx is non-empty and contains only
// unique, in-range state indices in [0, n), marking each state in seen.
func markBlockStates(block []int, bIdx, n int, seen []bool) (int, error) {
	if len(block) == 0 {
		return 0, fmt.Errorf("%w: empty block at index %d", ErrInvalidPartition, bIdx)
	}
	for _, s := range block {
		if s < 0 || s >= n {
			return 0, fmt.Errorf("%w: state index %d out of range [0,%d)", ErrInvalidPartition, s, n)
		}
		if seen[s] {
			return 0, fmt.Errorf("%w: duplicate state index %d", ErrInvalidPartition, s)
		}
		seen[s] = true
	}
	return len(block), nil
}

// setDistributorRow populates row i of the distributor matrix U for equivalence
// class Bᵢ = block:
//
//	U[i, s] = 1 / |Bᵢ|                      when weights == nil (uniform)
//	U[i, s] = w[s] / Σ_{r∈Bᵢ} w[r]          when weights != nil (weighted)
func setDistributorRow(U *mat.Dense, blockIdx int, block []int, weights Distribution, tol float64) error {
	if weights == nil {
		w := 1.0 / float64(len(block))
		for _, s := range block {
			U.Set(blockIdx, s, w)
		}
		return nil
	}
	blockSum, err := validatedBlockWeightSum(block, blockIdx, weights, tol)
	if err != nil {
		return err
	}
	for _, s := range block {
		U.Set(blockIdx, s, clampProbabilityEntry(weights[s])/blockSum)
	}
	return nil
}

// validatedBlockWeightSum validates that every state s ∈ block has weight
// w[s] ≥ -tol and returns the clamped block weight sum Σ_{r∈block} max(w[r], 0).
// Returns ErrInvalidDistribution if any weight is < -tol or the sum is ≤ tol.
func validatedBlockWeightSum(block []int, blockIdx int, weights Distribution, tol float64) (float64, error) {
	blockSum := 0.0
	for _, s := range block {
		w := weights[s]
		if w < -tol {
			return 0, fmt.Errorf("%w: negative weight %g at state %d", ErrInvalidDistribution, w, s)
		}
		if w < 0 && math.Abs(w) <= tol {
			w = 0
		}
		blockSum += w
	}
	if blockSum <= tol {
		return 0, fmt.Errorf("%w: block %d has zero total weight", ErrInvalidDistribution, blockIdx)
	}
	return blockSum, nil
}
