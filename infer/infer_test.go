package infer_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/infer"
	"pgregory.net/rapid"
)

func TestNewEstimator(t *testing.T) {
	type testCase struct {
		name     string
		n        int
		names    []string
		prior    float64
		halflife float64
		wantErr  error
	}
	cases := []testCase{
		{
			name:    "MUST return error when n is less than 1",
			n:       0,
			prior:   1.0,
			wantErr: markov.ErrNotStochastic,
		},
		{
			name:    "MUST return error when prior is negative",
			n:       2,
			prior:   -0.1,
			wantErr: markov.ErrInvalidDistribution,
		},
		{
			name:     "MUST return error when halflife is negative",
			n:        2,
			prior:    1.0,
			halflife: -1.0,
			wantErr:  markov.ErrInvalidDistribution,
		},
		{
			name:    "MUST return error when names length mismatches n",
			n:       2,
			names:   []string{"only_one"},
			prior:   1.0,
			wantErr: markov.ErrNotStochastic,
		},
		{
			name:     "MUST succeed for valid parameters with zero or positive halflife",
			n:        3,
			names:    []string{"s0", "s1", "s2"},
			prior:    0.5,
			halflife: 10.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			est, err := infer.NewEstimator(tc.n, tc.names, tc.prior, tc.halflife)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if est == nil {
				t.Fatal("expected non-nil Estimator")
			}
		})
	}
}

