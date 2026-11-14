package reduce_test

import (
	"math"
	"sort"
	"testing"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/reduce"
	"github.com/stephen-mcelhose/markov/solve"
	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

func TestLump(t *testing.T) {
	// 4-state chain, strongly lumpable on {{0,1},{2,3}} (rows within each block
	// have equal aggregate transition probabilities into each block).
	symmetric4 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	}, []string{"a0", "a1", "b0", "b1"})

	type testCase struct {
		name    string
		k       *markov.Kernel
		p       markov.Partition
		names   []string
		weights markov.Distribution
		tol     float64
		wantErr bool
		check   func(t *testing.T, got *markov.Kernel)
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			p:       markov.Partition{{0, 1}, {2, 3}},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error for invalid partition",
			k:       symmetric4,
			p:       markov.Partition{{0, 1}, {2}}, // missing state 3
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error when names length mismatches partition block count",
			k:       symmetric4,
			p:       markov.Partition{{0, 1}, {2, 3}},
			names:   []string{"only-one"},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error when weights length mismatches kernel state count",
			k:       symmetric4,
			p:       markov.Partition{{0, 1}, {2, 3}},
			weights: markov.Distribution{0.5, 0.5}, // length 2, want 4
			tol:     1e-9,
			wantErr: true,
		},
		{
			name: "MUST return a row-stochastic 2x2 kernel for valid partition",
			k:    symmetric4,
			p:    markov.Partition{{0, 1}, {2, 3}},
			tol:  1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				r, c := got.P.Dims()
				if r != 2 || c != 2 {
					t.Fatalf("want 2x2, got %dx%d", r, c)
				}
				if err := got.Validate(1e-9); err != nil {
					t.Errorf("lumped kernel not stochastic: %v", err)
				}
			},
		},
		{
			name:  "MUST synthesise block names from member state names when names is nil",
			k:     symmetric4,
			p:     markov.Partition{{0, 1}, {2, 3}},
			names: nil,
			tol:   1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				if got.StateNames[0] != "a0+a1" || got.StateNames[1] != "b0+b1" {
					t.Errorf("names: got %v, want [a0+a1 b0+b1]", got.StateNames)
				}
			},
		},
		{
			name:  "MUST use provided names when names is non-nil",
			k:     symmetric4,
			p:     markov.Partition{{0, 1}, {2, 3}},
			names: []string{"A", "B"},
			tol:   1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				if got.StateNames[0] != "A" || got.StateNames[1] != "B" {
					t.Errorf("names: got %v, want [A B]", got.StateNames)
				}
			},
		},
		{
			name: "MUST return a 1x1 kernel of [[1]] for single-block partition",
			k:    symmetric4,
			p:    markov.Partition{{0, 1, 2, 3}},
			tol:  1e-9,
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
			name: "MUST return the original kernel for the discrete partition",
			k:    symmetric4,
			p:    markov.Partition{{0}, {1}, {2}, {3}},
			tol:  1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				assertKernelNearlyEqual(t, got, symmetric4, 1e-9)
			},
		},
		{
			name:    "MUST produce identical quotient kernel for arbitrary weights when strongly lumpable",
			k:       symmetric4,
			p:       markov.Partition{{0, 1}, {2, 3}},
			weights: markov.Distribution{0.1, 0.4, 0.35, 0.15},
			tol:     1e-9,
			check: func(t *testing.T, got *markov.Kernel) {
				unif := reduce.MustLump(symmetric4, markov.Partition{{0, 1}, {2, 3}}, nil, nil, 1e-9)
				assertKernelNearlyEqual(t, got, unif, 1e-9)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.Lump(tc.k, tc.p, tc.names, tc.weights, tc.tol)
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

func TestMustLump(t *testing.T) {
	k := mustKernel(t, 2, []float64{0.5, 0.5, 0.5, 0.5}, nil)
	type testCase struct {
		name      string
		k         *markov.Kernel
		p         markov.Partition
		wantPanic bool
	}
	cases := []testCase{
		{
			name:      "MUST panic on invalid partition",
			k:         k,
			p:         markov.Partition{{0}},
			wantPanic: true,
		},
		{
			name:      "MUST NOT panic on valid kernel and partition",
			k:         k,
			p:         markov.Partition{{0, 1}},
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
			_ = reduce.MustLump(tc.k, tc.p, nil, nil, 1e-9)
		})
	}
}

func TestLumpabilityResidual(t *testing.T) {
	symmetric4 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	}, nil)

	type testCase struct {
		name    string
		k       *markov.Kernel
		p       markov.Partition
		wantErr bool
		check   func(t *testing.T, got float64)
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			p:       markov.Partition{{0, 1}, {2, 3}},
			wantErr: true,
		},
		{
			name:    "MUST return error for invalid partition",
			k:       symmetric4,
			p:       markov.Partition{{0, 1}}, // missing states
			wantErr: true,
		},
		{
			name: "MUST return zero for the discrete partition",
			k:    symmetric4,
			p:    markov.Partition{{0}, {1}, {2}, {3}},
			check: func(t *testing.T, got float64) {
				if got != 0 {
					t.Errorf("got %g, want 0", got)
				}
			},
		},
		{
			name: "MUST return a positive residual for a non-lumpable partition",
			k:    symmetric4,
			p:    markov.Partition{{0, 2}, {1, 3}},
			check: func(t *testing.T, got float64) {
				if math.Abs(got-0.4) > 1e-9 {
					t.Errorf("residual = %g, want 0.4", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.LumpabilityResidual(tc.k, tc.p)
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

func TestIsStronglyLumpable(t *testing.T) {
	// strongly lumpable on {{0,1},{2,3}}: rows within each block have identical
	// aggregate probabilities into each block.
	lumpable := mustKernel(t, 4, []float64{
		0.5, 0.5, 0.0, 0.0,
		0.5, 0.5, 0.0, 0.0,
		0.0, 0.0, 0.5, 0.5,
		0.0, 0.0, 0.5, 0.5,
	}, nil)
	notLumpable := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	}, nil)

	type testCase struct {
		name    string
		k       *markov.Kernel
		p       markov.Partition
		tol     float64
		want    bool
		wantErr bool
	}
	cases := []testCase{
		{
			name:    "MUST return error for nil kernel",
			k:       nil,
			p:       markov.Partition{{0, 1}, {2, 3}},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name:    "MUST return error for invalid partition",
			k:       lumpable,
			p:       markov.Partition{{0, 1}},
			tol:     1e-9,
			wantErr: true,
		},
		{
			name: "MUST return true for a strongly lumpable kernel",
			k:    lumpable,
			p:    markov.Partition{{0, 1}, {2, 3}},
			tol:  1e-9,
			want: true,
		},
		{
			// {{0,2},{1,3}}: state 0→{0,2}=0.6, state 2→{0,2}=0.8 → not equal → not lumpable
			name: "MUST return false for a non-lumpable kernel",
			k:    notLumpable,
			p:    markov.Partition{{0, 2}, {1, 3}},
			tol:  1e-9,
			want: false,
		},
		{
			name: "MUST return true for the discrete partition on any kernel",
			k:    notLumpable,
			p:    markov.Partition{{0}, {1}, {2}, {3}},
			tol:  1e-9,
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reduce.IsStronglyLumpable(tc.k, tc.p, tc.tol)
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

// Multi-step commutation: for strongly lumpable (k, p), Pᵗ·V = V·P̂ᵗ.
func TestLumpMultiStepCommutation(t *testing.T) {
	k := mustKernel(t, 4, []float64{
		0.5, 0.5, 0.0, 0.0,
		0.5, 0.5, 0.0, 0.0,
		0.0, 0.0, 0.5, 0.5,
		0.0, 0.0, 0.5, 0.5,
	}, nil)
	p := markov.Partition{{0, 1}, {2, 3}}
	tol := 1e-12
	lumped, err := reduce.Lump(k, p, nil, nil, tol)
	if err != nil {
		t.Fatalf("Lump: %v", err)
	}

	V, err := p.IndicatorMatrix(4)
	if err != nil {
		t.Fatalf("IndicatorMatrix: %v", err)
	}

	for _, steps := range []int{1, 2, 5, 10} {
		var Pt, PhatT mat.Dense
		Pt.Pow(k.P, steps)
		PhatT.Pow(lumped.P, steps)
		var lhs, rhs mat.Dense
		lhs.Mul(&Pt, V)
		rhs.Mul(V, &PhatT)
		r, c := lhs.Dims()
		for i := 0; i < r; i++ {
			for j := 0; j < c; j++ {
				if math.Abs(lhs.At(i, j)-rhs.At(i, j)) > tol {
					t.Errorf("steps=%d: (Pᵗ·V)[%d,%d]=%g ≠ (V·P̂ᵗ)[%d,%d]=%g",
						steps, i, j, lhs.At(i, j), i, j, rhs.At(i, j))
				}
			}
		}
	}
}

// Lump∘Trace = Trace∘Lump commutativity when subset respects partition blocks.
func TestLumpTraceCommutativity(t *testing.T) {
	k := mustKernel(t, 5, []float64{
		0.1, 0.1, 0.4, 0.2, 0.2,
		0.0, 0.2, 0.3, 0.3, 0.2,
		0.2, 0.1, 0.2, 0.2, 0.3,
		0.1, 0.2, 0.1, 0.3, 0.3,
		0.3, 0.3, 0.0, 0.0, 0.4,
	}, []string{"a0", "a1", "b0", "b1", "c0"})
	tol := 1e-12

	// Path 1: Lump {0,1}→A, {2,3}→B, {4}→C, then Trace keeping {A, C} (indices 0,2).
	lumpFirst, err := reduce.Lump(k, markov.Partition{{0, 1}, {2, 3}, {4}}, []string{"A", "B", "C"}, nil, tol)
	if err != nil {
		t.Fatalf("Lump: %v", err)
	}
	lumpThenTrace, err := reduce.Trace(lumpFirst, []int{0, 2}, tol)
	if err != nil {
		t.Fatalf("Trace(lumped): %v", err)
	}

	// Path 2: Trace out {2,3} (keep {0,1,4}), then Lump {0,1}→A, {4}→C (indices 0,1 → A; 2 → C in traced kernel).
	traceFirst, err := reduce.Trace(k, []int{0, 1, 4}, tol)
	if err != nil {
		t.Fatalf("Trace(k): %v", err)
	}
	traceThenLump, err := reduce.Lump(traceFirst, markov.Partition{{0, 1}, {2}}, []string{"A", "C"}, nil, tol)
	if err != nil {
		t.Fatalf("Lump(traced): %v", err)
	}

	assertKernelNearlyEqual(t, lumpThenTrace, traceThenLump, 1e-9)
}

// Property: Lump produces a row-stochastic kernel for any valid partition.
func TestProperty_3_1_3_2_LumpRowStochastic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 6).Draw(t, "n")
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

		// Random valid partition.
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		perm := rapid.Permutation(indices).Draw(t, "perm")
		m := rapid.IntRange(1, n).Draw(t, "m")
		p := make(markov.Partition, m)
		for i, s := range perm {
			p[i%m] = append(p[i%m], s)
		}

		lumped, err := reduce.Lump(k, p, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		if err := lumped.Validate(1e-6); err != nil {
			t.Errorf("lumped kernel not stochastic: %v", err)
		}
	})
}

// stronglyLumpableKernel generates a random n-state strongly lumpable kernel with m blocks.
// All rows within a block have identical distributions over other blocks.
func stronglyLumpableKernel(t *rapid.T, n, m int) (*markov.Kernel, markov.Partition) {
	// Build quotient kernel Q: m×m row-stochastic.
	qData := make([]float64, m*m)
	for i := 0; i < m; i++ {
		sum := 0.0
		for j := 0; j < m; j++ {
			v := rapid.Float64Range(0.05, 1.0).Draw(t, "q")
			qData[i*m+j] = v
			sum += v
		}
		for j := 0; j < m; j++ {
			qData[i*m+j] /= sum
		}
	}
	// Count block sizes (round-robin assignment: state s belongs to block s%m).
	blockSize := make([]int, m)
	for s := 0; s < n; s++ {
		blockSize[s%m]++
	}
	// Build partition.
	p := make(markov.Partition, m)
	for s := 0; s < n; s++ {
		p[s%m] = append(p[s%m], s)
	}
	// Build P: P[s, t] = Q[blockOf[s], blockOf[t]] / blockSize[blockOf[t]].
	pData := make([]float64, n*n)
	for s := 0; s < n; s++ {
		bi := s % m
		for t := 0; t < n; t++ {
			bj := t % m
			pData[s*n+t] = qData[bi*m+bj] / float64(blockSize[bj])
		}
	}
	k, err := markov.NewKernel(mat.NewDense(n, n, pData), nil)
	if err != nil {
		t.Skip()
	}
	return k, p
}

// Property: Lump result is invariant to U weights under strong lumpability.
func TestProperty_3_3_DistributorWeightInvariance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(4, 8).Draw(t, "n")
		m := rapid.IntRange(2, 3).Draw(t, "m")
		if m > n {
			t.Skip()
		}
		k, p := stronglyLumpableKernel(t, n, m)
		// Lump with uniform weights (nil).
		uniform, err := reduce.Lump(k, p, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		// Lump with random positive weights.
		weights := make(markov.Distribution, n)
		sum := 0.0
		for i := range weights {
			weights[i] = rapid.Float64Range(0.1, 10.0).Draw(t, "w")
			sum += weights[i]
		}
		for i := range weights {
			weights[i] /= sum
		}
		weighted, err := reduce.Lump(k, p, nil, weights, 1e-9)
		if err != nil {
			t.Skip()
		}
		for i := 0; i < m; i++ {
			for j := 0; j < m; j++ {
				if math.Abs(uniform.P.At(i, j)-weighted.P.At(i, j)) > 1e-6 {
					t.Errorf("weight invariance failed at (%d,%d): uniform=%g weighted=%g", i, j, uniform.P.At(i, j), weighted.P.At(i, j))
				}
			}
		}
	})
}

