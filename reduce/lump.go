package reduce

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// Lump projects k onto the quotient state space defined by partition using
// Kemeny-Snell projection:
//
//	P̂ = U·P·V
//
// where V (n×m) is the collector matrix and U (m×n) is the distributor matrix.
//
// When weights is nil, uniform block weights are used (U[i,s] = 1/|Bᵢ|).
// When weights is non-nil (len = n), each block Bᵢ restricts and normalizes
// weights onto Bᵢ — use a stationary distribution for weak lumping.
//
// When names is nil, block names are synthesised from member state names
// joined with "+". Otherwise len(names) must equal len(partition).
//
// Returns [markov.ErrInvalidPartition] for invalid partition,
// [markov.ErrInvalidDistribution] for invalid weights, or
// [markov.ErrNotStochastic] if k is nil, tol < 0, or P̂ fails row-stochasticity.
func Lump(k *markov.Kernel, p markov.Partition, names []string, weights markov.Distribution, tol float64) (*markov.Kernel, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	n := k.NumStates()
	V, err := p.IndicatorMatrix(n)
	if err != nil {
		return nil, err
	}
	if names != nil && len(names) != len(p) {
		return nil, fmt.Errorf("%w: names length %d does not match partition block count %d", markov.ErrInvalidPartition, len(names), len(p))
	}
	U, err := p.DistributorMatrix(n, weights, tol)
	if err != nil {
		return nil, err
	}
	var UP, Phat mat.Dense
	UP.Mul(U, k.P)
	Phat.Mul(&UP, V)

	lumped, err := markov.NewKernelWithTol(&Phat, resolveMacroBlockNames(k, p, names), tol)
	if err != nil {
		return nil, err
	}
	if err := lumped.NormalizeRows(tol); err != nil {
		return nil, err
	}
	return lumped, nil
}

// MustLump is like Lump but panics on error.
func MustLump(k *markov.Kernel, p markov.Partition, names []string, weights markov.Distribution, tol float64) *markov.Kernel {
	lumped, err := Lump(k, p, names, weights, tol)
	if err != nil {
		panic(err)
	}
	return lumped
}

// IsStronglyLumpable reports whether k satisfies the Kemeny-Snell strong
// lumpability condition on partition within tol:
//
//	LumpabilityResidual(k, partition) ≤ tol
//
// When this holds, the lumped process Y_t = φ(X_t) is strictly Markovian for
// every initial distribution, and every valid distributor U yields the same P̂.
func IsStronglyLumpable(k *markov.Kernel, p markov.Partition, tol float64) (bool, error) {
	if tol < 0 {
		return false, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	res, err := LumpabilityResidual(k, p)
	if err != nil {
		return false, err
	}
	return res <= tol, nil
}

// LumpabilityResidual computes the Kemeny-Snell strong lumpability residual:
//
//	R(P, 𝒫) = max_{Bᵢ} max_{s,s'∈Bᵢ} max_{Bⱼ} |(PV)[s,j] − (PV)[s′,j]|
//
// A residual of 0 indicates exact strong lumpability (PV = VP̂).
// Returns [markov.ErrInvalidPartition] for invalid partition.
func LumpabilityResidual(k *markov.Kernel, p markov.Partition) (float64, error) {
	if k == nil || k.P == nil {
		return 0, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	V, err := p.IndicatorMatrix(k.NumStates())
	if err != nil {
		return 0, err
	}
	var PV mat.Dense
	PV.Mul(k.P, V)

	var maxResidual float64
	for _, block := range p {
		if d := blockMaxColumnDiameter(&PV, block, len(p)); d > maxResidual {
			maxResidual = d
		}
	}
	return maxResidual, nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// blockMaxColumnDiameter returns max_{j ∈ 0..m-1} max_{s,s' ∈ block} |(PV)[s,j] - (PV)[s',j]|,
// the maximum disagreement among states in block over their transition probability
// into any macro-block j.
func blockMaxColumnDiameter(PV *mat.Dense, block []int, numBlocks int) float64 {
	if len(block) <= 1 {
		return 0
	}
	var maxDiff float64
	for j := 0; j < numBlocks; j++ {
		if diff := blockColumnDiameter(PV, block, j); diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

// blockColumnDiameter returns max_{s,s' ∈ block} |(PV)[s, col] - (PV)[s', col]|.
func blockColumnDiameter(PV *mat.Dense, block []int, col int) float64 {
	minVal := PV.At(block[0], col)
	maxVal := minVal
	for _, s := range block[1:] {
		v := PV.At(s, col)
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal - minVal
}

// resolveMacroBlockNames returns a copy of names when non-nil, or synthesises
// macro-block names by joining each block's constituent state names with "+".
func resolveMacroBlockNames(k *markov.Kernel, p markov.Partition, names []string) []string {
	if names != nil {
		return slices.Clone(names)
	}
	out := make([]string, len(p))
	for i, block := range p {
		out[i] = synthesizeBlockName(k.StateNames, block)
	}
	return out
}

// synthesizeBlockName joins the names of states in block with "+".
func synthesizeBlockName(stateNames []string, block []int) string {
	parts := make([]string, len(block))
	for j, s := range block {
		if s < len(stateNames) && stateNames[s] != "" {
			parts[j] = stateNames[s]
		} else {
			parts[j] = fmt.Sprintf("%d", s)
		}
	}
	return strings.Join(parts, "+")
}