func TestEstimatorObserveAndKernel(t *testing.T) {
	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST return uniform 1/n kernel when counts are zero and prior is positive",
			run: func(t *testing.T) {
				est, err := infer.NewEstimator(3, nil, 1.0, 0)
				if err != nil {
					t.Fatal(err)
				}
				k, err := est.Kernel(1e-9)
				if err != nil {
					t.Fatalf("Kernel: %v", err)
				}
				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						if !markov.NearlyEqual(k.P.At(i, j), 1.0/3.0, 1e-9) {
							t.Errorf("P[%d,%d] = %g, want 1/3", i, j, k.P.At(i, j))
						}
					}
				}
			},
		},
		{
			name: "MUST return ErrNotStochastic when a row has zero count and prior is zero",
			run: func(t *testing.T) {
				est, _ := infer.NewEstimator(2, nil, 0, 0)
				_ = est.Observe(0, 1)
				_, err := est.Kernel(1e-9)
				if !errors.Is(err, markov.ErrNotStochastic) {
					t.Fatalf("got %v, want ErrNotStochastic", err)
				}
			},
		},
		{
			name: "MUST satisfy sequential composition for ObserveSequence with overlapping boundary state",
			run: func(t *testing.T) {
				seq := []int{0, 1, 0, 1, 1, 0}
				eFull, _ := infer.NewEstimator(2, nil, 0.5, 4.0)
				if err := eFull.ObserveSequence(seq); err != nil {
					t.Fatal(err)
				}
				eSplit, _ := infer.NewEstimator(2, nil, 0.5, 4.0)
				if err := eSplit.ObserveSequence(seq[:4]); err != nil {
					t.Fatal(err)
				}
				if err := eSplit.ObserveSequence(seq[3:]); err != nil {
					t.Fatal(err)
				}
				kFull, _ := eFull.Kernel(1e-9)
				kSplit, _ := eSplit.Kernel(1e-9)
				for i := 0; i < 2; i++ {
					for j := 0; j < 2; j++ {
						if !markov.NearlyEqual(kFull.P.At(i, j), kSplit.P.At(i, j), 1e-12) {
							t.Errorf("mismatch at (%d,%d): full=%g vs split=%g", i, j, kFull.P.At(i, j), kSplit.P.At(i, j))
						}
					}
				}
			},
		},
		{
			name: "MUST satisfy scale invariance when halflife is zero and prior is zero",
			run: func(t *testing.T) {
				e1, _ := infer.NewEstimator(2, nil, 0, 0)
				e2, _ := infer.NewEstimator(2, nil, 0, 0)
				pairs := [][2]int{{0, 0}, {0, 1}, {0, 1}, {1, 0}, {1, 1}}
				for _, p := range pairs {
					_ = e1.Observe(p[0], p[1])
					for c := 0; c < 5; c++ {
						_ = e2.Observe(p[0], p[1])
					}
				}
				k1, err := e1.Kernel(1e-9)
				if err != nil {
					t.Fatal(err)
				}
				k2, err := e2.Kernel(1e-9)
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 2; i++ {
					for j := 0; j < 2; j++ {
						if !markov.NearlyEqual(k1.P.At(i, j), k2.P.At(i, j), 1e-12) {
							t.Errorf("scale invariance violated at (%d,%d): %g != %g", i, j, k1.P.At(i, j), k2.P.At(i, j))
						}
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestFromTransitionsAndFromSequence(t *testing.T) {
	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST return error when FromTransitions receives odd-length slice",
			run: func(t *testing.T) {
				_, err := infer.FromTransitions([]int{0, 1, 0}, 2, 1.0, 1e-9)
				if err == nil {
					t.Fatal("expected error for odd pairs slice")
				}
			},
		},
		{
			name: "MUST match Estimator parity in FromTransitions and FromSequence",
			run: func(t *testing.T) {
				seq := []int{0, 1, 1, 0, 0, 1}
				pairs := []int{0, 1, 1, 1, 1, 0, 0, 0, 0, 1}
				kPairs, err := infer.FromTransitions(pairs, 2, 0.5, 1e-9)
				if err != nil {
					t.Fatalf("FromTransitions: %v", err)
				}
				kSeq, err := infer.FromSequence(seq, 2, 0.5, 0, 1e-9)
				if err != nil {
					t.Fatalf("FromSequence: %v", err)
				}
				for i := 0; i < 2; i++ {
					for j := 0; j < 2; j++ {
						if !markov.NearlyEqual(kPairs.P.At(i, j), kSeq.P.At(i, j), 1e-12) {
							t.Errorf("parity mismatch at (%d,%d): %g != %g", i, j, kPairs.P.At(i, j), kSeq.P.At(i, j))
						}
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestMarkovOrderTest(t *testing.T) {
	// Construct an exact empirical first-order sequence where P_hat[i,j,k] == P_hat[j,k] for all i,j,k:
	// Every pair (i,j) is followed by k=0 once and k=1 once.
	// De Bruijn sequence of order 3 over {0,1}: 0,0,0,1,0,1,1,1,0,0 (length 10, 8 triples, each triple in {0,1}^3 appears once!).
	exactFirstOrder := []int{0, 0, 0, 1, 0, 1, 1, 1, 0, 0}

	// Deterministic 2nd-order sequence: (0,0)->1, (0,1)->1, (1,1)->0, (1,0)->0 (period-4 cycle 0,0,1,1,0,0,1,1...).
	// Given j=0: if i=0 then k=1 (100%), if i=1 then k=0 (100%)! Strongly 2nd-order!
	secondOrder := make([]int, 0, 80)
	for rep := 0; rep < 20; rep++ {
		secondOrder = append(secondOrder, 0, 0, 1, 1)
	}

	type testCase struct {
		name    string
		states  []int
		n       int
		wantErr bool
		check   func(t *testing.T, chi2, pVal float64)
	}
	cases := []testCase{
		{
			name:    "MUST return error when sequence has fewer than 3 states",
			states:  []int{0, 1},
			n:       2,
			wantErr: true,
		},
		{
			name:    "MUST return error when n is less than 2",
			states:  []int{0, 0, 0},
			n:       1,
			wantErr: true,
		},
		{
			name:    "MUST return error when state index is out of range",
			states:  []int{0, 1, 5},
			n:       2,
			wantErr: true,
		},
		{
			name:   "MUST return G2 == 0 and pValue == 1 when empirical transitions are strictly first-order",
			states: exactFirstOrder,
			n:      2,
			check: func(t *testing.T, chi2, pVal float64) {
				if !markov.NearlyEqual(chi2, 0, 1e-12) || !markov.NearlyEqual(pVal, 1.0, 1e-12) {
					t.Errorf("got (chi2=%g, p=%g), want (0, 1)", chi2, pVal)
				}
			},
		},
		{
			name:   "MUST detect strong second-order dependence with large G2 and near-zero pValue",
			states: secondOrder,
			n:      2,
			check: func(t *testing.T, chi2, pVal float64) {
				if chi2 <= 20 || pVal >= 1e-4 {
					t.Errorf("expected strong rejection of H0, got chi2=%g, p=%g", chi2, pVal)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chi2, pVal, err := infer.MarkovOrderTest(tc.states, tc.n)
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
				tc.check(t, chi2, pVal)
			}
		})
	}
}

func TestInferProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 5).Draw(t, "n")
		seqLen := rapid.IntRange(3, 30).Draw(t, "seqLen")
		states := make([]int, seqLen)
		for i := range states {
			states[i] = rapid.IntRange(0, n-1).Draw(t, "s")
		}
		prior := rapid.Float64Range(0.01, 2.0).Draw(t, "prior")
		halflife := rapid.Float64Range(0.0, 10.0).Draw(t, "halflife")

		k, err := infer.FromSequence(states, n, prior, halflife, 1e-9)
		if err != nil {
			t.Fatalf("FromSequence: %v", err)
		}
		if err := k.Validate(1e-9); err != nil {
			t.Errorf("estimated kernel not row-stochastic: %v", err)
		}

		chi2, pVal, err := infer.MarkovOrderTest(states, n)
		if err != nil {
			t.Fatalf("MarkovOrderTest: %v", err)
		}
		if chi2 < 0 || math.IsNaN(chi2) {
			t.Errorf("invalid chi2 = %g", chi2)
		}
		if pVal < 0 || pVal > 1 || math.IsNaN(pVal) {
			t.Errorf("pValue = %g outside [0, 1]", pVal)
		}

		// Uniform prior identity on fresh estimator.
		fresh, err := infer.NewEstimator(n, nil, prior, halflife)
		if err != nil {
			t.Fatal(err)
		}
		kUnif, err := fresh.Kernel(0)
		if err != nil {
			t.Fatalf("fresh Kernel(tol=0): %v", err)
		}
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if !markov.NearlyEqual(kUnif.P.At(i, j), 1.0/float64(n), 1e-9) {
					t.Errorf("uniform prior identity failed at (%d,%d)", i, j)
				}
			}
		}

		// Estimator parity between FromTransitions and FromSequence (halflife == 0).
		pairs := make([]int, 0, 2*(len(states)-1))
		for tIdx := 0; tIdx+1 < len(states); tIdx++ {
			pairs = append(pairs, states[tIdx], states[tIdx+1])
		}
		kTrans, err := infer.FromTransitions(pairs, n, prior, 1e-9)
		kSeq0, err2 := infer.FromSequence(states, n, prior, 0, 1e-9)
		if err != nil || err2 != nil {
			t.Fatalf("FromTransitions/FromSequence: %v, %v", err, err2)
		}
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if !markov.NearlyEqual(kTrans.P.At(i, j), kSeq0.P.At(i, j), 1e-12) {
					t.Errorf("Estimator parity failed at (%d,%d)", i, j)
				}
			}
		}
	})
}