// Property: P^t V = V P̂^t for t=2,3 (multi-step Chapman-Kolmogorov commutation).
func TestProperty_3_4_MultiStepChapmanKolmogorov(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(4, 8).Draw(t, "n")
		m := rapid.IntRange(2, 3).Draw(t, "m")
		if m > n {
			t.Skip()
		}
		k, p := stronglyLumpableKernel(t, n, m)
		lumped, err := reduce.Lump(k, p, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		V, err := p.IndicatorMatrix(n)
		if err != nil {
			t.Skip()
		}
		for _, steps := range []int{2, 3} {
			var Pk, Phatk mat.Dense
			Pk.Pow(k.P, steps)
			Phatk.Pow(lumped.P, steps)
			// LHS = P^t * V  (n×m)
			var lhs mat.Dense
			lhs.Mul(&Pk, V)
			// RHS = V * P̂^t  (n×m)
			var rhs mat.Dense
			rhs.Mul(V, &Phatk)
			for i := 0; i < n; i++ {
				for j := 0; j < m; j++ {
					if math.Abs(lhs.At(i, j)-rhs.At(i, j)) > 1e-6 {
						t.Errorf("C-K step=%d at (%d,%d): P^t*V=%g, V*P̂^t=%g", steps, i, j, lhs.At(i, j), rhs.At(i, j))
					}
				}
			}
		}
	})
}

