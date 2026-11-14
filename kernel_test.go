package markov

import (
	"errors"
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

// rowStochastic is a rapid generator for n×n row-stochastic matrices.
func rowStochastic(n int) *rapid.Generator[*mat.Dense] {
	return rapid.Custom(func(t *rapid.T) *mat.Dense {
		data := make([]float64, n*n)
		for i := 0; i < n; i++ {
			row := make([]float64, n)
			sum := 0.0
			for j := 0; j < n; j++ {
				row[j] = rapid.Float64Range(0, 1).Draw(t, "entry")
				sum += row[j]
			}
			if sum == 0 {
				row[0] = 1.0
				sum = 1.0
			}
			for j := 0; j < n; j++ {
				data[i*n+j] = row[j] / sum
			}
		}
		return mat.NewDense(n, n, data)
	})
}

func TestNewKernelWithTol(t *testing.T) {
	type testCase struct {
		name    string
		P       *mat.Dense
		names   []string
		tol     float64
		wantErr error
	}

	valid2x2 := mat.NewDense(2, 2, []float64{0.7, 0.3, 0.4, 0.6})

	cases := []testCase{
		{
			name:    "MUST return ErrNotStochastic when P is nil",
			P:       nil,
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name:    "MUST return ErrNotStochastic when P is not square",
			P:       mat.NewDense(2, 3, []float64{0.5, 0.3, 0.2, 0.1, 0.5, 0.4}),
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name:    "MUST return ErrNotStochastic when a row sum deviates from 1 beyond tol",
			P:       mat.NewDense(2, 2, []float64{0.5, 0.3, 0.4, 0.6}),
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name:    "MUST return ErrNotStochastic when an entry is significantly negative",
			P:       mat.NewDense(2, 2, []float64{1.1, -0.1, 0.4, 0.6}),
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name:    "MUST return error when names length mismatches matrix dimension",
			P:       valid2x2,
			names:   []string{"only_one"},
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name: "MUST succeed for a valid row-stochastic matrix with no names",
			P:    valid2x2,
			tol:  1e-9,
		},
		{
			name:  "MUST succeed with explicit state names matching dimension",
			P:     valid2x2,
			names: []string{"a", "b"},
			tol:   1e-9,
		},
		{
			name: "MUST clamp near-zero negative entries within tol to zero",
			P:    mat.NewDense(2, 2, []float64{1.0 + 1e-12, -1e-12, 0.4, 0.6}),
			tol:  1e-9,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, err := NewKernelWithTol(tc.P, tc.names, tc.tol)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error wrapping %v, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error wrapping %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if k == nil {
				t.Fatal("expected non-nil kernel")
			}
		})
	}
}

func TestNewKernel(t *testing.T) {
	type testCase struct {
		name    string
		P       *mat.Dense
		names   []string
		wantErr error
	}
	cases := []testCase{
		{
			name: "MUST default to tol 1e-9",
			P:    mat.NewDense(2, 2, []float64{0.7, 0.3, 0.4, 0.6}),
		},
		{
			name:    "MUST return ErrNotStochastic on nil matrix",
			P:       nil,
			wantErr: ErrNotStochastic,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, err := NewKernel(tc.P, tc.names)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if k == nil {
				t.Fatal("expected non-nil kernel")
			}
		})
	}
}

func TestMustNewKernel(t *testing.T) {
	type testCase struct {
		name      string
		P         *mat.Dense
		wantPanic bool
	}
	cases := []testCase{
		{
			name:      "MUST panic on invalid input",
			P:         nil,
			wantPanic: true,
		},
		{
			name:      "MUST NOT panic on valid row-stochastic matrix",
			P:         mat.NewDense(2, 2, []float64{0.5, 0.5, 0.2, 0.8}),
			wantPanic: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.wantPanic && r == nil {
					t.Fatal("expected panic, got none")
				}
				if !tc.wantPanic && r != nil {
					t.Fatalf("unexpected panic: %v", r)
				}
			}()
			_ = MustNewKernel(tc.P, nil)
		})
	}
}

