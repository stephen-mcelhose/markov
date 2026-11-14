package markov

import (
	"errors"
	"math"
	"testing"

	"pgregory.net/rapid"
)

// validDistribution is a rapid generator for valid probability distributions of length n.
func validDistribution(n int) *rapid.Generator[Distribution] {
	return rapid.Custom(func(t *rapid.T) Distribution {
		raw := make([]float64, n)
		sum := 0.0
		for i := range raw {
			raw[i] = rapid.Float64Range(0, 1).Draw(t, "entry")
			sum += raw[i]
		}
		if sum == 0 {
			raw[0] = 1.0
			sum = 1.0
		}
		d := make(Distribution, n)
		for i := range d {
			d[i] = raw[i] / sum
		}
		return d
	})
}

func TestDistributionValidate(t *testing.T) {
	type testCase struct {
		name    string
		d       Distribution
		tol     float64
		wantErr error
	}
	cases := []testCase{
		{
			name:    "MUST return ErrInvalidDistribution for an empty distribution",
			d:       Distribution{},
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name:    "MUST return ErrInvalidDistribution when an entry is negative",
			d:       Distribution{-0.1, 1.1},
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name:    "MUST return ErrInvalidDistribution when sum deviates from 1 beyond tol",
			d:       Distribution{0.3, 0.3},
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name: "MUST succeed for a valid distribution",
			d:    Distribution{0.5, 0.5},
			tol:  1e-9,
		},
		{
			name: "MUST succeed for a point mass distribution",
			d:    Distribution{0, 0, 1},
			tol:  1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.d.Validate(tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDistributionRestrict(t *testing.T) {
	type testCase struct {
		name    string
		d       Distribution
		subset  []int
		tol     float64
		wantErr error
		wantSum float64
	}
	cases := []testCase{
		{
			name:    "MUST return ErrInvalidDistribution when a subset index is out of range",
			d:       Distribution{0.5, 0.5},
			subset:  []int{0, 5},
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name:    "MUST return error when total mass on subset is zero",
			d:       Distribution{0, 0, 1},
			subset:  []int{0, 1},
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name:    "MUST renormalize so restricted distribution sums to 1",
			d:       Distribution{0.2, 0.3, 0.5},
			subset:  []int{0, 1},
			tol:     1e-9,
			wantSum: 1.0,
		},
		{
			name:    "MUST return a single-entry distribution of 1 for a singleton subset with all mass",
			d:       Distribution{0, 1, 0},
			subset:  []int{1},
			tol:     1e-9,
			wantSum: 1.0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.d.Restrict(tc.subset, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sum := 0.0
			for _, v := range got {
				sum += v
			}
			if math.Abs(sum-tc.wantSum) > 1e-9 {
				t.Errorf("sum = %g, want %g", sum, tc.wantSum)
			}
		})
	}
}

func TestDistributionProject(t *testing.T) {
	type testCase struct {
		name      string
		d         Distribution
		partition Partition
		tol       float64
		wantErr   error
		wantOut   Distribution
	}
	cases := []testCase{
		{
			name:      "MUST aggregate mass into macro-blocks",
			d:         Distribution{0.1, 0.2, 0.3, 0.4},
			partition: Partition{{0, 1}, {2, 3}},
			tol:       1e-9,
			wantOut:   Distribution{0.3, 0.7},
		},
		{
			name:      "MUST preserve total probability mass",
			d:         Distribution{0.25, 0.25, 0.25, 0.25},
			partition: Partition{{0, 2}, {1, 3}},
			tol:       1e-9,
			wantOut:   Distribution{0.5, 0.5},
		},
		{
			name:      "MUST return ErrInvalidPartition for invalid partition",
			d:         Distribution{0.5, 0.5},
			partition: Partition{{0, 1}, {0}},
			tol:       1e-9,
			wantErr:   ErrInvalidPartition,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.d.Project(tc.partition, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.wantOut) {
				t.Fatalf("len = %d, want %d", len(got), len(tc.wantOut))
			}
			for i := range got {
				if math.Abs(got[i]-tc.wantOut[i]) > 1e-9 {
					t.Errorf("[%d]: got %g, want %g", i, got[i], tc.wantOut[i])
				}
			}
		})
	}
}

func TestValidateDistributionTopLevel(t *testing.T) {
	type testCase struct {
		name    string
		pi      Distribution
		tol     float64
		wantErr error
	}
	cases := []testCase{
		{
			name:    "MUST return ErrInvalidDistribution for nil distribution",
			pi:      nil,
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name: "MUST succeed for valid distribution",
			pi:   Distribution{0.4, 0.6},
			tol:  1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDistribution(tc.pi, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewDistribution(t *testing.T) {
	type testCase struct {
		name    string
		probs   []float64
		tol     float64
		wantErr error
	}
	cases := []testCase{
		{
			name:    "MUST return ErrInvalidDistribution for empty slice",
			probs:   nil,
			tol:     1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name:    "MUST return ErrInvalidDistribution for negative tolerance",
			probs:   []float64{0.5, 0.5},
			tol:     -1e-9,
			wantErr: ErrInvalidDistribution,
		},
		{
			name:  "MUST clamp near-zero negatives after validating sum within tol",
			probs: []float64{1.0 + 0.4e-9, -0.2e-9, -0.2e-9},
			tol:   1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewDistribution(tc.probs, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i, v := range d {
				if v < 0 {
					t.Errorf("entry %d not clamped: %g", i, v)
				}
			}
		})
	}
}

func TestMustNewDistribution(t *testing.T) {
	type testCase struct {
		name      string
		probs     []float64
		wantPanic bool
	}
	cases := []testCase{
		{
			name:      "MUST panic on invalid distribution",
			probs:     []float64{-1, 2},
			wantPanic: true,
		},
		{
			name:      "MUST NOT panic on valid distribution",
			probs:     []float64{0.3, 0.7},
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
			_ = MustNewDistribution(tc.probs, 1e-9)
		})
	}
}

func TestTopLevelDistributionFunctions(t *testing.T) {
	type testCase struct {
		name    string
		pi      Distribution
		wantErr error
	}
	cases := []testCase{
		{
			name:    "MUST return ErrInvalidDistribution when RestrictDistribution receives nil",
			pi:      nil,
			wantErr: ErrInvalidDistribution,
		},
		{
			name: "MUST succeed when RestrictDistribution and ProjectDistribution receive valid inputs",
			pi:   Distribution{0.2, 0.8},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err1 := RestrictDistribution(tc.pi, []int{0, 1}, 1e-9)
			_, err2 := ProjectDistribution(tc.pi, Partition{{0}, {1}}, 1e-9)
			if tc.wantErr != nil {
				if !errors.Is(err1, tc.wantErr) || !errors.Is(err2, tc.wantErr) {
					t.Fatalf("got (%v, %v), want %v", err1, err2, tc.wantErr)
				}
				return
			}
			if err1 != nil || err2 != nil {
				t.Fatalf("unexpected errors: %v, %v", err1, err2)
			}
		})
	}
}

// Property: Restrict output always sums to 1 for any valid distribution and non-zero subset.
func TestDistributionPropertyRestrictSumsToOne(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 8).Draw(t, "n")
		d := validDistribution(n).Draw(t, "d")
		k := rapid.IntRange(1, n).Draw(t, "k")
		subset := rapid.SliceOfNDistinct(rapid.IntRange(0, n-1), k, k, func(x int) int { return x }).Draw(t, "subset")

		got, err := d.Restrict(subset, 1e-9)
		if err != nil {
			t.Skip() // zero mass on subset — skip
		}
		sum := 0.0
		for _, v := range got {
			sum += v
		}
		if math.Abs(sum-1) > 1e-9 {
			t.Errorf("Restrict sum = %g, want 1", sum)
		}

		// Full-space identity when k == n.
		full := make([]int, n)
		for i := range full {
			full[i] = i
		}
		fullGot, err := d.Restrict(full, 1e-9)
		if err != nil {
			t.Fatalf("Restrict full space: %v", err)
		}
		for i := range d {
			if !NearlyEqual(fullGot[i], d[i], 1e-9) {
				t.Errorf("Full-space identity failed at %d: got %g, want %g", i, fullGot[i], d[i])
			}
		}

		// Tower law (nested conditioning): B ⊆ A => Restrict(Restrict(d, A), B_in_A) == Restrict(d, B).
		if len(subset) >= 2 {
			A := CanonicalSubset(subset)
			m := rapid.IntRange(1, len(A)-1).Draw(t, "m")
			B := A[:m]
			relB := make([]int, m)
			for i := range relB {
				relB[i] = i
			}
			dA, err1 := d.Restrict(A, 1e-9)
			dAB, err2 := dA.Restrict(relB, 1e-9)
			dB, err3 := d.Restrict(B, 1e-9)
			if err1 == nil && err2 == nil && err3 == nil {
				for i := range dB {
					if !NearlyEqual(dAB[i], dB[i], 1e-9) {
						t.Errorf("Tower law failed at %d: Restrict(Restrict(d,A),B)=%g != Restrict(d,B)=%g", i, dAB[i], dB[i])
					}
				}
			}
		}
	})
}

// Property: Project output always sums to 1 for any valid distribution and valid partition.
func TestDistributionPropertyProjectSumsToOne(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 8).Draw(t, "n")
		d := validDistribution(n).Draw(t, "d")
		// Build a valid random partition.
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		perm := rapid.Permutation(indices).Draw(t, "perm")
		m := rapid.IntRange(1, n).Draw(t, "m")
		partition := make(Partition, m)
		for i, s := range perm {
			partition[i%m] = append(partition[i%m], s)
		}

		got, err := d.Project(partition, 1e-9)
		if err != nil {
			t.Skip()
		}
		sum := 0.0
		for _, v := range got {
			sum += v
		}
		if math.Abs(sum-1) > 1e-9 {
			t.Errorf("Project sum = %g, want 1", sum)
		}

		// Trivial partition bounds: discrete => pi, single-block => [1.0].
		discrete := make(Partition, n)
		single := make([]int, n)
		for i := 0; i < n; i++ {
			discrete[i] = []int{i}
			single[i] = i
		}
		gotDisc, err := d.Project(discrete, 1e-9)
		if err != nil {
			t.Fatalf("Project discrete: %v", err)
		}
		for i := range d {
			if !NearlyEqual(gotDisc[i], d[i], 1e-9) {
				t.Errorf("discrete bound mismatch at %d", i)
			}
		}
		gotSingle, err := d.Project(Partition{single}, 1e-9)
		if err != nil {
			t.Fatalf("Project single: %v", err)
		}
		if len(gotSingle) != 1 || !NearlyEqual(gotSingle[0], 1.0, 1e-9) {
			t.Errorf("single-block bound mismatch: got %v, want [1.0]", gotSingle)
		}
	})
}