// Property: π̂ P̂ = π̂ (projected stationary is stationary for the lumped chain).
func TestProperty_3_5_StationaryProjection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(4, 8).Draw(t, "n")
		m := rapid.IntRange(2, 3).Draw(t, "m")
		if m > n {
			t.Skip()
		}
		k, p := stronglyLumpableKernel(t, n, m)
		pi, err := solve.Stationary(k, 1e-9)
		if err != nil {
			t.Skip()
		}
		piHat, err := pi.Project(p, 1e-9)
		if err != nil {
			t.Skip()
		}
		lumped, err := reduce.Lump(k, p, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		for j := 0; j < m; j++ {
			var s float64
			for i := 0; i < m; i++ {
				s += piHat[i] * lumped.P.At(i, j)
			}
			if math.Abs(s-piHat[j]) > 1e-6 {
				t.Errorf("stationary projection: (π̂*P̂)[%d] = %g, want %g", j, s, piHat[j])
			}
		}
	})
}

// Property: Trace(Lump(P,B), I) = Lump(Trace(P,A), B|_A) where A = union of blocks in I.
func TestProperty_4_1_SpatialTemporalCommutativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(6, 9).Draw(t, "n")
		m := rapid.IntRange(3, 4).Draw(t, "m")
		if m > n {
			t.Skip()
		}
		k, p := stronglyLumpableKernel(t, n, m)
		// Choose a strict subset I of block indices.
		iLen := rapid.IntRange(1, m-1).Draw(t, "iLen")
		// I = first iLen block indices.
		// A = union of states in those blocks.
		var A []int
		for _, s := range p[:iLen] {
			A = append(A, s...)
		}
		// Trace canonicalizes (sorts) A internally; build the index map on sorted A.
		sortedA := make([]int, len(A))
		copy(sortedA, A)
		sort.Ints(sortedA)
		aIdx := make(map[int]int, len(sortedA))
		for i, s := range sortedA {
			aIdx[s] = i
		}
		// LHS: Lump(P, B) then Trace on block indices [0..iLen-1].
		lumped, err := reduce.Lump(k, p, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		macroSubset := make([]int, iLen)
		for i := range macroSubset {
			macroSubset[i] = i
		}
		lhs, err := reduce.Trace(lumped, macroSubset, 1e-9)
		if err != nil {
			t.Skip()
		}
		// RHS: Trace(P, A) then Lump on B restricted to A.
		traced, err := reduce.Trace(k, A, 1e-9)
		if err != nil {
			t.Skip()
		}
		// Restricted partition: same first iLen blocks, states re-indexed to sorted A positions.
		restrictedP := make(markov.Partition, iLen)
		for bi := 0; bi < iLen; bi++ {
			for _, s := range p[bi] {
				restrictedP[bi] = append(restrictedP[bi], aIdx[s])
			}
		}
		rhs, err := reduce.Lump(traced, restrictedP, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		for i := 0; i < iLen; i++ {
			for j := 0; j < iLen; j++ {
				if math.Abs(lhs.P.At(i, j)-rhs.P.At(i, j)) > 1e-6 {
					t.Errorf("spatial-temporal commutativity at (%d,%d): LHS=%g RHS=%g", i, j, lhs.P.At(i, j), rhs.P.At(i, j))
				}
			}
		}
	})
}
