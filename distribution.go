package markov

import (
	"fmt"
	"math"
	"slices"
)

// Distribution is a validated probability vector π ∈ ℝⁿ.
// Every entry π[i] ≥ 0 and Σ π[i] = 1 within tolerance.
type Distribution []float64

// NewDistribution validates probs and returns a cloned Distribution.
// Near-zero negative entries (v ∈ [-tol, 0)) are clamped to 0 after validation.
// Returns ErrInvalidDistribution if probs is empty, tol < 0, any entry is < -tol,
// or the sum deviates from 1 by more than tol.
func NewDistribution(probs []float64, tol float64) (Distribution, error) {
	d := Distribution(slices.Clone(probs))
	if err := d.Validate(tol); err != nil {
		return nil, err
	}
	clampDistributionNegatives(d, tol)
	return d, nil
}

// MustNewDistribution is like NewDistribution but panics on error.
func MustNewDistribution(probs []float64, tol float64) Distribution {
	d, err := NewDistribution(probs, tol)
	if err != nil {
		panic(err)
	}
	return d
}

// Validate reports whether d is a valid probability distribution within tol:
// every entry ≥ -tol and |Σ d[i] - 1| ≤ tol.
func (d Distribution) Validate(tol float64) error {
	if len(d) == 0 {
		return fmt.Errorf("%w: empty distribution", ErrInvalidDistribution)
	}
	if tol < 0 {
		return fmt.Errorf("%w: negative tolerance %g", ErrInvalidDistribution, tol)
	}
	sum := 0.0
	for i, v := range d {
		if v < -tol {
			return fmt.Errorf("%w: negative entry at index %d: %g", ErrInvalidDistribution, i, v)
		}
		sum += v
	}
	if math.Abs(sum-1) > tol {
		return fmt.Errorf("%w: entries sum to %g, want 1", ErrInvalidDistribution, sum)
	}
	return nil
}

// Restrict slices d to the canonical (sorted, deduplicated) subset A of state
// indices and renormalizes to form the conditional distribution on A:
//
//	π_A[i] = π[A[i]] / Σ_{s∈A} π[s]
//
// Returns ErrInvalidDistribution if d is invalid, subset is empty, any index
// is out of range, or the total probability mass on A is ≤ tol.
func (d Distribution) Restrict(subset []int, tol float64) (Distribution, error) {
	if err := d.Validate(tol); err != nil {
		return nil, err
	}
	A, err := validateStateSubset(subset, len(d))
	if err != nil {
		return nil, err
	}
	mass := subsetProbabilityMass(d, A)
	if mass <= tol {
		return nil, fmt.Errorf("%w: zero mass on subset", ErrInvalidDistribution)
	}
	out := make(Distribution, len(A))
	for i, s := range A {
		out[i] = clampProbabilityEntry(d[s]) / mass
	}
	return out, nil
}

// Project aggregates d onto the macro-blocks B₀, …, Bₘ₋₁ of partition p:
//
//	π̂[j] = Σ_{s∈Bⱼ} π[s] = (π·V)[j]
//
// Returns ErrInvalidPartition if partition is invalid for len(d) states, or
// ErrInvalidDistribution if d fails validation within tol.
func (d Distribution) Project(p Partition, tol float64) (Distribution, error) {
	if err := p.Validate(len(d)); err != nil {
		return nil, err
	}
	if err := d.Validate(tol); err != nil {
		return nil, err
	}
	out := make(Distribution, len(p))
	for j, block := range p {
		out[j] = subsetProbabilityMass(d, block)
	}
	return out, nil
}

// ValidateDistribution is the top-level form of Distribution.Validate.
func ValidateDistribution(pi Distribution, tol float64) error {
	if pi == nil {
		return fmt.Errorf("%w: nil distribution", ErrInvalidDistribution)
	}
	return pi.Validate(tol)
}

// RestrictDistribution is the top-level form of Distribution.Restrict.
func RestrictDistribution(pi Distribution, subset []int, tol float64) (Distribution, error) {
	if pi == nil {
		return nil, fmt.Errorf("%w: nil distribution", ErrInvalidDistribution)
	}
	return pi.Restrict(subset, tol)
}

// ProjectDistribution is the top-level form of Distribution.Project.
func ProjectDistribution(pi Distribution, p Partition, tol float64) (Distribution, error) {
	if pi == nil {
		return nil, fmt.Errorf("%w: nil distribution", ErrInvalidDistribution)
	}
	return pi.Project(p, tol)
}

// ── internal helpers ──────────────────────────────────────────────────────────

// clampDistributionNegatives zeroes out entries v ∈ [-tol, 0) in place.
func clampDistributionNegatives(d Distribution, tol float64) {
	for i, v := range d {
		if v < 0 && math.Abs(v) <= tol {
			d[i] = 0
		}
	}
}

// validateStateSubset canonicalizes subset and verifies that it is non-empty
// and every state index s lies in [0, n).
func validateStateSubset(subset []int, n int) ([]int, error) {
	A := CanonicalSubset(subset)
	if len(A) == 0 {
		return nil, fmt.Errorf("%w: empty subset", ErrInvalidDistribution)
	}
	for _, s := range A {
		if s < 0 || s >= n {
			return nil, fmt.Errorf("%w: subset index %d out of range [0,%d)", ErrInvalidDistribution, s, n)
		}
	}
	return A, nil
}

// subsetProbabilityMass returns Σ_{s∈indices} max(d[s], 0), the total
// probability mass of distribution d on the given state indices.
func subsetProbabilityMass(d Distribution, indices []int) float64 {
	mass := 0.0
	for _, s := range indices {
		mass += clampProbabilityEntry(d[s])
	}
	return mass
}

// clampProbabilityEntry clamps a near-zero negative probability entry to 0.
func clampProbabilityEntry(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