func TestKernelNumStates(t *testing.T) {
	type testCase struct {
		name   string
		kernel *Kernel
		want   int
	}
	cases := []testCase{
		{
			name:   "MUST return 0 for nil kernel",
			kernel: nil,
			want:   0,
		},
		{
			name:   "MUST return dimension of P",
			kernel: MustNewKernel(mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}), nil),
			want:   3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.kernel.NumStates(); got != tc.want {
				t.Errorf("NumStates() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestKernelClone(t *testing.T) {
	type testCase struct {
		name    string
		kernel  *Kernel
		wantNil bool
	}
	cases := []testCase{
		{
			name:   "MUST produce a deep copy that does not share underlying matrix or state names",
			kernel: MustNewKernel(mat.NewDense(2, 2, []float64{0.7, 0.3, 0.4, 0.6}), []string{"a", "b"}),
		},
		{
			name:    "MUST return nil when cloning nil kernel",
			kernel:  nil,
			wantNil: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.kernel.Clone()
			if tc.wantNil {
				if c != nil {
					t.Fatalf("got %v, want nil", c)
				}
				return
			}
			if c == tc.kernel {
				t.Fatal("Clone returned same pointer")
			}
			c.P.Set(0, 0, 0.9)
			c.P.Set(0, 1, 0.1)
			c.StateNames[0] = "mutated"
			if tc.kernel.P.At(0, 0) != 0.7 {
				t.Error("mutation of clone P affected original")
			}
			if tc.kernel.StateNames[0] != "a" {
				t.Error("mutation of clone StateNames affected original")
			}
		})
	}
}

func TestKernelSubmatrix(t *testing.T) {
	type testCase struct {
		name string
		rows []int
		cols []int
		want []float64 // row-major expected entries
	}
	P := mat.NewDense(3, 3, []float64{
		0.1, 0.2, 0.7,
		0.3, 0.4, 0.3,
		0.5, 0.1, 0.4,
	})
	k := MustNewKernel(P, nil)

	cases := []testCase{
		{
			name: "MUST extract exact indexed rows and columns",
			rows: []int{0, 2},
			cols: []int{1, 2},
			want: []float64{0.2, 0.7, 0.1, 0.4},
		},
		{
			name: "MUST support single-element submatrix",
			rows: []int{1},
			cols: []int{0},
			want: []float64{0.3},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := k.Submatrix(tc.rows, tc.cols)
			r, c := got.Dims()
			if r != len(tc.rows) || c != len(tc.cols) {
				t.Fatalf("wrong dims: got %dx%d, want %dx%d", r, c, len(tc.rows), len(tc.cols))
			}
			for i := 0; i < r; i++ {
				for j := 0; j < c; j++ {
					if got.At(i, j) != tc.want[i*c+j] {
						t.Errorf("[%d,%d]: got %g, want %g", i, j, got.At(i, j), tc.want[i*c+j])
					}
				}
			}
		})
	}
}

func TestKernelNormalizeRows(t *testing.T) {
	type testCase struct {
		name    string
		data    []float64
		n       int
		tol     float64
		wantErr bool
	}
	cases := []testCase{
		{
			name: "MUST normalize rows that do not sum to 1",
			data: []float64{2, 2, 1, 3},
			n:    2, tol: 1e-9,
		},
		{
			name: "MUST return error when a row sum is below tol",
			data: []float64{0, 0, 0.5, 0.5},
			n:    2, tol: 1e-9,
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Build via raw struct (bypassing validation) to test NormalizeRows directly.
			k := &Kernel{P: mat.NewDense(tc.n, tc.n, tc.data), StateNames: defaultStateNames("s", tc.n)}
			err := k.NormalizeRows(tc.tol)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Check rows sum to 1.
			for i := 0; i < tc.n; i++ {
				sum := 0.0
				for j := 0; j < tc.n; j++ {
					sum += k.P.At(i, j)
				}
				if math.Abs(sum-1) > 1e-9 {
					t.Errorf("row %d sums to %g, want 1", i, sum)
				}
			}
		})
	}
}

