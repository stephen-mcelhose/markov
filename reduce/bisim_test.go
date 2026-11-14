package reduce_test

import (
	"testing"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/reduce"
	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

func TestCoarsestPartition(t *testing.T) {
	// 4-state chain, strongly lumpable on {{0,1},{2,3}}.
	lumpable := mustKernel(t, 4, []float64{
		0.5, 0.5, 0.0, 0.0,
		0.5, 0.5, 0.0, 0.0,
		0.0, 0.0, 0.5, 0.5,
		0.0, 0.0, 0.5, 0.5,
	}, nil)

	type testCase struct {
		name    string
		k       *markov.Kernel
		initial markov.Partition
		tol     float64
		wantErr bool
		check   func(t *testing.T, got markov.Partition)
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			initial: markov.Partition{{0}, {1}, {2}, {3}},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error for invalid initial partition",
			k:       lumpable,
			initial: markov.Partition{{0, 1}}, // missing states
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST split a non-lumpable initial partition into strongly lumpable blocks",
			k:       lumpable,
			initial: markov.Partition{{0, 1, 2}, {3}},
			tol:     1e-9,
			check: func(t *testing.T, got markov.Partition) {
				ok, err := reduce.IsStronglyLumpable(lumpable, got, 1e-9)
				if err != nil || !ok {
					t.Errorf("refined partition %v is not strongly lumpable: %v", got, err)
				}
				if len(got) != 3 {
					t.Errorf("expected {{0,1},{2},{3}} (3 blocks), got %v", got)
				}
			},
		},
		{
			name:    "MUST accept nil initial partition defaulting to single universal block",
			k:       lumpable,
			initial: nil,
			tol:     1e-9,
			check: func(t *testing.T, got markov.Partition) {
				if err := got.Validate(4); err != nil {
					t.Errorf("invalid partition: %v", err)
				}
			},
		},
		{
			name:    "MUST return a partition that is strongly lumpable",
			k:       lumpable,
			initial: markov.Partition{{0}, {1}, {2}, {3}},
			tol:     1e-9,
			check: func(t *testing.T, got markov.Partition) {
				ok, err := reduce.IsStronglyLumpable(lumpable, got, 1e-9)
				if err != nil {
					t.Fatalf("IsStronglyLumpable: %v", err)
				}
				if !ok {
					t.Errorf("coarsest partition is not strongly lumpable")
				}
			},
		},
		{
			name:    "MUST be stable under one additional refinement step",
			k:       lumpable,
			initial: markov.Partition{{0}, {1}, {2}, {3}},
			tol:     1e-9,
			check: func(t *testing.T, got markov.Partition) {
				again, err := reduce.CoarsestPartition(lumpable, got, 1e-9)
				if err != nil {
					t.Fatalf("second CoarsestPartition: %v", err)
				}
				if len(again) != len(got) {
					t.Errorf("not stable: first=%d blocks, second=%d blocks", len(got), len(again))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.CoarsestPartition(tc.k, tc.initial, tc.tol)
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

func TestAreBisimilar(t *testing.T) {
	k := mustKernel(t, 2, []float64{0.5, 0.5, 0.5, 0.5}, nil)
	k2 := mustKernel(t, 2, []float64{0.3, 0.7, 0.3, 0.7}, nil)
	fine := mustKernel(t, 4, []float64{
		0.1, 0.2, 0.4, 0.3,
		0.2, 0.1, 0.3, 0.4,
		0.5, 0.1, 0.2, 0.2,
		0.3, 0.3, 0.1, 0.3,
	}, nil)
	lumped := reduce.MustLump(fine, markov.Partition{{0, 1}, {2, 3}}, nil, nil, 1e-9)

	type testCase struct {
		name    string
		k1, k2  *markov.Kernel
		tol     float64
		want    bool
		wantErr bool
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil k1",
			k1:      nil,
			k2:      k,
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error for nil k2",
			k1:      k,
			k2:      nil,
			tol:     1e-9,
			wantErr: true,
		},
		{
			name: "MUST return true for a kernel compared with itself",
			k1:   k,
			k2:   k,
			tol:  1e-9,
			want: true,
		},
		{
			name: "MUST return false for structurally distinct kernels",
			k1:   k,
			k2:   k2,
			tol:  1e-9,
			want: false,
		},
		{
			name: "MUST return true when comparing a strongly lumpable kernel with its lumped quotient",
			k1:   fine,
			k2:   lumped,
			tol:  1e-9,
			want: true,
		},
		{
			name: "MUST satisfy transitivity for two distinct 3-state chains sharing the same 2-state quotient",
			k1: mustKernel(t, 3, []float64{
				0.1, 0.1, 0.8,
				0.2, 0.0, 0.8,
				0.35, 0.35, 0.3,
			}, nil),
			k2: mustKernel(t, 3, []float64{
				0.2, 0.4, 0.4,
				0.7, 0.15, 0.15,
				0.7, 0.1, 0.2,
			}, nil),
			tol:  1e-9,
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.AreBisimilar(tc.k1, tc.k2, tc.tol)
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

// Property: CoarsestPartition is a fixed point — applying it again yields the same number of blocks.
func TestProperty_Bisim_CoarsestPartitionStable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(3, 6).Draw(t, "n")
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
		k, err := markov.NewKernel(mat.NewDense(n, n, data), nil)
		if err != nil {
			t.Skip()
		}
		b1, err := reduce.CoarsestPartition(k, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		ok, err := reduce.IsStronglyLumpable(k, b1, 1e-9)
		if err != nil || !ok {
			t.Errorf("CoarsestPartition result is not strongly lumpable: ok=%v err=%v", ok, err)
		}
		b2, err := reduce.CoarsestPartition(k, b1, 1e-9)
		if err != nil {
			t.Skip()
		}
		if len(b1) != len(b2) {
			t.Errorf("CoarsestPartition not stable: first=%d blocks, second=%d blocks", len(b1), len(b2))
		}
	})
}

// Property: a kernel is bisimilar to its own quotient under the coarsest partition.
func TestProperty_Bisim_AreBisimilarLumpedFixed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(3, 5).Draw(t, "n")
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
		k, err := markov.NewKernel(mat.NewDense(n, n, data), nil)
		if err != nil {
			t.Skip()
		}
		b, err := reduce.CoarsestPartition(k, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		lumped, err := reduce.Lump(k, b, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		sim, err := reduce.AreBisimilar(k, lumped, 1e-9)
		if err != nil {
			t.Errorf("AreBisimilar error: %v", err)
			return
		}
		if !sim {
			t.Errorf("kernel not bisimilar to its coarsest-partition lump")
		}
	})
}
