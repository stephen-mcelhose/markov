package markov

import (
	"errors"
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

func TestPartitionValidate(t *testing.T) {
	type testCase struct {
		name      string
		partition Partition
		n         int
		wantErr   error
	}
	cases := []testCase{
		{
			name:      "MUST succeed for a valid multi-block partition",
			partition: Partition{{0, 2}, {1, 3}},
			n:         4,
		},
		{
			name:      "MUST succeed for a discrete partition where each state is its own block",
			partition: Partition{{0}, {1}, {2}},
			n:         3,
		},
		{
			name:      "MUST succeed for a single-block partition containing all states",
			partition: Partition{{0, 1, 2}},
			n:         3,
		},
		{
			name:      "MUST return ErrInvalidPartition when n is zero",
			partition: Partition{{0}},
			n:         0,
			wantErr:   ErrInvalidPartition,
		},
		{
			name:      "MUST return ErrInvalidPartition when partition is empty",
			partition: Partition{},
			n:         3,
			wantErr:   ErrInvalidPartition,
		},
		{
			name:      "MUST return ErrInvalidPartition when a block is empty",
			partition: Partition{{0, 1}, {}, {2}},
			n:         3,
			wantErr:   ErrInvalidPartition,
		},
		{
			name:      "MUST return ErrInvalidPartition when a state index is negative",
			partition: Partition{{-1, 0}, {1, 2}},
			n:         3,
			wantErr:   ErrInvalidPartition,
		},
		{
			name:      "MUST return ErrInvalidPartition when a state index is out of range",
			partition: Partition{{0, 1}, {2, 3}},
			n:         3,
			wantErr:   ErrInvalidPartition,
		},
		{
			name:      "MUST return ErrInvalidPartition when a state appears in more than one block",
			partition: Partition{{0, 1}, {1, 2}},
			n:         3,
			wantErr:   ErrInvalidPartition,
		},
		{
			name:      "MUST return ErrInvalidPartition when partition does not cover all states",
			partition: Partition{{0, 1}},
			n:         3,
			wantErr:   ErrInvalidPartition,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.partition.Validate(tc.n)
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

func TestPartitionIndicatorMatrix(t *testing.T) {
	type testCase struct {
		name      string
		partition Partition
		n         int
		wantErr   bool
		// expected V entries as [row][col]
		check func(t *testing.T, V *mat.Dense)
	}
	cases := []testCase{
		{
			name:      "MUST return ErrInvalidPartition for an invalid partition",
			partition: Partition{{0, 1}, {1, 2}},
			n:         3,
			wantErr:   true,
		},
		{
			name:      "MUST produce a matrix where each row has exactly one 1",
			partition: Partition{{0, 2}, {1, 3}},
			n:         4,
			check: func(t *testing.T, V *mat.Dense) {
				r, _ := V.Dims()
				for i := 0; i < r; i++ {
					ones := 0
					_, c := V.Dims()
					for j := 0; j < c; j++ {
						v := V.At(i, j)
						if v != 0 && v != 1 {
							t.Errorf("row %d col %d: expected 0 or 1, got %g", i, j, v)
						}
						if v == 1 {
							ones++
						}
					}
					if ones != 1 {
						t.Errorf("row %d has %d ones, want exactly 1", i, ones)
					}
				}
			},
		},
		{
			name:      "MUST produce V where right-multiplying ones-vector gives all-ones",
			partition: Partition{{0, 2}, {1, 3}},
			n:         4,
			check: func(t *testing.T, V *mat.Dense) {
				r, c := V.Dims()
				ones := mat.NewDense(c, 1, nil)
				for j := 0; j < c; j++ {
					ones.Set(j, 0, 1.0)
				}
				result := mat.NewDense(r, 1, nil)
				result.Mul(V, ones)
				for i := 0; i < r; i++ {
					if math.Abs(result.At(i, 0)-1) > 1e-9 {
						t.Errorf("V·1 row %d = %g, want 1", i, result.At(i, 0))
					}
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			V, err := tc.partition.IndicatorMatrix(tc.n)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, V)
			}
		})
	}
}

func TestPartitionDistributorMatrix(t *testing.T) {
	type testCase struct {
		name      string
		partition Partition
		n         int
		weights   Distribution
		tol       float64
		wantErr   bool
		check     func(t *testing.T, U *mat.Dense)
	}
	cases := []testCase{
		{
			name:      "MUST return error for an invalid partition",
			partition: Partition{{0, 1}, {1, 2}},
			n:         3,
			tol:       1e-9,
			wantErr:   true,
		},
		{
			name:      "MUST produce uniform rows when weights is nil",
			partition: Partition{{0, 1}, {2, 3}},
			n:         4,
			weights:   nil,
			tol:       1e-9,
			check: func(t *testing.T, U *mat.Dense) {
				// Block 0: states 0,1 → each should be 0.5
				if math.Abs(U.At(0, 0)-0.5) > 1e-9 || math.Abs(U.At(0, 1)-0.5) > 1e-9 {
					t.Errorf("uniform weights: row 0 = [%g, %g], want [0.5, 0.5]", U.At(0, 0), U.At(0, 1))
				}
			},
		},
		{
			name:      "MUST return error when block has zero total weight",
			partition: Partition{{0, 1}, {2}},
			n:         3,
			weights:   Distribution{0, 0, 1},
			tol:       1e-9,
			wantErr:   true,
		},
		{
			name:      "MUST return error when a weight is negative",
			partition: Partition{{0, 1}, {2}},
			n:         3,
			weights:   Distribution{-0.5, 0.5, 1.0},
			tol:       1e-9,
			wantErr:   true,
		},
		{
			name:      "MUST produce weight-proportional rows for non-nil weights",
			partition: Partition{{0, 1}, {2, 3}},
			n:         4,
			weights:   Distribution{0.25, 0.75, 0.6, 0.4},
			tol:       1e-9,
			check: func(t *testing.T, U *mat.Dense) {
				// Block 0: states {0,1}, weights {0.25, 0.75} → U[0,0]=0.25, U[0,1]=0.75
				if math.Abs(U.At(0, 0)-0.25) > 1e-9 || math.Abs(U.At(0, 1)-0.75) > 1e-9 {
					t.Errorf("weighted row 0: got [%g, %g], want [0.25, 0.75]", U.At(0, 0), U.At(0, 1))
				}
				// Block 1: states {2,3}, weights {0.6, 0.4} → U[1,2]=0.6, U[1,3]=0.4
				if math.Abs(U.At(1, 2)-0.6) > 1e-9 || math.Abs(U.At(1, 3)-0.4) > 1e-9 {
					t.Errorf("weighted row 1: got [%g, %g], want [0.6, 0.4]", U.At(1, 2), U.At(1, 3))
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			U, err := tc.partition.DistributorMatrix(tc.n, tc.weights, tc.tol)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, U)
			}
		})
	}
}

func checkUVIdentity(t *rapid.T, m, n int, partition Partition, weights Distribution) {
	t.Helper()
	V, err := partition.IndicatorMatrix(n)
	if err != nil {
		t.Fatal(err)
	}
	U, err := partition.DistributorMatrix(n, weights, 1e-9)
	if err != nil {
		t.Skip() // e.g. zero-weight block — skip this draw
	}
	UV := mat.NewDense(m, m, nil)
	UV.Mul(U, V)
	for i := 0; i < m; i++ {
		rowSumU := 0.0
		for s := 0; s < n; s++ {
			rowSumU += U.At(i, s)
		}
		if math.Abs(rowSumU-1.0) > 1e-9 {
			t.Errorf("U row %d sums to %g, want 1", i, rowSumU)
		}
		colSumV := 0.0
		for s := 0; s < n; s++ {
			colSumV += V.At(s, i)
		}
		if math.Abs(colSumV-float64(len(partition[i]))) > 1e-9 {
			t.Errorf("V col %d sums to %g, want |B_%d|=%d", i, colSumV, i, len(partition[i]))
		}
		for j := 0; j < m; j++ {
			want := 0.0
			if i == j {
				want = 1.0
			}
			if math.Abs(UV.At(i, j)-want) > 1e-9 {
				t.Errorf("UV[%d,%d] = %g, want %g", i, j, UV.At(i, j), want)
			}
		}
	}
}

func TestTopLevelPartitionFunctions(t *testing.T) {
	type testCase struct {
		name    string
		p       Partition
		n       int
		wantErr error
	}
	cases := []testCase{
		{
			name:    "MUST return ErrInvalidPartition when partition is nil",
			p:       nil,
			n:       2,
			wantErr: ErrInvalidPartition,
		},
		{
			name: "MUST succeed for valid partition across ValidatePartition IndicatorMatrix and DistributorMatrix",
			p:    Partition{{0}, {1}},
			n:    2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err1 := ValidatePartition(tc.p, tc.n)
			_, err2 := IndicatorMatrix(tc.p, tc.n)
			_, err3 := DistributorMatrix(tc.p, tc.n, nil, 1e-9)
			if tc.wantErr != nil {
				if !errors.Is(err1, tc.wantErr) || !errors.Is(err2, tc.wantErr) || !errors.Is(err3, tc.wantErr) {
					t.Fatalf("got (%v, %v, %v), want %v", err1, err2, err3, tc.wantErr)
				}
				return
			}
			if err1 != nil || err2 != nil || err3 != nil {
				t.Fatalf("unexpected errors: %v, %v, %v", err1, err2, err3)
			}
		})
	}
}

// Property (invariant §1.3): U·V = Iₘ for any valid partition and uniform weights.
func TestPartitionPropertyUVEqualsIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 8).Draw(t, "n")
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
		checkUVIdentity(t, m, n, partition, nil)
	})
}

// Property: U·V = Iₘ also holds for arbitrary non-negative weights.
func TestPartitionPropertyUVEqualsIdentityWeighted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 8).Draw(t, "n")
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
		weights := make(Distribution, n)
		for i := range weights {
			weights[i] = rapid.Float64Range(0, 1).Draw(t, "w")
		}
		checkUVIdentity(t, m, n, partition, weights)
	})
}
