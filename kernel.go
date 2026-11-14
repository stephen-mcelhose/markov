package markov

import (
	"fmt"
	"math"
	"slices"
	"sort"

	"gonum.org/v1/gonum/mat"
)

// DefaultTolerance is the default floating-point tolerance used when validating
// stochasticity and probability conservation.
const DefaultTolerance = 1e-9

// Kernel is a validated square row-stochastic Markov transition matrix.
//
// Every entry P[i,j] ≥ 0 and every row sums to 1 within the tolerance used
// at construction time. StateNames labels each state; it is always non-nil and
// has length NumStates().
type Kernel struct {
	P          *mat.Dense
	StateNames []string
}

// NewKernel constructs a validated Kernel with default tolerance 1e-9.
// Near-zero negative entries (|v| ≤ tol) are clamped to 0 after validation.
// Returns ErrNotStochastic if P is nil, non-square, or fails stochasticity.
// Returns an error if len(names) > 0 and len(names) ≠ n.
func NewKernel(P *mat.Dense, names []string) (*Kernel, error) {
	return NewKernelWithTol(P, names, DefaultTolerance)
}

// NewKernelWithTol constructs a validated Kernel with the given tolerance.
func NewKernelWithTol(P *mat.Dense, names []string, tol float64) (*Kernel, error) {
	if P == nil {
		return nil, fmt.Errorf("%w: nil matrix", ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", ErrNotStochastic, tol)
	}
	r, c := P.Dims()
	if r <= 0 || r != c {
		return nil, fmt.Errorf("%w: matrix must be non-empty square, got %dx%d", ErrNotStochastic, r, c)
	}
	if len(names) > 0 && len(names) != r {
		return nil, fmt.Errorf("%w: %d names for %d states", ErrNotStochastic, len(names), r)
	}
	k := &Kernel{
		P:          cloneTransitionMatrix(P),
		StateNames: slices.Clone(names),
	}
	if len(k.StateNames) == 0 {
		k.StateNames = defaultStateNames("s", r)
	}
	if err := k.Validate(tol); err != nil {
		return nil, err
	}
	clampNearZeroNegatives(k.P, tol)
	return k, nil
}

// MustNewKernel is like NewKernel but panics on error.
// Use when preconditions are guaranteed by construction.
func MustNewKernel(P *mat.Dense, names []string) *Kernel {
	k, err := NewKernel(P, names)
	if err != nil {
		panic(err)
	}
	return k
}

// NumStates returns the number of states n.
func (k *Kernel) NumStates() int {
	if k == nil || k.P == nil {
		return 0
	}
	r, _ := k.P.Dims()
	return r
}

// Clone returns a deep copy of k.
func (k *Kernel) Clone() *Kernel {
	if k == nil {
		return nil
	}
	return &Kernel{
		P:          cloneTransitionMatrix(k.P),
		StateNames: slices.Clone(k.StateNames),
	}
}

// Submatrix returns the submatrix of k.P indexed by rows and cols.
// Output[i][j] = P[rows[i]][cols[j]].
func (k *Kernel) Submatrix(rows, cols []int) *mat.Dense {
	if k == nil || k.P == nil {
		return nil
	}
	return Submatrix(k.P, rows, cols)
}

// NormalizeRows rescales each row of k.P to sum to 1, clamping near-zero
// negative entries to 0 first. Returns an error if any row sum is ≤ tol.
// Mutates k in place and re-validates.
func (k *Kernel) NormalizeRows(tol float64) error {
	if k == nil || k.P == nil {
		return fmt.Errorf("%w: nil kernel", ErrNotStochastic)
	}
	if err := NormalizeRows(k.P, tol); err != nil {
		return err
	}
	return k.Validate(math.Max(tol, DefaultTolerance))
}

// Validate reports whether every row of k.P is non-negative and sums to 1
// within tol. It is called automatically by NewKernelWithTol and NormalizeRows.
func (k *Kernel) Validate(tol float64) error {
	if k == nil || k.P == nil {
		return fmt.Errorf("%w: nil kernel", ErrNotStochastic)
	}
	if tol < 0 {
		return fmt.Errorf("%w: negative tolerance %g", ErrNotStochastic, tol)
	}
	r, _ := k.P.Dims()
	for i := 0; i < r; i++ {
		if err := validateStochasticRow(k.P.RawRowView(i), i, tol); err != nil {
			return err
		}
	}
	return nil
}

// validateStochasticRow checks the right-stochastic row invariants on row i:
//
//	P[i, j] ≥ -tol   for all j,   and   |Σ_j P[i, j] - 1| ≤ tol
func validateStochasticRow(row []float64, rowIdx int, tol float64) error {
	sum := 0.0
	for j, v := range row {
		if v < -tol {
			return fmt.Errorf("%w: negative entry at (%d,%d): %g", ErrNotStochastic, rowIdx, j, v)
		}
		sum += v
	}
	if math.Abs(sum-1) > tol {
		return fmt.Errorf("%w: row %d sums to %g", ErrNotStochastic, rowIdx, sum)
	}
	return nil
}

// clampNearZeroNegatives zeroes out entries v ∈ [-tol, 0) across all rows of P
// after stochasticity validation so small IEEE-754 solver residuals do not
// leave negative probabilities in the kernel.
func clampNearZeroNegatives(P *mat.Dense, tol float64) {
	r, _ := P.Dims()
	for i := 0; i < r; i++ {
		row := P.RawRowView(i)
		for j, v := range row {
			if v < 0 && math.Abs(v) <= tol {
				row[j] = 0
			}
		}
	}
}

// Submatrix returns the submatrix of P indexed by rows and cols.
// This is the top-level form of Kernel.Submatrix.
func Submatrix(P *mat.Dense, rows, cols []int) *mat.Dense {
	if P == nil {
		return nil
	}
	out := mat.NewDense(len(rows), len(cols), nil)
	for i, r := range rows {
		for j, c := range cols {
			out.Set(i, j, P.At(r, c))
		}
	}
	return out
}

// NormalizeRows rescales each row of P in place to sum to 1, clamping near-zero
// negative entries (|v| ≤ tol) to 0 first. Returns ErrNotStochastic if P is nil,
// tol < 0, contains an entry < -tol, or has any row sum ≤ tol.
// This is the top-level matrix form of Kernel.NormalizeRows.
func NormalizeRows(P *mat.Dense, tol float64) error {
	if P == nil || P.IsEmpty() {
		return fmt.Errorf("%w: nil or empty matrix", ErrNotStochastic)
	}
	if tol < 0 {
		return fmt.Errorf("%w: negative tolerance %g", ErrNotStochastic, tol)
	}
	r, _ := P.Dims()
	for i := 0; i < r; i++ {
		if err := normalizeRow(P.RawRowView(i), i, tol); err != nil {
			return err
		}
	}
	return nil
}

// normalizeRow clamps near-zero negatives in row within tol, verifies the row
// sum exceeds tol, and rescales each entry in place so the row sums to 1.
func normalizeRow(row []float64, rowIdx int, tol float64) error {
	sum, err := clampAndSumRow(row, rowIdx, tol)
	if err != nil {
		return err
	}
	for j := range row {
		row[j] /= sum
	}
	return nil
}

// clampAndSumRow validates that every entry in row is ≥ -tol, clamps near-zero
// negative entries in [-tol, 0) to 0 in place, and returns the resulting row sum.
// Returns ErrNotStochastic if any entry is < -tol or if the row sum is ≤ tol.
func clampAndSumRow(row []float64, rowIdx int, tol float64) (float64, error) {
	sum := 0.0
	for j, v := range row {
		if v < -tol {
			return 0, fmt.Errorf("%w: negative entry at (%d,%d): %g", ErrNotStochastic, rowIdx, j, v)
		}
		if v < 0 {
			v = 0
			row[j] = 0
		}
		sum += v
	}
	if sum <= tol {
		return 0, fmt.Errorf("%w: row %d has near-zero sum %g", ErrNotStochastic, rowIdx, sum)
	}
	return sum, nil
}

// CanonicalSubset returns a sorted, deduplicated copy of subset.
func CanonicalSubset(subset []int) []int {
	return canonicalizeStateIndices(subset)
}

// NearlyEqual reports whether |a - b| ≤ tol.
func NearlyEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

// ── internal helpers ──────────────────────────────────────────────────────────

// cloneTransitionMatrix returns a deep copy of matrix p.
func cloneTransitionMatrix(p *mat.Dense) *mat.Dense {
	r, c := p.Dims()
	out := mat.NewDense(r, c, nil)
	out.Copy(p)
	return out
}

// defaultStateNames generates n state labels with the given prefix ("s0", "s1", ...).
func defaultStateNames(prefix string, n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return names
}

// canonicalizeStateIndices returns a sorted, deduplicated copy of stateIndices,
// or nil if stateIndices is empty.
func canonicalizeStateIndices(stateIndices []int) []int {
	if len(stateIndices) == 0 {
		return nil
	}
	cp := slices.Clone(stateIndices)
	sort.Ints(cp)
	out := cp[:1]
	for _, v := range cp[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}