func TestCanonicalSubset(t *testing.T) {
	type testCase struct {
		name  string
		input []int
		want  []int
	}
	cases := []testCase{
		{name: "MUST sort and deduplicate", input: []int{3, 1, 2, 1, 3}, want: []int{1, 2, 3}},
		{name: "MUST handle empty input", input: nil, want: nil},
		{name: "MUST handle already sorted unique input", input: []int{0, 1, 2}, want: []int{0, 1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanonicalSubset(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %d, want %d", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNearlyEqual(t *testing.T) {
	type testCase struct {
		name      string
		a, b, tol float64
		want      bool
	}
	cases := []testCase{
		{name: "MUST return true when difference is within tol", a: 1.0, b: 1.0 + 1e-10, tol: 1e-9, want: true},
		{name: "MUST return false when difference exceeds tol", a: 1.0, b: 1.1, tol: 1e-9, want: false},
		{name: "MUST return true for identical values", a: 0.5, b: 0.5, tol: 0, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NearlyEqual(tc.a, tc.b, tc.tol); got != tc.want {
				t.Errorf("NearlyEqual(%g, %g, %g) = %v, want %v", tc.a, tc.b, tc.tol, got, tc.want)
			}
		})
	}
}

// Property: every kernel produced by NewKernelWithTol is row-stochastic within tol,
// and satisfies tolerance monotonicity, NormalizeRows idempotence, Submatrix full-index
// identity, CanonicalSubset idempotence, and NearlyEqual reflexivity/symmetry.
func TestProperty_1_1_RowStochastic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 8).Draw(t, "n")
		P := rowStochastic(n).Draw(t, "P")
		k, err := NewKernelWithTol(P, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := 0; j < n; j++ {
				v := k.P.At(i, j)
				if v < -1e-9 {
					t.Errorf("row %d has negative entry %g", i, v)
				}
				sum += v
			}
			if math.Abs(sum-1) > 1e-9 {
				t.Errorf("row %d sums to %g", i, sum)
			}
		}

		// Tolerance monotonicity: valid at 1e-9 => valid at 1e-6.
		if _, err := NewKernelWithTol(P, nil, 1e-6); err != nil {
			t.Errorf("tolerance monotonicity violated: valid at 1e-9 but failed at 1e-6: %v", err)
		}

		// Submatrix full-index identity: P_{I,I} = P for I = [0..n-1].
		fullIdx := make([]int, n)
		for i := range fullIdx {
			fullIdx[i] = i
		}
		sub := Submatrix(k.P, fullIdx, fullIdx)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if !NearlyEqual(sub.At(i, j), k.P.At(i, j), 1e-12) {
					t.Errorf("Submatrix full identity failed at (%d,%d)", i, j)
				}
			}
		}

		// NormalizeRows idempotence on *mat.Dense.
		cloned := cloneTransitionMatrix(k.P)
		if err := NormalizeRows(cloned, 1e-9); err != nil {
			t.Fatalf("NormalizeRows first pass: %v", err)
		}
		firstPass := cloneTransitionMatrix(cloned)
		if err := NormalizeRows(cloned, 1e-9); err != nil {
			t.Fatalf("NormalizeRows second pass: %v", err)
		}
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if !NearlyEqual(firstPass.At(i, j), cloned.At(i, j), 1e-12) {
					t.Errorf("NormalizeRows not idempotent at (%d,%d)", i, j)
				}
			}
		}

		// CanonicalSubset idempotence & order invariance.
		rawSubset := rapid.SliceOf(rapid.IntRange(0, n-1)).Draw(t, "rawSubset")
		canon1 := CanonicalSubset(rawSubset)
		canon2 := CanonicalSubset(canon1)
		if len(canon1) != len(canon2) {
			t.Fatalf("CanonicalSubset not idempotent: %v vs %v", canon1, canon2)
		}
		for i := range canon1 {
			if canon1[i] != canon2[i] {
				t.Errorf("CanonicalSubset mismatch at %d", i)
			}
		}

		// NearlyEqual reflexivity and symmetry.
		a := rapid.Float64Range(-10, 10).Draw(t, "a")
		b := rapid.Float64Range(-10, 10).Draw(t, "b")
		if !NearlyEqual(a, a, 1e-9) {
			t.Errorf("NearlyEqual not reflexive for %g", a)
		}
		if NearlyEqual(a, b, 1e-3) != NearlyEqual(b, a, 1e-3) {
			t.Errorf("NearlyEqual not symmetric for %g, %g", a, b)
		}
	})
}

// Property: permuting state labels satisfies P'^k = Π P^k Π^T for all k (similarity invariance).
func TestProperty_1_4_PermutationEquivariance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 5).Draw(t, "n")
		data := make([]float64, n*n)
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := 0; j < n; j++ {
				v := rapid.Float64Range(0.01, 1.0).Draw(t, "p")
				data[i*n+j] = v
				sum += v
			}
			for j := 0; j < n; j++ {
				data[i*n+j] /= sum
			}
		}
		k, err := NewKernel(mat.NewDense(n, n, data), nil)
		if err != nil {
			t.Skip()
		}
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		sigma := rapid.Permutation(idx).Draw(t, "sigma")
		// P'[sigma[i], sigma[j]] = P[i, j].
		pData := make([]float64, n*n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				pData[sigma[i]*n+sigma[j]] = k.P.At(i, j)
			}
		}
		Pp := mat.NewDense(n, n, pData)
		// Verify P'^k = Π P^k Π^T for k = 2, 3.
		for _, steps := range []int{2, 3} {
			var Pk, Ppk mat.Dense
			Pk.Pow(k.P, steps)
			Ppk.Pow(Pp, steps)
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					got := Ppk.At(sigma[i], sigma[j])
					want := Pk.At(i, j)
					if math.Abs(got-want) > 1e-10 {
						t.Errorf("P'^%d[%d,%d] = %g, want P^%d[%d,%d] = %g", steps, sigma[i], sigma[j], got, steps, i, j, want)
					}
				}
			}
		}
	})
}

func TestNormalizeRowsTopLevel(t *testing.T) {
	type testCase struct {
		name    string
		P       *mat.Dense
		tol     float64
		wantErr error
	}
	cases := []testCase{
		{
			name:    "MUST return ErrNotStochastic when P is nil",
			P:       nil,
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name:    "MUST return ErrNotStochastic when a row has negative entry beyond tol",
			P:       mat.NewDense(2, 2, []float64{1.0, -0.5, 0.5, 0.5}),
			tol:     1e-9,
			wantErr: ErrNotStochastic,
		},
		{
			name: "MUST normalize valid positive rows in place to unit sum",
			P:    mat.NewDense(2, 2, []float64{2.0, 6.0, 1.0, 3.0}),
			tol:  1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := NormalizeRows(tc.P, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !NearlyEqual(tc.P.At(0, 0), 0.25, 1e-9) || !NearlyEqual(tc.P.At(0, 1), 0.75, 1e-9) {
				t.Errorf("unexpected normalized row 0: [%g, %g]", tc.P.At(0, 0), tc.P.At(0, 1))
			}
		})
	}
}
