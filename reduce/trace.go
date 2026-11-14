package reduce

import (
	"fmt"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// Trace computes the induced kernel on subset A by integrating out excursions
// through the complement B = S\A.
//
// If P is partitioned as
//
//	P = [ a  b ]   rows/cols A
//	    [ d  c ]   rows/cols B
//
// then the trace kernel is:
//
//	P_A = a + b(I - c)⁻¹d
//
// Returns [markov.ErrSingularOperator] if (I-c) is singular or B contains a
// closed recurrent trap so excursions cannot return to A.
// Returns [markov.ErrNotStochastic] if k is nil, tol < 0, or the resulting P_A
// fails row-stochasticity validation.
func Trace(k *markov.Kernel, subset []int, tol float64) (*markov.Kernel, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	n := k.NumStates()
	A, err := validateSubset(n, subset)
	if err != nil {
		return nil, err
	}
	Ac := complementStateIndices(n, A)
	a := k.Submatrix(A, A)
	names := stateNamesForSubset(k, A)
	if len(Ac) == 0 {
		return markov.NewKernelWithTol(a, names, tol)
	}
	if err := checkExcursionCanReturn(k, A, Ac, tol); err != nil {
		return nil, err
	}
	b := k.Submatrix(A, Ac)
	c := k.Submatrix(Ac, Ac)
	d := k.Submatrix(Ac, A)
	result, err := solveSchurComplement(a, b, c, d)
	if err != nil {
		return nil, err
	}
	tr, err := markov.NewKernelWithTol(result, names, tol)
	if err != nil {
		return nil, err
	}
	if err := tr.NormalizeRows(tol); err != nil {
		return nil, err
	}
	return tr, nil
}

// MustTrace is like Trace but panics on error.
func MustTrace(k *markov.Kernel, subset []int, tol float64) *markov.Kernel {
	tr, err := Trace(k, subset, tol)
	if err != nil {
		panic(err)
	}
	return tr
}

// IsTraceOf reports whether k matches Trace(parent, subset) within tol:
//
//	‖k.P - Trace(parent, subset).P‖∞ ≤ tol
func IsTraceOf(k, parent *markov.Kernel, subset []int, tol float64) (bool, error) {
	if k == nil || k.P == nil {
		return false, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if parent == nil || parent.P == nil {
		return false, fmt.Errorf("%w: nil parent kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return false, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	tr, err := Trace(parent, subset, tol)
	if err != nil {
		return false, err
	}
	return matricesNearlyEqual(k.P, tr.P, tol), nil
}

// TraceReward computes the expected cumulative reward accumulated during
// excursions through B = S\A, given per-state rewards r:
//
//	r_A[i] = r_a[i] + (b(I-c)⁻¹r_c)[i]   for i ∈ A
//
// where r_a = rewards[A], r_c = rewards[B].
func TraceReward(k *markov.Kernel, rewards []float64, subset []int, tol float64) ([]float64, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	n := k.NumStates()
	if len(rewards) != n {
		return nil, fmt.Errorf("%w: rewards length %d does not match kernel state count %d", markov.ErrInvalidDistribution, len(rewards), n)
	}
	A, err := validateSubset(n, subset)
	if err != nil {
		return nil, err
	}
	Ac := complementStateIndices(n, A)
	ra := restrictRewardVector(rewards, A)
	if len(Ac) == 0 {
		return ra, nil
	}
	if err := checkExcursionCanReturn(k, A, Ac, tol); err != nil {
		return nil, err
	}
	rc := restrictRewardVector(rewards, Ac)
	raVec := mat.NewDense(len(A), 1, ra)
	rcVec := mat.NewDense(len(Ac), 1, rc)
	resVec, err := solveSchurComplement(raVec, k.Submatrix(A, Ac), k.Submatrix(Ac, Ac), rcVec)
	if err != nil {
		return nil, err
	}
	return columnVectorToSlice(resVec), nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// matricesNearlyEqual reports whether A and B have identical dimensions and
// satisfy |A[i, j] - B[i, j]| ≤ tol for every entry.
func matricesNearlyEqual(A, B *mat.Dense, tol float64) bool {
	r1, c1 := A.Dims()
	r2, c2 := B.Dims()
	if r1 != r2 || c1 != c2 {
		return false
	}
	for i := 0; i < r1; i++ {
		if !rowsNearlyEqual(A.RawRowView(i), B.RawRowView(i), tol) {
			return false
		}
	}
	return true
}

// rowsNearlyEqual reports whether two equal-length row slices agree within tol.
func rowsNearlyEqual(rowA, rowB []float64, tol float64) bool {
	for j, a := range rowA {
		if !markov.NearlyEqual(a, rowB[j], tol) {
			return false
		}
	}
	return true
}

// restrictRewardVector extracts the sub-vector [rewards[s] for s in subset].
func restrictRewardVector(rewards []float64, subset []int) []float64 {
	out := make([]float64, len(subset))
	for i, s := range subset {
		out[i] = rewards[s]
	}
	return out
}

// columnVectorToSlice extracts column 0 of an m×1 Dense matrix as a []float64 slice.
func columnVectorToSlice(col *mat.Dense) []float64 {
	m, _ := col.Dims()
	out := make([]float64, m)
	for i := range out {
		out[i] = col.At(i, 0)
	}
	return out
}

// validateSubset canonicalizes subset and bounds-checks every index.
func validateSubset(n int, subset []int) ([]int, error) {
	A := markov.CanonicalSubset(subset)
	if len(A) == 0 {
		return nil, fmt.Errorf("%w: empty subset", markov.ErrInvalidPartition)
	}
	for _, idx := range A {
		if idx < 0 || idx >= n {
			return nil, fmt.Errorf("%w: subset index %d out of range [0,%d)", markov.ErrInvalidPartition, idx, n)
		}
	}
	return A, nil
}

// complementStateIndices returns the sorted indices in {0..n-1} \ A, given sorted A.
func complementStateIndices(n int, A []int) []int {
	inA := make(map[int]bool, len(A))
	for _, v := range A {
		inA[v] = true
	}
	out := make([]int, 0, n-len(A))
	for i := 0; i < n; i++ {
		if !inA[i] {
			out = append(out, i)
		}
	}
	return out
}

// stateNamesForSubset returns the state names for the given (sorted) indices.
func stateNamesForSubset(k *markov.Kernel, A []int) []string {
	names := make([]string, len(A))
	for i, idx := range A {
		names[i] = k.StateNames[idx]
	}
	return names
}

// checkExcursionCanReturn verifies that every state in Ac has a directed path
// of transitions > tol back into A (so Ac contains no closed recurrent trap).
func checkExcursionCanReturn(k *markov.Kernel, A, Ac []int, tol float64) error {
	n := k.NumStates()
	canReachA := make([]bool, n)
	queue := make([]int, 0, n)
	for _, s := range A {
		canReachA[s] = true
		queue = append(queue, s)
	}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		queue = enqueueReachingStates(k, v, canReachA, queue, tol)
	}
	for _, s := range Ac {
		if !canReachA[s] {
			return fmt.Errorf("%w: complement state %d cannot reach watched subset", markov.ErrSingularOperator, s)
		}
	}
	return nil
}

// enqueueReachingStates marks and enqueues any unvisited state u that transitions
// into state v with probability P[u, v] > tol.
func enqueueReachingStates(k *markov.Kernel, v int, canReachA []bool, queue []int, tol float64) []int {
	for u := 0; u < k.NumStates(); u++ {
		if !canReachA[u] && k.P.At(u, v) > tol {
			canReachA[u] = true
			queue = append(queue, u)
		}
	}
	return queue
}

// solveSchurComplement computes the Schur complement folding a + b(I-c)⁻¹d.
// Returns markov.ErrSingularOperator if (I-c) is singular.
func solveSchurComplement(a, b, c, d *mat.Dense) (*mat.Dense, error) {
	IminusC := identityMinusMatrix(c)
	var X mat.Dense
	if err := X.Solve(IminusC, d); err != nil {
		return nil, fmt.Errorf("%w: %v", markov.ErrSingularOperator, err)
	}
	var bX mat.Dense
	bX.Mul(b, &X)
	var result mat.Dense
	result.Add(a, &bX)
	return &result, nil
}

// identityMinusMatrix returns (I - c) for a square m×m matrix c.
func identityMinusMatrix(c *mat.Dense) *mat.Dense {
	m, _ := c.Dims()
	IminusC := mat.NewDense(m, m, nil)
	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			v := -c.At(i, j)
			if i == j {
				v += 1.0
			}
			IminusC.Set(i, j, v)
		}
	}
	return IminusC
}
