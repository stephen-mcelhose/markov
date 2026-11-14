package control

import (
	"fmt"
	"math"
	"slices"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// MDP represents a discounted finite-state, finite-action Markov Decision Process
// (S, A, {Pᵃ}_{a∈A}, R, γ).
type MDP struct {
	Actions     []string
	Transitions []*markov.Kernel
	Rewards     *mat.Dense
	Discount    float64
}

// NewMDP constructs and validates a discounted MDP:
//
//	|A| ≥ 1,   |Transitions| = |A|,   dim(R) = (n, |A|),   γ ∈ [0, 1)
//
// Returns an error if actions or transitions are empty, dimensions mismatch,
// any transition kernel is invalid within tol, rewards is nil or non-finite,
// or discount ∉ [0, 1).
func NewMDP(actions []string, transitions []*markov.Kernel, rewards *mat.Dense, discount, tol float64) (*MDP, error) {
	if err := validateNonNegativeTol(tol); err != nil {
		return nil, err
	}
	if len(actions) > 0 && len(actions) != len(transitions) {
		return nil, fmt.Errorf("%w: %d actions for %d transition kernels", markov.ErrNotStochastic, len(actions), len(transitions))
	}
	if err := validateDiscountFactor(discount); err != nil {
		return nil, err
	}
	clonedTransitions, n, err := validateAndCloneTransitions(transitions, tol)
	if err != nil {
		return nil, err
	}
	clonedRewards, err := validateAndCloneRewards(rewards, n, len(transitions))
	if err != nil {
		return nil, err
	}

	return &MDP{
		Actions:     resolveActionNames(actions, len(transitions)),
		Transitions: clonedTransitions,
		Rewards:     clonedRewards,
		Discount:    discount,
	}, nil
}

// NumStates returns the number of states n = |S|.
func (m *MDP) NumStates() int {
	if m == nil || len(m.Transitions) == 0 || m.Transitions[0] == nil {
		return 0
	}
	return m.Transitions[0].NumStates()
}

// NumActions returns the number of actions k = |A|.
func (m *MDP) NumActions() int {
	if m == nil {
		return 0
	}
	return len(m.Transitions)
}

// InducedKernel constructs the n×n Markov transition kernel P^π induced by the
// deterministic policy π: S → A:
//
//	P^π[s, s'] = P^{π(s)}[s, s']
//
// Returns an error if m is nil, len(policy) ≠ NumStates(), or any action index
// is out of range [0, NumActions()).
func (m *MDP) InducedKernel(policy []int, tol float64) (*markov.Kernel, error) {
	if err := validateMDP(m); err != nil {
		return nil, err
	}
	if err := validateNonNegativeTol(tol); err != nil {
		return nil, err
	}
	n := m.NumStates()
	if len(policy) != n {
		return nil, fmt.Errorf("%w: policy length %d does not match state count %d", markov.ErrInvalidPartition, len(policy), n)
	}
	P, err := assemblePolicyTransitionMatrix(m, policy)
	if err != nil {
		return nil, err
	}
	return markov.NewKernelWithTol(P, m.Transitions[0].StateNames, math.Max(tol, markov.DefaultTolerance))
}

// ── internal helpers ──────────────────────────────────────────────────────────

// validateNonNegativeTol checks that tol is finite and ≥ 0.
func validateNonNegativeTol(tol float64) error {
	if tol < 0 || math.IsNaN(tol) || math.IsInf(tol, 0) {
		return fmt.Errorf("%w: invalid tolerance %g", markov.ErrNotStochastic, tol)
	}
	return nil
}

// validateDiscountFactor checks that the MDP discount factor γ lies in [0, 1).
func validateDiscountFactor(discount float64) error {
	if discount < 0 || discount >= 1.0 || math.IsNaN(discount) {
		return fmt.Errorf("%w: discount %g must be in [0, 1)", markov.ErrInvalidDistribution, discount)
	}
	return nil
}

// validateAndCloneTransitions verifies that transitions is non-empty and every
// action kernel Pᵃ is non-nil, has equal state dimension n ≥ 1, and is valid
// within tol, returning a deep-copied slice of kernels and state count n.
func validateAndCloneTransitions(transitions []*markov.Kernel, tol float64) ([]*markov.Kernel, int, error) {
	n, err := checkTransitionSliceDimensions(transitions)
	if err != nil {
		return nil, 0, err
	}
	cloned := make([]*markov.Kernel, len(transitions))
	for a, k := range transitions {
		if err := k.Validate(tol); err != nil {
			return nil, 0, fmt.Errorf("action %d: %w", a, err)
		}
		cloned[a] = k.Clone()
	}
	return cloned, n, nil
}

// checkTransitionSliceDimensions verifies that transitions is non-empty and
// every action kernel is non-nil with identical state count n ≥ 1.
func checkTransitionSliceDimensions(transitions []*markov.Kernel) (int, error) {
	if len(transitions) == 0 {
		return 0, fmt.Errorf("%w: transitions must be non-empty", markov.ErrNotStochastic)
	}
	if transitions[0] == nil || transitions[0].P == nil {
		return 0, fmt.Errorf("%w: nil transition kernel at action 0", markov.ErrNotStochastic)
	}
	n := transitions[0].NumStates()
	if n <= 0 {
		return 0, fmt.Errorf("%w: invalid state count %d", markov.ErrNotStochastic, n)
	}
	for a, k := range transitions {
		if k == nil || k.P == nil {
			return 0, fmt.Errorf("%w: nil transition kernel at action %d", markov.ErrNotStochastic, a)
		}
		if k.NumStates() != n {
			return 0, fmt.Errorf("%w: action %d kernel has %d states, want %d", markov.ErrNotStochastic, a, k.NumStates(), n)
		}
	}
	return n, nil
}

// validateAndCloneRewards checks that rewards has dimensions n×numActions with
// finite entries and returns a deep copy.
func validateAndCloneRewards(rewards *mat.Dense, n, numActions int) (*mat.Dense, error) {
	if err := checkRewardMatrixFinite(rewards, n, numActions); err != nil {
		return nil, err
	}
	cloned := mat.NewDense(n, numActions, nil)
	cloned.Copy(rewards)
	return cloned, nil
}

// checkRewardMatrixFinite verifies that rewards is non-nil, has dimensions
// n×numActions, and contains no NaN or ±Inf entries.
func checkRewardMatrixFinite(rewards *mat.Dense, n, numActions int) error {
	if rewards == nil {
		return fmt.Errorf("%w: nil rewards matrix", markov.ErrInvalidDistribution)
	}
	rRows, rCols := rewards.Dims()
	if rRows != n || rCols != numActions {
		return fmt.Errorf("%w: rewards matrix has dims %dx%d, want %dx%d", markov.ErrInvalidDistribution, rRows, rCols, n, numActions)
	}
	for s := 0; s < rRows; s++ {
		if err := checkRowFinite(rewards.RawRowView(s), s); err != nil {
			return err
		}
	}
	return nil
}

// checkRowFinite verifies that every entry in row s is finite (not NaN or ±Inf).
func checkRowFinite(row []float64, rowIdx int) error {
	for a, v := range row {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w: non-finite reward at (%d,%d): %g", markov.ErrInvalidDistribution, rowIdx, a, v)
		}
	}
	return nil
}

// resolveActionNames clones actions when provided, or synthesizes ["a0", "a1", ...].
func resolveActionNames(actions []string, numActions int) []string {
	if len(actions) == numActions {
		return slices.Clone(actions)
	}
	out := make([]string, numActions)
	for a := range out {
		out[a] = fmt.Sprintf("a%d", a)
	}
	return out
}

// assemblePolicyTransitionMatrix builds P^π where row s is copied from P^{π(s)}[s, ·].
func assemblePolicyTransitionMatrix(m *MDP, policy []int) (*mat.Dense, error) {
	n := m.NumStates()
	numActions := m.NumActions()
	P := mat.NewDense(n, n, nil)
	for s, a := range policy {
		if a < 0 || a >= numActions {
			return nil, fmt.Errorf("%w: policy[%d] = %d out of range [0,%d)", markov.ErrInvalidPartition, s, a, numActions)
		}
		copy(P.RawRowView(s), m.Transitions[a].P.RawRowView(s))
	}
	return P, nil
}
