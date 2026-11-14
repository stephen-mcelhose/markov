package solve

import (
	"fmt"
	"math"

	"github.com/stephen-mcelhose/markov"
)

// EntropyRate computes the Shannon entropy rate H(P) in bits per step of a
// unichain Markov kernel k:
//
//	H(P) = -Σ_i π[i] Σ_j P[i, j] log₂ P[i, j]   (with 0 log₂ 0 := 0)
//
// Returns [markov.ErrNotStochastic] if k is nil or tol < 0.
// Returns [markov.ErrSingularOperator] if k is not unichain.
func EntropyRate(k *markov.Kernel, tol float64) (float64, error) {
	pi, err := Stationary(k, tol)
	if err != nil {
		return 0, err
	}
	var h float64
	for i, weight := range pi {
		if weight > 0 {
			h += weight * rowShannonEntropy(k.P.RawRowView(i))
		}
	}
	return clampNearZeroEntropy(h, tol), nil
}

// KLDivergenceRate computes the Kullback-Leibler divergence rate D_KL(P ‖ Q)
// in bits per step between two Markov kernels p and q of the same dimension:
//
//	D_KL(P ‖ Q) = Σ_i π_P[i] Σ_j P[i, j] log₂(P[i, j] / Q[i, j])
//
// If P[i, j] > tol and Q[i, j] ≤ tol for a state i in the stationary support
// of P (π_P[i] > tol), returns +Inf (math.Inf(1)) without error.
//
// Returns [markov.ErrNotStochastic] if p or q is nil or tol < 0.
// Returns [markov.ErrSingularOperator] if dimensions mismatch or p is not unichain.
func KLDivergenceRate(p, q *markov.Kernel, tol float64) (float64, error) {
	if err := validateKLPair(p, q, tol); err != nil {
		return 0, err
	}
	piP, err := Stationary(p, tol)
	if err != nil {
		return 0, err
	}

	var dkl float64
	for i, weight := range piP {
		if weight <= tol {
			continue
		}
		rowKL := rowKLDivergence(p.P.RawRowView(i), q.P.RawRowView(i), tol)
		if math.IsInf(rowKL, 1) {
			return math.Inf(1), nil
		}
		dkl += weight * rowKL
	}
	return clampNearZeroEntropy(dkl, tol), nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// rowShannonEntropy computes the one-step conditional Shannon entropy (in bits)
// of transition row P[i, ·]:
//
//	H(X₁ | X₀ = i) = -Σ_j P[i, j] log₂ P[i, j]   (with 0 log₂ 0 := 0)
func rowShannonEntropy(row []float64) float64 {
	var h float64
	for _, pij := range row {
		if pij > 0 {
			h -= pij * math.Log2(pij)
		}
	}
	return h
}

// rowKLDivergence computes the conditional Kullback-Leibler divergence (in bits)
// between row P[i, ·] and row Q[i, ·]:
//
//	D_KL(P[i, ·] ‖ Q[i, ·]) = Σ_j P[i, j] log₂(P[i, j] / Q[i, j])
//
// Returns +Inf if P[i, j] > tol and Q[i, j] ≤ tol (failure of absolute continuity).
func rowKLDivergence(pRow, qRow []float64, tol float64) float64 {
	var kl float64
	for j, pij := range pRow {
		if pij <= tol {
			continue
		}
		qij := qRow[j]
		if qij <= tol {
			return math.Inf(1)
		}
		kl += pij * math.Log2(pij/qij)
	}
	return kl
}

// validateKLPair checks that p and q are non-nil kernels of equal dimension
// and tol ≥ 0.
func validateKLPair(p, q *markov.Kernel, tol float64) error {
	if p == nil || p.P == nil {
		return fmt.Errorf("%w: nil kernel p", markov.ErrNotStochastic)
	}
	if q == nil || q.P == nil {
		return fmt.Errorf("%w: nil kernel q", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	if p.NumStates() != q.NumStates() {
		return fmt.Errorf("%w: dimension mismatch: p has %d states, q has %d", markov.ErrSingularOperator, p.NumStates(), q.NumStates())
	}
	return nil
}

// clampNearZeroEntropy clamps a tiny negative entropy value in [-tol, 0) to 0.
func clampNearZeroEntropy(v, tol float64) float64 {
	if v < 0 && math.Abs(v) <= tol {
		return 0
	}
	return v
}
