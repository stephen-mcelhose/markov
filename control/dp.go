package control

import (
	"fmt"
	"math"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// ValueIteration computes an optimal deterministic policy π* and optimal value
// function V* via Bellman optimality fixed-point iteration:
//
//	V_{t+1}(s) = max_{a ∈ A} [ R(s, a) + γ Σ_{s'} Pᵃ[s, s'] V_t(s') ]
//
// Iteration terminates when the Bellman residual ‖T*V - V‖∞ ≤ tol.
// Returns an error if m is nil, tol ≤ 0, or iteration fails to converge.
func (m *MDP) ValueIteration(tol float64) (policy []int, values []float64, err error) {
	if err := validateMDP(m); err != nil {
		return nil, nil, err
	}
	if tol <= 0 || math.IsNaN(tol) || math.IsInf(tol, 0) {
		return nil, nil, fmt.Errorf("%w: tolerance %g must be finite and > 0", markov.ErrNotStochastic, tol)
	}

	n := m.NumStates()
	V := make([]float64, n)
	nextV := make([]float64, n)
	pol := make([]int, n)
	stopDiff := contractionStoppingThreshold(tol, m.Discount)

	const maxIter = 100000
	converged := false
	for iter := 0; iter < maxIter; iter++ {
		maxDiff := m.bellmanOptimalitySweep(V, nextV, pol)
		copy(V, nextV)
		if m.Discount == 0 || maxDiff <= stopDiff {
			converged = true
			break
		}
	}
	if !converged {
		return nil, nil, fmt.Errorf("%w: value iteration did not converge within %d iterations", markov.ErrSingularOperator, maxIter)
	}

	m.extractGreedyPolicy(V, pol)
	return pol, V, nil
}

// PolicyIteration computes an optimal deterministic policy π* and optimal value
// function V* via Howard's exact policy evaluation and policy improvement algorithm:
//
//	V^{π_t}    = (I - γ P^{π_t})⁻¹ R^{π_t}
//	π_{t+1}(s) = argmax_{a ∈ A} [ R(s, a) + γ Σ_{s'} Pᵃ[s, s'] V^{π_t}(s') ]
//
// For γ ∈ [0, 1), (I - γ P^{π_t}) is guaranteed invertible by the Neumann series
// theorem (ρ(γ P^{π_t}) = γ < 1); returns [markov.ErrSingularOperator] only on
// numerical ill-conditioning as γ → 1⁻ or non-convergence.
func (m *MDP) PolicyIteration(tol float64) (policy []int, values []float64, err error) {
	if err := validateMDP(m); err != nil {
		return nil, nil, err
	}
	if err := validateNonNegativeTol(tol); err != nil {
		return nil, nil, err
	}

	n := m.NumStates()
	pol := make([]int, n)
	improveEps := policyImprovementThreshold(tol, m.Discount)

	const maxIter = 10000
	for iter := 0; iter < maxIter; iter++ {
		V, err := m.evaluatePolicyValues(pol)
		if err != nil {
			return nil, nil, err
		}
		if m.improvePolicyGreedily(pol, V, improveEps) {
			return pol, V, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: policy iteration did not converge within %d iterations", markov.ErrSingularOperator, maxIter)
}

// ValueIteration is the top-level form of [MDP.ValueIteration].
func ValueIteration(m *MDP, tol float64) (policy []int, values []float64, err error) {
	if m == nil {
		return nil, nil, fmt.Errorf("%w: nil MDP", markov.ErrNotStochastic)
	}
	return m.ValueIteration(tol)
}

// PolicyIteration is the top-level form of [MDP.PolicyIteration].
func PolicyIteration(m *MDP, tol float64) (policy []int, values []float64, err error) {
	if m == nil {
		return nil, nil, fmt.Errorf("%w: nil MDP", markov.ErrNotStochastic)
	}
	return m.PolicyIteration(tol)
}

// ── internal helpers: Bellman backups & policy evaluation ─────────────────────

// contractionStoppingThreshold scales tol by (1 - γ) so that stopping when
// ‖V_{t+1} - V_t‖∞ ≤ tol·(1 - γ) guarantees ‖V* - V_{t+1}‖∞ ≤ tol.
func contractionStoppingThreshold(tol, gamma float64) float64 {
	if d := tol * (1.0 - gamma); d > 0 {
		return d
	}
	return tol
}

// policyImprovementThreshold computes a small anti-cycling threshold scaled by
// (1 - γ) and capped at 1e-12 so one-step Bellman advantages > 1e-12 are always
// taken and ‖V*_PI - V*_VI‖∞ ≤ tol.
func policyImprovementThreshold(tol, gamma float64) float64 {
	eps := 1e-12
	if tol > 0 {
		if scaled := 0.5 * tol * (1.0 - gamma); scaled < eps {
			eps = scaled
		}
	}
	return eps
}

// bellmanOptimalitySweep applies the Bellman optimality operator T* across all
// states s ∈ S:
//
//	nextV[s] = max_{a ∈ A} Q(s, a, V)
//
// and returns the ℓ∞ Bellman residual max_s |nextV[s] - V[s]|.
func (m *MDP) bellmanOptimalitySweep(V, nextV []float64, pol []int) float64 {
	maxDiff := 0.0
	for s := range V {
		bestAct, bestVal := m.bellmanOptimalBackup(s, V)
		nextV[s] = bestVal
		pol[s] = bestAct
		if diff := math.Abs(bestVal - V[s]); diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

// extractGreedyPolicy populates pol[s] = argmax_{a ∈ A} Q(s, a, V) for all states s.
func (m *MDP) extractGreedyPolicy(V []float64, pol []int) {
	for s := range V {
		pol[s], _ = m.bellmanOptimalBackup(s, V)
	}
}

// bellmanOptimalBackup computes the greedy action and Bellman optimality backup
// at state s given value vector V:
//
//	bestAct = argmax_{a ∈ A} Q(s, a, V)
//	bestVal = max_{a ∈ A} Q(s, a, V)
func (m *MDP) bellmanOptimalBackup(s int, V []float64) (bestAct int, bestVal float64) {
	bestVal = math.Inf(-1)
	numActions := m.NumActions()
	for a := 0; a < numActions; a++ {
		if q := m.bellmanActionValue(s, a, V); q > bestVal {
			bestVal = q
			bestAct = a
		}
	}
	return bestAct, bestVal
}

// bellmanActionValue computes the state-action value Q(s, a, V):
//
//	Q(s, a, V) = R(s, a) + γ Σ_{s' ∈ S} Pᵃ[s, s'] · V[s']
func (m *MDP) bellmanActionValue(s, a int, V []float64) float64 {
	q := m.Rewards.At(s, a)
	if m.Discount == 0 {
		return q
	}
	row := m.Transitions[a].P.RawRowView(s)
	var expectedNext float64
	for sp, prob := range row {
		expectedNext += prob * V[sp]
	}
	return q + m.Discount*expectedNext
}

// evaluatePolicyValues solves the linear Bellman evaluation equation for a
// fixed deterministic policy π = pol:
//
//	(I - γ P^π) V^π = R^π
func (m *MDP) evaluatePolicyValues(pol []int) ([]float64, error) {
	n := m.NumStates()
	gamma := m.Discount
	A := mat.NewDense(n, n, nil)
	b := mat.NewDense(n, 1, nil)
	for s, a := range pol {
		b.Set(s, 0, m.Rewards.At(s, a))
		row := m.Transitions[a].P.RawRowView(s)
		aRow := A.RawRowView(s)
		for sp, prob := range row {
			aRow[sp] = -gamma * prob
		}
		aRow[s] += 1.0
	}
	var vVec mat.Dense
	if err := vVec.Solve(A, b); err != nil {
		return nil, fmt.Errorf("%w: policy evaluation solve failed: %v", markov.ErrSingularOperator, err)
	}
	V := make([]float64, n)
	for s := range V {
		V[s] = vVec.At(s, 0)
	}
	return V, nil
}

// improvePolicyGreedily updates pol[s] in place whenever an action a achieves
// Q(s, a, V) > Q(s, pol[s], V) + improveEps, and returns true if the policy is
// already stable across all states.
func (m *MDP) improvePolicyGreedily(pol []int, V []float64, improveEps float64) bool {
	stable := true
	for s, curAct := range pol {
		curQ := m.bellmanActionValue(s, curAct, V)
		bestAct, bestQ := m.bellmanOptimalBackup(s, V)
		if bestQ > curQ+improveEps && bestAct != curAct {
			pol[s] = bestAct
			stable = false
		}
	}
	return stable
}

// validateMDP checks that m is non-nil and structurally valid even if m was
// constructed as a raw struct literal rather than via NewMDP.
func validateMDP(m *MDP) error {
	if m == nil || len(m.Transitions) == 0 || m.Rewards == nil {
		return fmt.Errorf("%w: nil or incomplete MDP", markov.ErrNotStochastic)
	}
	if err := validateDiscountFactor(m.Discount); err != nil {
		return err
	}
	n, err := checkTransitionSliceDimensions(m.Transitions)
	if err != nil {
		return err
	}
	return checkRewardMatrixFinite(m.Rewards, n, len(m.Transitions))
}
