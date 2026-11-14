package solve

import (
	"fmt"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// FundamentalMatrix computes Kemeny & Snell's ergodic fundamental matrix:
//
//	Z = (I - P + 1π)⁻¹
//
// where π is the unique stationary distribution of the unichain k and 1π is the
// n×n matrix with every row equal to π.
//
// Returns [markov.ErrNotStochastic] if k is nil or tol < 0.
// Returns [markov.ErrSingularOperator] if k is not unichain or (I - P + 1π) is singular.
func FundamentalMatrix(k *markov.Kernel, tol float64) (*mat.Dense, error) {
	pi, err := Stationary(k, tol)
	if err != nil {
		return nil, err
	}
	n := k.NumStates()
	A, I := buildErgodicFundamentalSystem(k.P, pi, n)

	var Z mat.Dense
	if err := Z.Solve(A, I); err != nil {
		return nil, fmt.Errorf("%w: fundamental matrix solve failed: %v", markov.ErrSingularOperator, err)
	}
	return &Z, nil
}

// AbsorbingFundamental computes the absorbing fundamental matrix on the given
// transient state subset T:
//
//	N = (I - Q)⁻¹,   Q = P[T, T]
//
// Entry N[i, j] gives the expected number of visits to transient state T[j]
// before absorption when starting in transient state T[i].
//
// Returns [markov.ErrNotStochastic] if k is nil or tol < 0.
// Returns [markov.ErrSingularOperator] if transient is empty, out of bounds,
// covers the entire state space, or contains a closed recurrent class (so I - Q
// is singular or has negative entries).
func AbsorbingFundamental(k *markov.Kernel, transient []int, tol float64) (*mat.Dense, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	T, err := validateTransientSubset(transient, k.NumStates())
	if err != nil {
		return nil, err
	}
	if err := verifyCanReachAbsorbing(k, T, tol); err != nil {
		return nil, err
	}

	IminusQ, I := buildAbsorbingSystem(k.P, T)
	var N mat.Dense
	if err := N.Solve(IminusQ, I); err != nil {
		return nil, fmt.Errorf("%w: absorbing fundamental solve failed: %v", markov.ErrSingularOperator, err)
	}
	if err := clampNonNegativeInverse(&N, tol); err != nil {
		return nil, err
	}
	return &N, nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// buildErgodicFundamentalSystem constructs A = (I - P + 1π) and the n×n
// identity matrix I for solving Kemeny & Snell's ergodic fundamental matrix Z.
func buildErgodicFundamentalSystem(P *mat.Dense, pi markov.Distribution, n int) (*mat.Dense, *mat.Dense) {
	A := mat.NewDense(n, n, nil)
	I := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		I.Set(i, i, 1.0)
		for j := 0; j < n; j++ {
			v := -P.At(i, j) + pi[j]
			if i == j {
				v += 1.0
			}
			A.Set(i, j, v)
		}
	}
	return A, I
}

// validateTransientSubset canonicalizes transient and checks that T is a
// non-empty proper subset of {0, …, n-1}.
func validateTransientSubset(transient []int, n int) ([]int, error) {
	T := markov.CanonicalSubset(transient)
	if len(T) == 0 || len(T) >= n {
		return nil, fmt.Errorf("%w: transient subset must be non-empty proper subset of [0,%d)", markov.ErrSingularOperator, n)
	}
	for _, s := range T {
		if s < 0 || s >= n {
			return nil, fmt.Errorf("%w: transient state %d out of range [0,%d)", markov.ErrSingularOperator, s, n)
		}
	}
	return T, nil
}

// verifyCanReachAbsorbing verifies graph-theoretically at threshold tol that
// every state in T has a directed path of transitions P[u,v] > tol to S \ T.
// By Berman & Plemmons (Ch. 6), this ensures ρ(Q) < 1 and prevents closed
// recurrent traps with leaky transitions ≤ tol from producing O(1/tol) inverses.
func verifyCanReachAbsorbing(k *markov.Kernel, T []int, tol float64) error {
	n := k.NumStates()
	inT := make([]bool, n)
	for _, s := range T {
		inT[s] = true
	}
	canExit := make([]bool, n)
	var queue []int
	for s := 0; s < n; s++ {
		if !inT[s] {
			canExit[s] = true
			queue = append(queue, s)
		}
	}
	for head := 0; head < len(queue); head++ {
		queue = enqueuePredecessorsInT(k, queue[head], inT, canExit, queue, tol)
	}
	for _, s := range T {
		if !canExit[s] {
			return fmt.Errorf("%w: transient state %d cannot reach absorbing complement", markov.ErrSingularOperator, s)
		}
	}
	return nil
}

// enqueuePredecessorsInT marks and enqueues any unvisited transient state u ∈ T
// that transitions into state v with probability P[u, v] > tol.
func enqueuePredecessorsInT(k *markov.Kernel, v int, inT, canExit []bool, queue []int, tol float64) []int {
	n := k.NumStates()
	for u := 0; u < n; u++ {
		if inT[u] && !canExit[u] && k.P.At(u, v) > tol {
			canExit[u] = true
			queue = append(queue, u)
		}
	}
	return queue
}

// buildAbsorbingSystem constructs (I - Q) where Q = P[T, T] and the m×m
// identity matrix I.
func buildAbsorbingSystem(P *mat.Dense, T []int) (*mat.Dense, *mat.Dense) {
	m := len(T)
	IminusQ := mat.NewDense(m, m, nil)
	I := mat.NewDense(m, m, nil)
	for i, si := range T {
		I.Set(i, i, 1.0)
		for j, sj := range T {
			v := -P.At(si, sj)
			if i == j {
				v += 1.0
			}
			IminusQ.Set(i, j, v)
		}
	}
	return IminusQ, I
}

// clampNonNegativeInverse verifies the M-matrix property N = (I - Q)⁻¹ ≥ -tol
// entrywise and clamps near-zero negative entries in [-tol, 0) to 0.
func clampNonNegativeInverse(N *mat.Dense, tol float64) error {
	m, _ := N.Dims()
	for i := 0; i < m; i++ {
		row := N.RawRowView(i)
		for j, v := range row {
			if v < -tol {
				return fmt.Errorf("%w: transient subset contains a closed recurrent trap (N[%d,%d]=%g < 0)",
					markov.ErrSingularOperator, i, j, v)
			}
			if v < 0 {
				row[j] = 0
			}
		}
	}
	return nil
}
