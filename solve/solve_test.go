package solve_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/reduce"
	"github.com/stephen-mcelhose/markov/solve"
	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

func mustKernel(t *testing.T, n int, data []float64) *markov.Kernel {
	t.Helper()
	return markov.MustNewKernel(mat.NewDense(n, n, data), nil)
}

func randomErgodicKernel(n int) *rapid.Generator[*markov.Kernel] {
	return rapid.Custom(func(t *rapid.T) *markov.Kernel {
		data := make([]float64, n*n)
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := 0; j < n; j++ {
				v := rapid.Float64Range(0.05, 1.0).Draw(t, "p")
				data[i*n+j] = v
				sum += v
			}
			for j := 0; j < n; j++ {
				data[i*n+j] /= sum
			}
		}
		return markov.MustNewKernel(mat.NewDense(n, n, data), nil)
	})
}

func TestClassifyStates(t *testing.T) {
	// 3-state chain: 0 transient -> {1, 2} periodic 2-cycle (1 <-> 2).
	transientTo2Cycle := mustKernel(t, 3, []float64{
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0,
		0.0, 1.0, 0.0,
	})
	// 2 absorbing traps.
	twoTraps := mustKernel(t, 2, []float64{
		1.0, 0.0,
		0.0, 1.0,
	})

	type testCase struct {
		name    string
		k       *markov.Kernel
		tol     float64
		wantErr error
		check   func(t *testing.T, sc *solve.StateClasses)
	}
	cases := []testCase{
		{
			name:    "MUST return ErrNotStochastic for nil kernel",
			k:       nil,
			tol:     1e-9,
			wantErr: markov.ErrNotStochastic,
		},
		{
			name:    "MUST return ErrNotStochastic for negative tolerance",
			k:       twoTraps,
			tol:     -1e-9,
			wantErr: markov.ErrNotStochastic,
		},
		{
			name: "MUST identify transient state and periodic recurrent class with period 2",
			k:    transientTo2Cycle,
			tol:  1e-9,
			check: func(t *testing.T, sc *solve.StateClasses) {
				if err := sc.AsPartition().Validate(3); err != nil {
					t.Errorf("AsPartition invalid: %v", err)
				}
				if len(sc.Recurrent) != 1 || len(sc.Recurrent[0]) != 2 {
					t.Errorf("Recurrent = %v, want [[1 2]]", sc.Recurrent)
				}
				if len(sc.Transient) != 1 || sc.Transient[0] != 0 {
					t.Errorf("Transient = %v, want [0]", sc.Transient)
				}
				if sc.Periods[1] != 2 || sc.Periods[2] != 2 {
					t.Errorf("Periods = %v, want period 2 on states 1,2", sc.Periods)
				}
			},
		},
		{
			name: "MUST identify multiple closed recurrent classes for absorbing traps",
			k:    twoTraps,
			tol:  1e-9,
			check: func(t *testing.T, sc *solve.StateClasses) {
				if len(sc.Recurrent) != 2 {
					t.Errorf("len(Recurrent) = %d, want 2", len(sc.Recurrent))
				}
				if len(sc.Transient) != 0 {
					t.Errorf("Transient = %v, want empty", sc.Transient)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := solve.ClassifyStates(tc.k, tc.tol)
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

func TestStationary(t *testing.T) {
	// Asymmetric 2-state chain: P = [[0.8, 0.2], [0.6, 0.4]] -> pi = [0.75, 0.25].
	asym2 := mustKernel(t, 2, []float64{
		0.8, 0.2,
		0.6, 0.4,
	})
	// Doubly stochastic 3-state chain -> pi = [1/3, 1/3, 1/3].
	doubly3 := mustKernel(t, 3, []float64{
		0.5, 0.3, 0.2,
		0.2, 0.5, 0.3,
		0.3, 0.2, 0.5,
	})
	// Unichain with transient state 0 -> {1, 2}.
	unichainTransient := mustKernel(t, 3, []float64{
		0.5, 0.5, 0.0,
		0.0, 0.8, 0.2,
		0.0, 0.6, 0.4,
	})
	// Reducible 2-class chain.
	reducible := mustKernel(t, 2, []float64{
		1.0, 0.0,
		0.0, 1.0,
	})

	type testCase struct {
		name    string
		k       *markov.Kernel
		tol     float64
		wantErr error
		wantPi  markov.Distribution
	}
	cases := []testCase{
		{
			name:    "MUST return ErrNotStochastic for nil kernel",
			k:       nil,
			tol:     1e-9,
			wantErr: markov.ErrNotStochastic,
		},
		{
			name:    "MUST return ErrSingularOperator when chain has multiple recurrent classes",
			k:       reducible,
			tol:     1e-9,
			wantErr: markov.ErrSingularOperator,
		},
		{
			name:   "MUST compute exact GTH stationary distribution for asymmetric 2-state chain",
			k:      asym2,
			tol:    1e-9,
			wantPi: markov.Distribution{0.75, 0.25},
		},
		{
			name:   "MUST return uniform stationary distribution for doubly stochastic chain",
			k:      doubly3,
			tol:    1e-9,
			wantPi: markov.Distribution{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0},
		},
		{
			name:   "MUST assign zero stationary mass to transient states in a unichain",
			k:      unichainTransient,
			tol:    1e-9,
			wantPi: markov.Distribution{0.0, 0.75, 0.25},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := solve.Stationary(tc.k, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i := range tc.wantPi {
				if !markov.NearlyEqual(got[i], tc.wantPi[i], 1e-9) {
					t.Errorf("pi[%d] = %g, want %g", i, got[i], tc.wantPi[i])
				}
			}
		})
	}
}

func TestMustStationary(t *testing.T) {
	type testCase struct {
		name      string
		k         *markov.Kernel
		wantPanic bool
	}
	cases := []testCase{
		{
			name:      "MUST panic on reducible multi-class kernel",
			k:         mustKernel(t, 2, []float64{1, 0, 0, 1}),
			wantPanic: true,
		},
		{
			name:      "MUST NOT panic on unichain kernel",
			k:         mustKernel(t, 2, []float64{0.5, 0.5, 0.5, 0.5}),
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
			_ = solve.MustStationary(tc.k, 1e-9)
		})
	}
}

func TestFundamentalAndAbsorbing(t *testing.T) {
	ergodic := mustKernel(t, 3, []float64{
		0.5, 0.3, 0.2,
		0.2, 0.6, 0.2,
		0.3, 0.3, 0.4,
	})
	absorbing := mustKernel(t, 3, []float64{
		0.5, 0.3, 0.2,
		0.2, 0.4, 0.4,
		0.0, 0.0, 1.0,
	})

	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST satisfy two-sided inverse identity unit row sums and stationary annihilation for FundamentalMatrix",
			run: func(t *testing.T) {
				Z, err := solve.FundamentalMatrix(ergodic, 1e-9)
				if err != nil {
					t.Fatalf("FundamentalMatrix: %v", err)
				}
				pi := solve.MustStationary(ergodic, 1e-9)
				for i := 0; i < 3; i++ {
					rowSum := 0.0
					for j := 0; j < 3; j++ {
						rowSum += Z.At(i, j)
					}
					if !markov.NearlyEqual(rowSum, 1.0, 1e-9) {
						t.Errorf("Z row %d sum = %g, want 1", i, rowSum)
					}
				}
				for j := 0; j < 3; j++ {
					piZ := 0.0
					for i := 0; i < 3; i++ {
						piZ += pi[i] * Z.At(i, j)
					}
					if !markov.NearlyEqual(piZ, pi[j], 1e-9) {
						t.Errorf("(pi*Z)[%d] = %g, want pi[%d]=%g", j, piZ, j, pi[j])
					}
				}
				// Two-sided inverse identity: (I - P + 1*pi)*Z == I and Z*(I - P + 1*pi) == I.
				A := mat.NewDense(3, 3, nil)
				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						v := -ergodic.P.At(i, j) + pi[j]
						if i == j {
							v += 1.0
						}
						A.Set(i, j, v)
					}
				}
				var AZ, ZA mat.Dense
				AZ.Mul(A, Z)
				ZA.Mul(Z, A)
				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						want := 0.0
						if i == j {
							want = 1.0
						}
						if !markov.NearlyEqual(AZ.At(i, j), want, 1e-9) || !markov.NearlyEqual(ZA.At(i, j), want, 1e-9) {
							t.Errorf("FundamentalMatrix two-sided inverse failed at (%d,%d)", i, j)
						}
					}
				}
			},
		},
		{
			name: "MUST compute non-negative AbsorbingFundamental N two-sided inverse and match Schur excursion b*N*d = P_A - a",
			run: func(t *testing.T) {
				N, err := solve.AbsorbingFundamental(absorbing, []int{0, 1}, 1e-9)
				if err != nil {
					t.Fatalf("AbsorbingFundamental: %v", err)
				}
				IminusQ := mat.NewDense(2, 2, nil)
				for i := 0; i < 2; i++ {
					for j := 0; j < 2; j++ {
						if N.At(i, j) < 0 {
							t.Errorf("N[%d,%d] = %g < 0", i, j, N.At(i, j))
						}
						v := -absorbing.P.At(i, j)
						if i == j {
							v += 1.0
						}
						IminusQ.Set(i, j, v)
					}
				}
				var lhs, rhs mat.Dense
				lhs.Mul(IminusQ, N)
				rhs.Mul(N, IminusQ)
				for i := 0; i < 2; i++ {
					for j := 0; j < 2; j++ {
						want := 0.0
						if i == j {
							want = 1.0
						}
						if !markov.NearlyEqual(lhs.At(i, j), want, 1e-9) || !markov.NearlyEqual(rhs.At(i, j), want, 1e-9) {
							t.Errorf("AbsorbingFundamental two-sided inverse failed at (%d,%d)", i, j)
						}
					}
				}
				// Schur excursion equivalence on ergodic chain with A = {0}, T = A^c = {1, 2}.
				Nerg, err := solve.AbsorbingFundamental(ergodic, []int{1, 2}, 1e-9)
				if err != nil {
					t.Fatalf("AbsorbingFundamental(ergodic): %v", err)
				}
				traced := reduce.MustTrace(ergodic, []int{0}, 1e-9)
				a := ergodic.Submatrix([]int{0}, []int{0})
				b := ergodic.Submatrix([]int{0}, []int{1, 2})
				d := ergodic.Submatrix([]int{1, 2}, []int{0})
				var bN, bNd mat.Dense
				bN.Mul(b, Nerg)
				bNd.Mul(&bN, d)
				if !markov.NearlyEqual(bNd.At(0, 0), traced.P.At(0, 0)-a.At(0, 0), 1e-9) {
					t.Errorf("Schur excursion mismatch: bNd=%g vs P_A-a=%g", bNd.At(0, 0), traced.P.At(0, 0)-a.At(0, 0))
				}
			},
		},
		{
			name: "MUST return ErrSingularOperator when AbsorbingFundamental transient subset contains a closed recurrent trap",
			run: func(t *testing.T) {
				_, err := solve.AbsorbingFundamental(absorbing, []int{2}, 1e-9)
				if !errors.Is(err, markov.ErrSingularOperator) {
					t.Fatalf("got %v, want ErrSingularOperator", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestMFPTAndPerturbation(t *testing.T) {
	ergodic := mustKernel(t, 3, []float64{
		0.5, 0.3, 0.2,
		0.2, 0.6, 0.2,
		0.3, 0.3, 0.4,
	})

	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST satisfy Kemeny constant zero diagonal and first-step recurrence for MFPT",
			run: func(t *testing.T) {
				M, err := solve.MFPT(ergodic, 1e-9)
				if err != nil {
					t.Fatalf("MFPT: %v", err)
				}
				pi := solve.MustStationary(ergodic, 1e-9)
				Z, _ := solve.FundamentalMatrix(ergodic, 1e-9)
				wantKemeny := Z.At(0, 0) + Z.At(1, 1) + Z.At(2, 2) - 1.0

				for i := 0; i < 3; i++ {
					if M.At(i, i) != 0 {
						t.Errorf("M[%d,%d] = %g, want 0", i, i, M.At(i, i))
					}
					rowWeighted := 0.0
					for j := 0; j < 3; j++ {
						rowWeighted += M.At(i, j) * pi[j]
						if i != j {
							rec := 1.0
							for r := 0; r < 3; r++ {
								if r != j {
									rec += ergodic.P.At(i, r) * M.At(r, j)
								}
							}
							if !markov.NearlyEqual(M.At(i, j), rec, 1e-9) {
								t.Errorf("first-step recurrence failed at (%d,%d): got %g, want %g", i, j, M.At(i, j), rec)
							}
						}
					}
					if !markov.NearlyEqual(rowWeighted, wantKemeny, 1e-9) {
						t.Errorf("Kemeny constant row %d = %g, want %g", i, rowWeighted, wantKemeny)
					}
				}
			},
		},
		{
			name: "MUST match column of full MFPT in MFPTToTarget",
			run: func(t *testing.T) {
				M, _ := solve.MFPT(ergodic, 1e-9)
				for target := 0; target < 3; target++ {
					col, err := solve.MFPTToTarget(ergodic, target, 1e-9)
					if err != nil {
						t.Fatalf("MFPTToTarget(%d): %v", target, err)
					}
					for i := 0; i < 3; i++ {
						if !markov.NearlyEqual(col[i], M.At(i, target), 1e-9) {
							t.Errorf("MFPTToTarget(%d)[%d] = %g, want M[%d,%d] = %g", target, i, col[i], i, target, M.At(i, target))
						}
					}
				}
			},
		},
		{
			name: "MUST satisfy zero-perturbation identity and direct target shortcut monotonicity in MFPTDelta",
			run: func(t *testing.T) {
				d0, err := solve.MFPTDelta(ergodic, 0, 2, 2, 0.0, 1e-9)
				if err != nil || d0 != 0 {
					t.Fatalf("MFPTDelta(delta=0) = (%g, %v), want (0, nil)", d0, err)
				}
				dPos, err := solve.MFPTDelta(ergodic, 0, 2, 2, 0.5, 1e-9)
				if err != nil {
					t.Fatalf("MFPTDelta(delta=0.5): %v", err)
				}
				if dPos > 0 {
					t.Errorf("direct shortcut to target increased MFPT: delta=%g > 0", dPos)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestEntropyAndKLDivergence(t *testing.T) {
	perm3 := mustKernel(t, 3, []float64{
		0, 1, 0,
		0, 0, 1,
		1, 0, 0,
	})
	unif4 := mustKernel(t, 4, []float64{
		0.25, 0.25, 0.25, 0.25,
		0.25, 0.25, 0.25, 0.25,
		0.25, 0.25, 0.25, 0.25,
		0.25, 0.25, 0.25, 0.25,
	})
	lumpable4 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	})

	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST return zero entropy rate for deterministic permutation chain",
			run: func(t *testing.T) {
				h, err := solve.EntropyRate(perm3, 1e-9)
				if err != nil || !markov.NearlyEqual(h, 0, 1e-9) {
					t.Fatalf("EntropyRate(perm3) = (%g, %v), want (0, nil)", h, err)
				}
			},
		},
		{
			name: "MUST return log2(n) maximum entropy rate for uniform kernel",
			run: func(t *testing.T) {
				h, err := solve.EntropyRate(unif4, 1e-9)
				if err != nil || !markov.NearlyEqual(h, 2.0, 1e-9) {
					t.Fatalf("EntropyRate(unif4) = (%g, %v), want (2.0, nil)", h, err)
				}
			},
		},
		{
			name: "MUST satisfy Lumpable Data Processing Inequality H(Lump(P,B)) <= H(P) when strongly lumpable",
			run: func(t *testing.T) {
				p := markov.Partition{{0, 1}, {2, 3}}
				lumped := reduce.MustLump(lumpable4, p, nil, nil, 1e-9)
				hFine, _ := solve.EntropyRate(lumpable4, 1e-9)
				hCoarse, _ := solve.EntropyRate(lumped, 1e-9)
				if hCoarse > hFine+1e-9 {
					t.Errorf("DPI violated: H(lumped)=%g > H(fine)=%g", hCoarse, hFine)
				}
			},
		},
		{
			name: "MUST satisfy Gibbs inequality and identity of indiscernibles for KLDivergenceRate",
			run: func(t *testing.T) {
				dSelf, err := solve.KLDivergenceRate(lumpable4, lumpable4, 1e-9)
				if err != nil || !markov.NearlyEqual(dSelf, 0, 1e-9) {
					t.Fatalf("KLDivergenceRate(P, P) = (%g, %v), want 0", dSelf, err)
				}
				dOther, err := solve.KLDivergenceRate(lumpable4, unif4, 1e-9)
				if err != nil || dOther <= 0 {
					t.Fatalf("KLDivergenceRate(lumpable4, unif4) = (%g, %v), want > 0", dOther, err)
				}
				// Support mismatch: lumpable4 has P[0,1]=0.3 > 0 while perm3 has Q[0,0]=0 -> +Inf.
				qZero := mustKernel(t, 4, []float64{
					0, 1, 0, 0,
					0, 0, 1, 0,
					0, 0, 0, 1,
					1, 0, 0, 0,
				})
				dInf, err := solve.KLDivergenceRate(lumpable4, qZero, 1e-9)
				if err != nil || !math.IsInf(dInf, 1) {
					t.Fatalf("KLDivergenceRate with structural zero = (%g, %v), want (+Inf, nil)", dInf, err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestSolveProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 5).Draw(t, "n")
		k := randomErgodicKernel(n).Draw(t, "k")
		q := randomErgodicKernel(n).Draw(t, "q")
		tol := 1e-9

		// 1. Stationary left-eigenvector fixed point & multi-step L1 invariance.
		pi, err := solve.Stationary(k, tol)
		if err != nil {
			t.Fatalf("Stationary: %v", err)
		}
		var l1Diff float64
		for j := 0; j < n; j++ {
			var sum float64
			for i := 0; i < n; i++ {
				sum += pi[i] * k.P.At(i, j)
			}
			l1Diff += math.Abs(sum - pi[j])
		}
		if l1Diff > 1e-7 {
			t.Errorf("||pi*P - pi||_1 = %g > 1e-7", l1Diff)
		}

		// Multi-step invariance: ||pi * P^t - pi||_1 <= t*tol (within numerical precision).
		for _, steps := range []int{2, 5} {
			var Pt mat.Dense
			Pt.Pow(k.P, steps)
			var stepDiff float64
			for j := 0; j < n; j++ {
				var sum float64
				for i := 0; i < n; i++ {
					sum += pi[i] * Pt.At(i, j)
				}
				stepDiff += math.Abs(sum - pi[j])
			}
			if stepDiff > float64(steps)*1e-7 {
				t.Errorf("steps=%d: ||pi*P^t - pi||_1 = %g", steps, stepDiff)
			}
		}

		// ClassifyStates closedness of recurrent classes: for C in Recurrent, i in C, j not in C => P[i,j] <= tol.
		sc, err := solve.ClassifyStates(k, tol)
		if err != nil {
			t.Fatalf("ClassifyStates: %v", err)
		}
		for _, rec := range sc.Recurrent {
			inRec := make(map[int]bool, len(rec))
			for _, s := range rec {
				inRec[s] = true
			}
			for _, i := range rec {
				for j := 0; j < n; j++ {
					if !inRec[j] && k.P.At(i, j) > tol {
						t.Errorf("recurrent class not closed: P[%d,%d] = %g > tol", i, j, k.P.At(i, j))
					}
				}
			}
		}

		// 2. Trace restriction compatibility: Restrict(pi, A) * P_A == Restrict(pi, A).
		m := rapid.IntRange(1, n-1).Draw(t, "m")
		subset := make([]int, m)
		for i := range subset {
			subset[i] = i
		}
		traced, err := reduce.Trace(k, subset, tol)
		if err == nil {
			piA, err := pi.Restrict(subset, tol)
			if err == nil {
				for j := 0; j < m; j++ {
					var s float64
					for i := 0; i < m; i++ {
						s += piA[i] * traced.P.At(i, j)
					}
					if math.Abs(s-piA[j]) > 1e-6 {
						t.Errorf("trace stationary restriction mismatch at %d: got %g, want %g", j, s, piA[j])
					}
				}
			}
		}

		// 3. Entropy bounds 0 <= H(P) <= log2(n) and Gibbs inequality D_KL(P || Q) >= 0.
		h, err := solve.EntropyRate(k, tol)
		if err != nil {
			t.Fatalf("EntropyRate: %v", err)
		}
		if h < -tol || h > math.Log2(float64(n))+1e-7 {
			t.Errorf("EntropyRate %g outside [0, log2(%d)]", h, n)
		}
		dkl, err := solve.KLDivergenceRate(k, q, tol)
		if err != nil {
			t.Fatalf("KLDivergenceRate: %v", err)
		}
		if dkl < -tol {
			t.Errorf("Gibbs inequality violated: D_KL = %g < 0", dkl)
		}
	})
}

// Property: H(Lump(P,B)) <= H(P) + 1e-7 under strong lumpability (data-processing inequality).
func TestProperty_4_2_EntropyRateDPI(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(4, 8).Draw(t, "n")
		m := rapid.IntRange(2, 3).Draw(t, "m")
		if m > n {
			t.Skip()
		}
		// Build strongly lumpable kernel inline (same construction as lump_test.go).
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
		blockSize := make([]int, m)
		for s := 0; s < n; s++ {
			blockSize[s%m]++
		}
		p := make(markov.Partition, m)
		for s := 0; s < n; s++ {
			p[s%m] = append(p[s%m], s)
		}
		pData := make([]float64, n*n)
		for s := 0; s < n; s++ {
			bi := s % m
			for tt := 0; tt < n; tt++ {
				bj := tt % m
				pData[s*n+tt] = qData[bi*m+bj] / float64(blockSize[bj])
			}
		}
		k, err := markov.NewKernel(mat.NewDense(n, n, pData), nil)
		if err != nil {
			t.Skip()
		}
		hP, err := solve.EntropyRate(k, 1e-9)
		if err != nil {
			t.Skip()
		}
		lumped, err := reduce.Lump(k, p, nil, nil, 1e-9)
		if err != nil {
			t.Skip()
		}
		hLumped, err := solve.EntropyRate(lumped, 1e-9)
		if err != nil {
			t.Skip()
		}
		if hLumped > hP+1e-7 {
			t.Errorf("entropy DPI violated: H(Lump)=%g > H(P)=%g", hLumped, hP)
		}
	})
}
