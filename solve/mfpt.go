package solve

import (
	"fmt"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// MFPT computes the n×n matrix M of mean first-passage times for an irreducible
// Markov chain k using Kemeny & Snell's Theorem 4.4.7:
//
//	M[i, j] = (Z[j, j] - Z[i, j]) / π[j]   for i ≠ j
//	M[i, i] = 0
//
// Returns [markov.ErrNotStochastic] if k is nil or tol < 0.
// Returns [markov.ErrSingularOperator] if k is reducible or has transient states
// (some π[j] ≤ tol).
func MFPT(k *markov.Kernel, tol float64) (*mat.Dense, error) {
	pi, err := Stationary(k, tol)
	if err != nil {
		return nil, err
	}
	if err := verifyStrictlyPositiveSupport(pi, tol); err != nil {
		return nil, err
	}
	Z, err := FundamentalMatrix(k, tol)
	if err != nil {
		return nil, err
	}
	return computeMFPTFromFundamental(Z, pi, tol), nil
}

// MFPTToTarget computes the vector m of mean first-passage times from every
// state i ∈ {0, …, n-1} to the given target state:
//
//	m[target] = 0
//	m[i]      = 1 + Σ_{j ≠ target} P[i, j]·m[j]   for i ≠ target
//
// Equivalently, m_T = (I - P[T, T])⁻¹·1 where T = S \ {target}.
// Returns [markov.ErrSingularOperator] if target is out of range or unreachable
// from some state in S.
func MFPTToTarget(k *markov.Kernel, target int, tol float64) ([]float64, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	n := k.NumStates()
	if target < 0 || target >= n {
		return nil, fmt.Errorf("%w: target state %d out of range [0,%d)", markov.ErrSingularOperator, target, n)
	}
	if n == 1 {
		return []float64{0}, nil
	}

	T := complementOfTargetState(n, target)
	N, err := AbsorbingFundamental(k, T, tol)
	if err != nil {
		return nil, err
	}

	out := make([]float64, n)
	for idx, s := range T {
		out[s] = expectedStepsFromRow(N.RawRowView(idx))
	}
	return out, nil
}

// MFPTDelta computes the change in mean first-passage time from state `from` to
// `target` when row `from` of k is perturbed toward state `to` by convex
// interpolation parameter delta ∈ [0, 1]:
//
//	P̃[from, ·] = (1 - delta)·P[from, ·] + delta·e_toᵀ
//	Δm         = m[from, target](P̃) - m[from, target](P)
//
// Returns [markov.ErrNotStochastic] if k is nil, tol < 0, or delta ∉ [0, 1].
// Returns [markov.ErrSingularOperator] if state indices are out of range or
// target is unreachable before or after the perturbation.
func MFPTDelta(k *markov.Kernel, from, to, target int, delta, tol float64) (float64, error) {
	if err := validatePerturbationInputs(k, from, to, target, delta, tol); err != nil {
		return 0, err
	}
	delta = clampPerturbationDelta(delta)

	baseMFPT, err := MFPTToTarget(k, target, tol)
	if err != nil {
		return 0, err
	}
	if delta == 0 {
		return 0, nil
	}

	perturbed, err := convexRowPerturbation(k, from, to, delta, tol)
	if err != nil {
		return 0, err
	}
	newMFPT, err := MFPTToTarget(perturbed, target, tol)
	if err != nil {
		return 0, err
	}
	return newMFPT[from] - baseMFPT[from], nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// verifyStrictlyPositiveSupport checks that π_j > tol for all states j (i.e.,
// the chain is irreducible with no transient states).
func verifyStrictlyPositiveSupport(pi markov.Distribution, tol float64) error {
	for j, p := range pi {
		if p <= tol {
			return fmt.Errorf("%w: state %d has zero stationary mass (%g); chain is not irreducible", markov.ErrSingularOperator, j, p)
		}
	}
	return nil
}

// computeMFPTFromFundamental evaluates Kemeny & Snell Theorem 4.4.7:
//
//	M[i, j] = (Z[j, j] - Z[i, j]) / π[j]   for i ≠ j,   M[i, i] = 0
func computeMFPTFromFundamental(Z *mat.Dense, pi markov.Distribution, tol float64) *mat.Dense {
	n := len(pi)
	M := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			mij := (Z.At(j, j) - Z.At(i, j)) / pi[j]
			if mij < 0 && mij >= -tol {
				mij = 0
			}
			M.Set(i, j, mij)
		}
	}
	return M
}

// complementOfTargetState returns S \ {target} = {0, …, n-1} \ {target}.
func complementOfTargetState(n, target int) []int {
	T := make([]int, 0, n-1)
	for i := 0; i < n; i++ {
		if i != target {
			T = append(T, i)
		}
	}
	return T
}

// expectedStepsFromRow returns (N·1)_i = Σ_j N[i, j], the expected total visits
// to transient states before absorption when starting from transient row i.
func expectedStepsFromRow(row []float64) float64 {
	var sum float64
	for _, v := range row {
		sum += v
	}
	return sum
}

// validatePerturbationInputs checks preconditions for MFPTDelta.
func validatePerturbationInputs(k *markov.Kernel, from, to, target int, delta, tol float64) error {
	if k == nil || k.P == nil {
		return fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	if delta < -tol || delta > 1+tol {
		return fmt.Errorf("%w: perturbation delta %g outside [0, 1]", markov.ErrNotStochastic, delta)
	}
	n := k.NumStates()
	if from < 0 || from >= n || to < 0 || to >= n || target < 0 || target >= n {
		return fmt.Errorf("%w: state index out of range [0,%d)", markov.ErrSingularOperator, n)
	}
	return nil
}

// clampPerturbationDelta clamps perturbation weight delta ∈ [-tol, 1+tol] onto [0, 1].
func clampPerturbationDelta(delta float64) float64 {
	if delta < 0 {
		return 0
	}
	if delta > 1 {
		return 1
	}
	return delta
}

// convexRowPerturbation returns a cloned kernel P̃ whose `from` row is replaced
// by the convex combination P̃[from, ·] = (1 - delta)·P[from, ·] + delta·e_toᵀ.
func convexRowPerturbation(k *markov.Kernel, from, to int, delta, tol float64) (*markov.Kernel, error) {
	perturbed := k.Clone()
	row := perturbed.P.RawRowView(from)
	for j := range row {
		row[j] *= (1.0 - delta)
	}
	row[to] += delta
	if err := perturbed.Validate(tol); err != nil {
		return nil, err
	}
	return perturbed, nil
}
