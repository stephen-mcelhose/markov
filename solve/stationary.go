package solve

import (
	"fmt"

	"github.com/stephen-mcelhose/markov"
)

// Stationary computes the unique stationary distribution π of a unichain Markov
// kernel k:
//
//	πP = π,   Σ π[i] = 1,   π[i] ≥ 0
//
// Uses the Grassmann-Taksar-Heyman (GTH) direct state-reduction algorithm on
// the unique closed recurrent communicating class, placing zero mass on all
// transient states.
//
// Returns [markov.ErrNotStochastic] if k is nil or tol < 0.
// Returns [markov.ErrSingularOperator] if k has multiple closed recurrent classes.
func Stationary(k *markov.Kernel, tol float64) (markov.Distribution, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	classes, err := ClassifyStates(k, tol)
	if err != nil {
		return nil, err
	}
	if len(classes.Recurrent) != 1 {
		return nil, fmt.Errorf("%w: chain has %d recurrent classes, want 1 (unichain)", markov.ErrSingularOperator, len(classes.Recurrent))
	}

	rec := classes.Recurrent[0]
	n := k.NumStates()
	if len(rec) == 1 {
		pi := make(markov.Distribution, n)
		pi[rec[0]] = 1.0
		return pi, nil
	}

	A := extractRecurrentSubmatrix(k, rec)
	if err := gthEliminateStates(A, rec); err != nil {
		return nil, err
	}
	return gthForwardSubstitute(A, rec, n)
}

// MustStationary is like Stationary but panics on error.
func MustStationary(k *markov.Kernel, tol float64) markov.Distribution {
	pi, err := Stationary(k, tol)
	if err != nil {
		panic(err)
	}
	return pi
}

// ── internal helpers ──────────────────────────────────────────────────────────

// extractRecurrentSubmatrix copies the m×m irreducible transition submatrix
// P[rec, rec] onto the closed recurrent communicating class rec.
func extractRecurrentSubmatrix(k *markov.Kernel, rec []int) [][]float64 {
	m := len(rec)
	A := make([][]float64, m)
	for i, si := range rec {
		A[i] = make([]float64, m)
		for j, sj := range rec {
			v := k.P.At(si, sj)
			if v < 0 {
				v = 0
			}
			A[i][j] = v
		}
	}
	return A
}

// gthEliminateStates performs the Grassmann-Taksar-Heyman (1985) subtraction-free
// Gaussian elimination on the m×m recurrent submatrix A:
//
//	For r = m-1 down to 1:
//	  S_r     = Σ_{j=0}^{r-1} A[r, j]
//	  A[i, r] = A[i, r] / S_r                    for i < r
//	  A[i, j] = A[i, j] + A[i, r] · A[r, j]      for i < r, j < r (i ≠ j)
//
// Because S_r replaces (1 - A[r, r]) using only off-diagonal additions, GTH
// never subtracts nearly equal floating-point numbers.
func gthEliminateStates(A [][]float64, rec []int) error {
	for r := len(rec) - 1; r >= 1; r-- {
		S := rowPrefixSum(A[r], r)
		if S <= 0 {
			return fmt.Errorf("%w: zero exit sum at state %d during GTH reduction", markov.ErrSingularOperator, rec[r])
		}
		gthPivotUpdate(A, r, S)
	}
	return nil
}

// rowPrefixSum returns Σ_{j=0}^{limit-1} row[j].
func rowPrefixSum(row []float64, limit int) float64 {
	var sum float64
	for j := 0; j < limit; j++ {
		sum += row[j]
	}
	return sum
}

// gthPivotUpdate eliminates state r into states 0..r-1 using pivot sum S:
//
//	A[i, r] ← A[i, r] / S
//	A[i, j] ← A[i, j] + A[i, r] · A[r, j]   (for i, j < r, i ≠ j)
func gthPivotUpdate(A [][]float64, r int, S float64) {
	for i := 0; i < r; i++ {
		A[i][r] /= S
		air := A[i][r]
		for j := 0; j < r; j++ {
			if i != j {
				A[i][j] += air * A[r][j]
			}
		}
	}
}

// gthForwardSubstitute recovers the unnormalized stationary weights u[0..m-1]
// via u[0] = 1 and u[j] = Σ_{i=0}^{j-1} u[i]·A[i, j], then normalizes u into a
// full n-state Distribution supported on rec.
func gthForwardSubstitute(A [][]float64, rec []int, n int) (markov.Distribution, error) {
	m := len(rec)
	u := make([]float64, m)
	u[0] = 1.0
	total := 1.0
	for j := 1; j < m; j++ {
		var sum float64
		for i := 0; i < j; i++ {
			sum += u[i] * A[i][j]
		}
		u[j] = sum
		total += sum
	}
	if total <= 0 {
		return nil, fmt.Errorf("%w: non-positive stationary mass", markov.ErrSingularOperator)
	}

	pi := make(markov.Distribution, n)
	for i, s := range rec {
		pi[s] = u[i] / total
	}
	return pi, nil
}
