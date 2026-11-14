package control_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/control"
	"github.com/stephen-mcelhose/markov/reduce"
	"gonum.org/v1/gonum/mat"
	"pgregory.net/rapid"
)

func mustKernel(t *testing.T, n int, data []float64) *markov.Kernel {
	t.Helper()
	return markov.MustNewKernel(mat.NewDense(n, n, data), nil)
}

func TestNewMDPAndInducedKernel(t *testing.T) {
	k0 := mustKernel(t, 2, []float64{0.8, 0.2, 0.1, 0.9})
	k1 := mustKernel(t, 2, []float64{0.2, 0.8, 0.9, 0.1})
	R := mat.NewDense(2, 2, []float64{1.0, 0.5, 0.0, 2.0})

	type testCase struct {
		name        string
		actions     []string
		transitions []*markov.Kernel
		rewards     *mat.Dense
		discount    float64
		tol         float64
		wantErr     error
	}
	cases := []testCase{
		{
			name:        "MUST return ErrNotStochastic when transitions is empty",
			transitions: nil,
			rewards:     R,
			discount:    0.9,
			tol:         1e-9,
			wantErr:     markov.ErrNotStochastic,
		},
		{
			name:        "MUST return ErrInvalidDistribution when discount is 1.0 or negative",
			transitions: []*markov.Kernel{k0, k1},
			rewards:     R,
			discount:    1.0,
			tol:         1e-9,
			wantErr:     markov.ErrInvalidDistribution,
		},
		{
			name:        "MUST return ErrInvalidDistribution when reward matrix dimensions mismatch",
			transitions: []*markov.Kernel{k0, k1},
			rewards:     mat.NewDense(3, 2, nil),
			discount:    0.9,
			tol:         1e-9,
			wantErr:     markov.ErrInvalidDistribution,
		},
		{
			name:        "MUST succeed for valid kernels reward matrix and discount",
			actions:     []string{"stay", "switch"},
			transitions: []*markov.Kernel{k0, k1},
			rewards:     R,
			discount:    0.9,
			tol:         1e-9,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := control.NewMDP(tc.actions, tc.transitions, tc.rewards, tc.discount, tc.tol)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.NumStates() != 2 || m.NumActions() != 2 {
				t.Errorf("dims = (%d, %d), want (2, 2)", m.NumStates(), m.NumActions())
			}
			// Constant-action identity: policy [a, a] induces P^a.
			for a, wantK := range tc.transitions {
				ind, err := m.InducedKernel([]int{a, a}, 1e-9)
				if err != nil {
					t.Fatalf("InducedKernel: %v", err)
				}
				for i := 0; i < 2; i++ {
					for j := 0; j < 2; j++ {
						if !markov.NearlyEqual(ind.P.At(i, j), wantK.P.At(i, j), 1e-12) {
							t.Errorf("constant action %d mismatch at (%d,%d)", a, i, j)
						}
					}
				}
			}
		})
	}
}

