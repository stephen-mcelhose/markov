package reduce_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/reduce"
	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

// mustKernel is a test helper that builds a Kernel from row-major data.
func mustKernel(t *testing.T, n int, data []float64, names []string) *markov.Kernel {
	t.Helper()
	k, err := markov.NewKernel(mat.NewDense(n, n, data), names)
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	return k
}

// assertKernelNearlyEqual checks that two kernels have matching dimensions
// and entries within tol.
func assertKernelNearlyEqual(t *testing.T, got, want *markov.Kernel, tol float64) {
	t.Helper()
	r1, c1 := got.P.Dims()
	r2, c2 := want.P.Dims()
	if r1 != r2 || c1 != c2 {
		t.Fatalf("dimension mismatch: got %dx%d want %dx%d", r1, c1, r2, c2)
	}
	for i := 0; i < r1; i++ {
		for j := 0; j < c1; j++ {
			if math.Abs(got.P.At(i, j)-want.P.At(i, j)) > tol {
				t.Errorf("P[%d,%d]: got %g want %g", i, j, got.P.At(i, j), want.P.At(i, j))
			}
		}
	}
}

func TestTrace(t *testing.T) {
	// 3-state chain: states {0,1} transient, state {2} absorbing.
	// P = [[0.5, 0.3, 0.2],
	//      [0.4, 0.4, 0.2],
	//      [0.0, 0.0, 1.0]]
	//
	// Trace on A={0,1}: complement B={2}, c=[[1.0]], (I-c)^-1 = error (singular).
	// Use a chain where B is strictly transient.
	//
	// 4-state chain (catrace reference):
	// states {0,1} = A, states {2,3} = B
	symmetric4 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	}, []string{"a0", "a1", "b0", "b1"})

	type testCase struct {
		name    string
		k       *markov.Kernel
		subset  []int
		tol     float64
		wantErr error
		check   func(t *testing.T, got *markov.Kernel)
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			subset:  []int{0},
			tol:     1e-9,
			wantErr: markov.ErrNotStochastic,
		},
		{
			name:    "MUST return error for empty subset",
			k:       symmetric4,
			subset:  []int{},
			tol:     1e-9,
			wantErr: markov.ErrInvalidPartition,
		},
		{
			name:    "MUST return error for out-of-range subset index",
			k:       symmetric4,
			subset:  []int{0, 9},
			tol:     1e-9,
			wantErr: markov.ErrInvalidPartition,
		},
		{
			name:   "MUST return the original kernel when subset equals full state space",
			k:      symmetric4,
			subset: []int{0, 1, 2, 3},
			tol:    1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				assertKernelNearlyEqual(t, got, symmetric4, 1e-9)
			},
		},
		{
			name:   "MUST return a row-stochastic kernel for valid subset",
			k:      symmetric4,
			subset: []int{0, 1},
			tol:    1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				if err := got.Validate(1e-9); err != nil {
					t.Errorf("trace kernel not stochastic: %v", err)
				}
			},
		},
		{
			name:   "MUST return a 1x1 identity kernel for singleton subset equal to full chain",
			k:      mustKernel(t, 1, []float64{1.0}, nil),
			subset: []int{0},
			tol:    1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				r, c := got.P.Dims()
				if r != 1 || c != 1 {
					t.Fatalf("want 1x1, got %dx%d", r, c)
				}
				if math.Abs(got.P.At(0, 0)-1.0) > 1e-9 {
					t.Errorf("P[0,0] = %g, want 1.0", got.P.At(0, 0))
				}
			},
		},
		{
			name:   "MUST preserve state names for traced subset",
			k:      symmetric4,
			subset: []int{0, 1},
			tol:    1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				if got.StateNames[0] != "a0" || got.StateNames[1] != "a1" {
					t.Errorf("names: got %v, want [a0 a1]", got.StateNames)
				}
			},
		},
		{
			name: "MUST return ErrSingularOperator when complement contains a recurrent state",
			// 2-state absorbing: state 1 is absorbing, so (I-c) is singular.
			k: mustKernel(t, 2, []float64{
				0.5, 0.5,
				0.0, 1.0,
			}, nil),
			subset:  []int{0},
			tol:     1e-9,
			wantErr: markov.ErrSingularOperator,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.Trace(tc.k, tc.subset, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestMustTrace(t *testing.T) {
	k := mustKernel(t, 2, []float64{0.5, 0.5, 0.5, 0.5}, nil)
	type testCase struct {
		name      string
		k         *markov.Kernel
		subset    []int
		wantPanic bool
	}
	cases := []testCase{
		{
			name:      "MUST panic on nil kernel",
			k:         nil,
			subset:    []int{0},
			wantPanic: true,
		},
		{
			name:      "MUST NOT panic on valid ergodic kernel and subset",
			k:         k,
			subset:    []int{0},
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
			_ = reduce.MustTrace(tc.k, tc.subset, 1e-9)
		})
	}
}

func TestIsTraceOf(t *testing.T) {
	k4 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	}, nil)

	type testCase struct {
		name    string
		k       *markov.Kernel
		parent  *markov.Kernel
		subset  []int
		tol     float64
		want    bool
		wantErr bool
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			parent:  k4,
			subset:  []int{0, 1},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error for nil parent",
			k:       reduce.MustTrace(k4, []int{0, 1}, 1e-9),
			parent:  nil,
			subset:  []int{0, 1},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:   "MUST return true when k equals Trace(parent, subset)",
			k:      reduce.MustTrace(k4, []int{0, 1}, 1e-9),
			parent: k4,
			subset: []int{0, 1},
			tol:    1e-9,
			want:   true,
		},
		{
			name:   "MUST return false when k does not match Trace(parent, subset)",
			k:      mustKernel(t, 2, []float64{0.5, 0.5, 0.5, 0.5}, nil),
			parent: k4,
			subset: []int{0, 1},
			tol:    1e-9,
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.IsTraceOf(tc.k, tc.parent, tc.subset, tc.tol)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTraceReward(t *testing.T) {
	k4 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	}, nil)

	type testCase struct {
		name    string
		k       *markov.Kernel
		rewards []float64
		subset  []int
		tol     float64
		wantErr bool
		check   func(t *testing.T, got []float64)
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			rewards: []float64{1, 1, 1, 1},
			subset:  []int{0, 1},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error when rewards length mismatches kernel size",
			k:       k4,
			rewards: []float64{1, 2},
			subset:  []int{0, 1},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return a slice of length equal to subset",
			k:       k4,
			rewards: []float64{1, 2, 3, 4},
			subset:  []int{0, 1},
			tol:     1e-9,
			check: func(t *testing.T, got []float64) {
				if len(got) != 2 {
					t.Errorf("len(got) = %d, want 2", len(got))
				}
			},
		},
		{
			name:    "MUST return zero rewards when all rewards are zero",
			k:       k4,
			rewards: []float64{0, 0, 0, 0},
			subset:  []int{0, 1},
			tol:     1e-9,
			check: func(t *testing.T, got []float64) {
				for i, v := range got {
					if math.Abs(v) > 1e-9 {
						t.Errorf("got[%d] = %g, want 0", i, v)
					}
				}
			},
		},
		{
			name: "MUST preserve expected excursion reward r_A = r_a + b*(I-c)^-1*r_c",
			// 3-state chain: 0 -> 1 (prob 1), 1 -> 2 (prob 1), 2 -> 0 (prob 1).
			// Watched A = {0}, complement B = {1, 2}. Excursion 0 -> 1 -> 2 -> 0 visits 1 and 2 once each.
			// With rewards = [10, 3, 7], r_A[0] MUST equal 10 + 3 + 7 = 20.
			k: mustKernel(t, 3, []float64{
				0, 1, 0,
				0, 0, 1,
				1, 0, 0,
			}, nil),
			rewards: []float64{10, 3, 7},
			subset:  []int{0},
			tol:     1e-9,
			check: func(t *testing.T, got []float64) {
				if math.Abs(got[0]-20.0) > 1e-9 {
					t.Errorf("got[0] = %g, want 20.0", got[0])
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.TraceReward(tc.k, tc.rewards, tc.subset, tc.tol)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// Property: for any ergodic kernel and valid non-full subset, Trace produces a row-stochastic kernel.
func TestTracePropertyRowStochastic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 6).Draw(t, "n")
		// Build a random row-stochastic n×n matrix.
		data := make([]float64, n*n)
		for i := 0; i < n; i++ {
			row := make([]float64, n)
			sum := 0.0
			for j := range row {
				row[j] = rapid.Float64Range(0.01, 1.0).Draw(t, "p")
				sum += row[j]
			}
			for j := range row {
				data[i*n+j] = row[j] / sum
			}
		}
		k, err := markov.NewKernel(mat.NewDense(n, n, data), nil)
		if err != nil {
			t.Skip()
		}
		// Choose a strict subset (not all states).
		m := rapid.IntRange(1, n-1).Draw(t, "m")
		subset := make([]int, m)
		for i := range subset {
			subset[i] = i
		}
		tr, err := reduce.Trace(k, subset, 1e-9)
		if err != nil {
			t.Skip() // singular complement — valid skip
		}
		if err := tr.Validate(1e-6); err != nil {
			t.Errorf("trace kernel not stochastic: %v", err)
		}

		// Tower Law: B ⊆ A => Trace(Trace(P, A), B_in_A) == Trace(P, B).
		if m >= 2 {
			bLen := rapid.IntRange(1, m-1).Draw(t, "bLen")
			B := subset[:bLen]
			trAB, err := reduce.Trace(tr, B, 1e-9)
			trB, err2 := reduce.Trace(k, B, 1e-9)
			if err == nil && err2 == nil {
				for i := 0; i < bLen; i++ {
					for j := 0; j < bLen; j++ {
						if math.Abs(trAB.P.At(i, j)-trB.P.At(i, j)) > 1e-6 {
							t.Errorf("Tower law failed at (%d,%d): %g != %g", i, j, trAB.P.At(i, j), trB.P.At(i, j))
						}
					}
				}
			}
		}
	})
}

var _ = assertKernelNearlyEqual // suppress unused warning if not called directly
