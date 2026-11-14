package control

import (
	"fmt"
	"sort"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// CoarsestPartition computes the Givan-Dean-Greig coarsest stochastic bisimulation
// partition of the MDP's state space within tolerance tol:
//
//	s ~ s'  ⟺  ∀a ∈ A: |R(s, a) - R(s', a)| ≤ tol  ∧
//	             ∀B ∈ 𝒫*: |Σ_{t∈B} Pᵃ[s, t] - Σ_{t∈B} Pᵃ[s', t]| ≤ tol
//
// First partitions states by immediate reward equivalence across all actions,
// then iteratively refines blocks across all action transition kernels Pᵃ until
// every action kernel is strongly lumpable on 𝒫* within tol.
func (m *MDP) CoarsestPartition(tol float64) (markov.Partition, error) {
	if err := validateMDP(m); err != nil {
		return nil, err
	}
	if err := validateNonNegativeTol(tol); err != nil {
		return nil, err
	}

	current := m.seedRewardEquivalencePartition(tol)
	for {
		next, changed := m.refineMDPPartitionPass(current, tol)
		current = next
		if !changed {
			return current, nil
		}
	}
}

// MDPBisimulation is the top-level form of [MDP.CoarsestPartition].
func MDPBisimulation(m *MDP, tol float64) (markov.Partition, error) {
	if m == nil {
		return nil, fmt.Errorf("%w: nil MDP", markov.ErrNotStochastic)
	}
	return m.CoarsestPartition(tol)
}

// ── internal helpers ──────────────────────────────────────────────────────────

// seedRewardEquivalencePartition builds the initial reward-equivalence partition
// 𝒫₀ where two states s, s' share a block iff |R(s, a) - R(s', a)| ≤ tol for
// every action a ∈ A.
func (m *MDP) seedRewardEquivalencePartition(tol float64) markov.Partition {
	n := m.NumStates()
	all := make([]int, n)
	for s := range all {
		all[s] = s
	}
	current := markov.Partition{all}
	for a := 0; a < m.NumActions(); a++ {
		current = refineBlocksByObservable(current, func(s int) float64 {
			return m.Rewards.At(s, a)
		}, tol)
	}
	return current
}

// refineMDPPartitionPass performs one Givan-Dean-Greig refinement pass across
// all blocks of current against all actions a ∈ A and splitter blocks Bⱼ ∈ current.
func (m *MDP) refineMDPPartitionPass(current markov.Partition, tol float64) (markov.Partition, bool) {
	changed := false
	var next markov.Partition
	for _, blk := range current {
		subBlocks, blkChanged := m.refineBlockAcrossActions(blk, current, tol)
		if blkChanged {
			changed = true
		}
		next = append(next, subBlocks...)
	}
	return next, changed
}

// refineBlockAcrossActions splits candidate block blk across every action kernel
// Pᵃ and every splitter block in splitters whenever states in a sub-block have
// aggregate transition probabilities into a splitter that differ by > tol.
func (m *MDP) refineBlockAcrossActions(blk []int, splitters markov.Partition, tol float64) (markov.Partition, bool) {
	subBlocks := markov.Partition{blk}
	changed := false
	for _, k := range m.Transitions {
		for _, splitter := range splitters {
			refined := refineBlocksByObservable(subBlocks, func(s int) float64 {
				return blockTransitionProbability(k.P, s, splitter)
			}, tol)
			if len(refined) > len(subBlocks) {
				changed = true
			}
			subBlocks = refined
		}
	}
	return subBlocks, changed
}

// blockTransitionProbability returns Σ_{t∈splitter} P[s, t].
func blockTransitionProbability(P *mat.Dense, s int, splitter []int) float64 {
	row := P.RawRowView(s)
	var sum float64
	for _, t := range splitter {
		sum += row[t]
	}
	return sum
}

// refineBlocksByObservable refines every block in blocks by partitioning states
// whose observable values observableFn(s) differ by more than tol.
func refineBlocksByObservable(blocks markov.Partition, observableFn func(s int) float64, tol float64) markov.Partition {
	var next markov.Partition
	for _, blk := range blocks {
		next = append(next, splitBlockByObservable(blk, observableFn, tol)...)
	}
	return next
}

// stateObservableValue pairs a state index s with a scalar observable (such as
// immediate reward R(s, a) or block transition probability Σ_{t∈B} Pᵃ[s, t]).
type stateObservableValue struct {
	state int
	val   float64
}

// splitBlockByObservable partitions states into sub-blocks of diameter ≤ tol
// with respect to observableFn(s).
func splitBlockByObservable(states []int, observableFn func(s int) float64, tol float64) [][]int {
	if len(states) <= 1 {
		return [][]int{states}
	}
	items := make([]stateObservableValue, len(states))
	for i, s := range states {
		items[i] = stateObservableValue{state: s, val: observableFn(s)}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].val < items[j].val
	})
	if items[len(items)-1].val-items[0].val <= tol {
		return [][]int{states}
	}

	var groups [][]int
	curGroup := []int{items[0].state}
	curMin := items[0].val
	for _, it := range items[1:] {
		if it.val-curMin <= tol {
			curGroup = append(curGroup, it.state)
		} else {
			sort.Ints(curGroup)
			groups = append(groups, curGroup)
			curGroup = []int{it.state}
			curMin = it.val
		}
	}
	sort.Ints(curGroup)
	groups = append(groups, curGroup)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i][0] < groups[j][0]
	})
	return groups
}