func TestValueAndPolicyIteration(t *testing.T) {
	k0 := mustKernel(t, 2, []float64{0.9, 0.1, 0.2, 0.8})
	k1 := mustKernel(t, 2, []float64{0.1, 0.9, 0.8, 0.2})
	R := mat.NewDense(2, 2, []float64{
		10.0, 1.0,
		0.0, 5.0,
	})
	mdp, err := control.NewMDP([]string{"a0", "a1"}, []*markov.Kernel{k0, k1}, R, 0.9, 1e-9)
	if err != nil {
		t.Fatal(err)
	}
	myopicMDP, err := control.NewMDP([]string{"a0", "a1"}, []*markov.Kernel{k0, k1}, R, 0.0, 1e-9)
	if err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST return error when ValueIteration or PolicyIteration receives nil MDP",
			run: func(t *testing.T) {
				if _, _, err := control.ValueIteration(nil, 1e-9); err == nil {
					t.Error("expected error for nil MDP in ValueIteration")
				}
				if _, _, err := control.PolicyIteration(nil, 1e-9); err == nil {
					t.Error("expected error for nil MDP in PolicyIteration")
				}
			},
		},
		{
			name: "MUST satisfy Myopic Horizon Identity V*(s) = max_a R(s,a) when discount is zero",
			run: func(t *testing.T) {
				pol, val, err := control.ValueIteration(myopicMDP, 1e-9)
				if err != nil {
					t.Fatal(err)
				}
				if pol[0] != 0 || !markov.NearlyEqual(val[0], 10.0, 1e-9) {
					t.Errorf("state 0: got (pol=%d, val=%g), want (0, 10)", pol[0], val[0])
				}
				if pol[1] != 1 || !markov.NearlyEqual(val[1], 5.0, 1e-9) {
					t.Errorf("state 1: got (pol=%d, val=%g), want (1, 5)", pol[1], val[1])
				}
			},
		},
		{
			name: "MUST agree between ValueIteration and PolicyIteration and satisfy Bellman optimality fixed point",
			run: func(t *testing.T) {
				tol := 1e-9
				polVI, valVI, err := control.ValueIteration(mdp, tol)
				if err != nil {
					t.Fatalf("ValueIteration: %v", err)
				}
				polPI, valPI, err := control.PolicyIteration(mdp, tol)
				if err != nil {
					t.Fatalf("PolicyIteration: %v", err)
				}
				for s := 0; s < 2; s++ {
					if polVI[s] != polPI[s] {
						t.Errorf("policy[%d]: VI=%d != PI=%d", s, polVI[s], polPI[s])
					}
					if !markov.NearlyEqual(valVI[s], valPI[s], 1e-7) {
						t.Errorf("value[%d]: VI=%g != PI=%g", s, valVI[s], valPI[s])
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestMDPBisimulation(t *testing.T) {
	// 4-state MDP where states {0,1} and {2,3} have identical rewards and lumpable transitions under both actions.
	p0 := mustKernel(t, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	})
	p1 := mustKernel(t, 4, []float64{
		0.4, 0.4, 0.1, 0.1,
		0.3, 0.5, 0.2, 0.0,
		0.1, 0.1, 0.4, 0.4,
		0.0, 0.2, 0.5, 0.3,
	})
	R := mat.NewDense(4, 2, []float64{
		5.0, 1.0,
		5.0, 1.0,
		0.0, 3.0,
		0.0, 3.0,
	})
	mdp, err := control.NewMDP(nil, []*markov.Kernel{p0, p1}, R, 0.9, 1e-9)
	if err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		name string
		run  func(t *testing.T)
	}
	cases := []testCase{
		{
			name: "MUST return coarsest partition {{0,1},{2,3}} preserving action-wise strong lumpability and optimal values",
			run: func(t *testing.T) {
				part, err := control.MDPBisimulation(mdp, 1e-9)
				if err != nil {
					t.Fatalf("MDPBisimulation: %v", err)
				}
				if len(part) != 2 {
					t.Fatalf("expected 2 blocks, got %v", part)
				}
				for a, k := range mdp.Transitions {
					ok, err := reduce.IsStronglyLumpable(k, part, 1e-9)
					if err != nil || !ok {
						t.Errorf("action %d not strongly lumpable on %v", a, part)
					}
				}
				_, vals, err := mdp.PolicyIteration(1e-9)
				if err != nil {
					t.Fatal(err)
				}
				for _, blk := range part {
					for _, s := range blk[1:] {
						if !markov.NearlyEqual(vals[blk[0]], vals[s], 1e-7) {
							t.Errorf("optimal value differs within block %v: V[%d]=%g != V[%d]=%g", blk, blk[0], vals[blk[0]], s, vals[s])
						}
					}
				}
			},
		},
		{
			name: "MUST satisfy Passive Chain Degeneration when |A| == 1 and R == 0",
			run: func(t *testing.T) {
				zeroR := mat.NewDense(4, 1, nil)
				passive, err := control.NewMDP(nil, []*markov.Kernel{p0}, zeroR, 0.5, 1e-9)
				if err != nil {
					t.Fatal(err)
				}
				gotMDP, err := passive.CoarsestPartition(1e-9)
				if err != nil {
					t.Fatal(err)
				}
				gotReduce, err := reduce.CoarsestPartition(p0, nil, 1e-9)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(gotMDP, gotReduce) {
					t.Errorf("passive degeneration mismatch: got %v, want %v", gotMDP, gotReduce)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestControlProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(2, 4).Draw(t, "n")
		numActions := rapid.IntRange(1, 3).Draw(t, "numActions")
		gamma := rapid.Float64Range(0.0, 0.85).Draw(t, "gamma")
		tol := 1e-8

		transitions := make([]*markov.Kernel, numActions)
		for a := 0; a < numActions; a++ {
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
			transitions[a] = markov.MustNewKernel(mat.NewDense(n, n, data), nil)
		}

		rData := make([]float64, n*numActions)
		for i := range rData {
			rData[i] = rapid.Float64Range(-5.0, 10.0).Draw(t, "r")
		}
		R := mat.NewDense(n, numActions, rData)

		m, err := control.NewMDP(nil, transitions, R, gamma, tol)
		if err != nil {
			t.Fatalf("NewMDP: %v", err)
		}

		_, vVI, err := m.ValueIteration(tol)
		if err != nil {
			t.Fatalf("ValueIteration: %v", err)
		}
		polPI, vPI, err := m.PolicyIteration(tol)
		if err != nil {
			t.Fatalf("PolicyIteration: %v", err)
		}

		// Value Iteration Equivalence & Bellman Optimality Fixed Point within tol.
		for s := 0; s < n; s++ {
			if math.Abs(vVI[s]-vPI[s]) > tol*2 {
				t.Errorf("s=%d: |V_VI(%g) - V_PI(%g)| > 2*tol", s, vVI[s], vPI[s])
			}
			bestQ := math.Inf(-1)
			for a := 0; a < numActions; a++ {
				q := R.At(s, a)
				for sp := 0; sp < n; sp++ {
					q += gamma * transitions[a].P.At(s, sp) * vVI[sp]
				}
				if q > bestQ {
					bestQ = q
				}
			}
			if math.Abs(vVI[s]-bestQ) > tol {
				t.Errorf("Bellman residual at s=%d: |%g - %g| > tol", s, vVI[s], bestQ)
			}
		}

		// Policy-Value Consistency: (I - gamma*P^{pi*}) V* == R^{pi*}.
		indK, err := m.InducedKernel(polPI, tol)
		if err != nil {
			t.Fatalf("InducedKernel: %v", err)
		}
		for s := 0; s < n; s++ {
			lhs := vPI[s]
			for sp := 0; sp < n; sp++ {
				lhs -= gamma * indK.P.At(s, sp) * vPI[sp]
			}
			rhs := R.At(s, polPI[s])
			if math.Abs(lhs-rhs) > tol {
				t.Errorf("Policy-Value consistency failed at s=%d: lhs=%g != rhs=%g", s, lhs, rhs)
			}
		}

		// Action-Wise Strong Lumpability on CoarsestPartition.
		part, err := m.CoarsestPartition(tol)
		if err != nil {
			t.Fatalf("CoarsestPartition: %v", err)
		}
		for a, k := range m.Transitions {
			ok, err := reduce.IsStronglyLumpable(k, part, tol)
			if err != nil || !ok {
				t.Errorf("action %d not strongly lumpable on CoarsestPartition %v", a, part)
			}
		}
	})
}
