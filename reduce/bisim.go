package reduce

import (
	"fmt"
	"slices"
	"sort"

	"github.com/stephen-mcelhose/markov"
)

// CoarsestPartition computes the coarsest partition that refines initial and
// under which k is strongly lumpable, using Paige-Tarjan partition refinement.
//
// The algorithm iteratively splits blocks of the current partition whenever two
// states in the same block have aggregate transition probabilities into some
// splitter block that differ by more than tol, until no further splits occur.
//
// When initial is nil, it defaults to the single-block partition {{0, …, n-1}}.
// Returns [markov.ErrInvalidPartition] if a non-nil initial is not a valid partition of k.
func CoarsestPartition(k *markov.Kernel, initial markov.Partition, tol float64) (markov.Partition, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}
	n := k.NumStates()
	if initial == nil {
		initial = singleUniversalBlock(n)
	}
	if err := initial.Validate(n); err != nil {
		return nil, err
	}
	current := clonePartition(initial)

	for {
		next, changed := refinePartitionPass(k, current, tol)
		current = next
		if !changed {
			return current, nil
		}
	}
}

// AreBisimilar reports whether k1 and k2 are bisimilar: i.e., they share a
// common strongly lumpable quotient up to state-space isomorphism σ:
//
//	∃ P̂: k1 → P̂ and k2 → P̂ are strongly lumpable quotients within tol
//
// In particular:
//   - AreBisimilar is reflexive, symmetric, and transitive on strongly lumpable
//     equivalence classes.
//   - Whenever IsStronglyLumpable(P, B, tol) holds, AreBisimilar(P, Lump(P, B), tol)
//     is true.
//   - Two chains with the same state count that have no common lumpable quotient
//     (and are not isomorphic) return false.
func AreBisimilar(k1, k2 *markov.Kernel, tol float64) (bool, error) {
	if k1 == nil || k1.P == nil {
		return false, fmt.Errorf("%w: nil k1", markov.ErrNotStochastic)
	}
	if k2 == nil || k2.P == nil {
		return false, fmt.Errorf("%w: nil k2", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return false, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}

	if k1.NumStates() == k2.NumStates() && areIsomorphic(k1, k2, tol) {
		return true, nil
	}
	if k1.NumStates() > k2.NumStates() && isQuotientOf(k1, k2, tol) {
		return true, nil
	}
	if k2.NumStates() > k1.NumStates() && isQuotientOf(k2, k1, tol) {
		return true, nil
	}
	return shareCommonLumpableQuotient(k1, k2, tol), nil
}

// ── internal helpers: partition refinement ────────────────────────────────────

// singleUniversalBlock returns the 1-block partition {{0, 1, …, n-1}}.
func singleUniversalBlock(n int) markov.Partition {
	all := make([]int, n)
	for i := range all {
		all[i] = i
	}
	return markov.Partition{all}
}

// clonePartition returns a deep copy of partition p.
func clonePartition(p markov.Partition) markov.Partition {
	out := make(markov.Partition, len(p))
	for i, blk := range p {
		out[i] = slices.Clone(blk)
	}
	return out
}

// refinePartitionPass performs one full sweep of Paige-Tarjan refinement across
// all blocks of current against all splitter blocks in current.
func refinePartitionPass(k *markov.Kernel, current markov.Partition, tol float64) (markov.Partition, bool) {
	changed := false
	var next markov.Partition
	for _, blk := range current {
		subBlocks, blkChanged := refineBlockAgainstSplitters(k, blk, current, tol)
		if blkChanged {
			changed = true
		}
		next = append(next, subBlocks...)
	}
	return next, changed
}

// refineBlockAgainstSplitters splits candidate block blk sequentially against
// every splitter block in splitters whenever states in a sub-block have
// aggregate transition probabilities into a splitter that differ by > tol.
func refineBlockAgainstSplitters(k *markov.Kernel, blk []int, splitters markov.Partition, tol float64) ([][]int, bool) {
	subBlocks := [][]int{blk}
	changed := false
	for _, splitter := range splitters {
		var refined [][]int
		for _, sb := range subBlocks {
			split := splitBlockBySplitter(k, sb, splitter, tol)
			if len(split) > 1 {
				changed = true
			}
			refined = append(refined, split...)
		}
		subBlocks = refined
	}
	return subBlocks, changed
}

// stateTransitionMass pairs a state index s with its aggregate transition
// probability Σ_{t∈splitter} P[s, t] into a splitter block.
type stateTransitionMass struct {
	state int
	mass  float64
}

// splitBlockBySplitter partitions states in sb into sub-blocks of diameter ≤ tol
// with respect to their aggregate transition probability into splitter.
func splitBlockBySplitter(k *markov.Kernel, sb, splitter []int, tol float64) [][]int {
	if len(sb) <= 1 {
		return [][]int{sb}
	}
	items := computeSplitterMasses(k, sb, splitter)
	if items[len(items)-1].mass-items[0].mass <= tol {
		return [][]int{sb}
	}
	return clusterSortedStatesByDiameter(items, tol)
}

// computeSplitterMasses computes Σ_{t∈splitter} P[s, t] for each state s ∈ sb
// and sorts the resulting pairs in ascending order of mass.
func computeSplitterMasses(k *markov.Kernel, sb, splitter []int) []stateTransitionMass {
	items := make([]stateTransitionMass, len(sb))
	for idx, s := range sb {
		items[idx] = stateTransitionMass{
			state: s,
			mass:  transitionMassIntoBlock(k.P.RawRowView(s), splitter),
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].mass < items[j].mass
	})
	return items
}

// transitionMassIntoBlock returns Σ_{t∈block} row[t].
func transitionMassIntoBlock(row []float64, block []int) float64 {
	var sum float64
	for _, t := range block {
		sum += row[t]
	}
	return sum
}

// clusterSortedStatesByDiameter groups mass-sorted states into contiguous
// clusters of diameter ≤ tol and sorts both the state indices within each
// cluster and the clusters themselves by minimum state index.
func clusterSortedStatesByDiameter(items []stateTransitionMass, tol float64) [][]int {
	var groups [][]int
	curGroup := []int{items[0].state}
	curMin := items[0].mass
	for _, it := range items[1:] {
		if it.mass-curMin <= tol {
			curGroup = append(curGroup, it.state)
		} else {
			sort.Ints(curGroup)
			groups = append(groups, curGroup)
			curGroup = []int{it.state}
			curMin = it.mass
		}
	}
	sort.Ints(curGroup)
	groups = append(groups, curGroup)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i][0] < groups[j][0]
	})
	return groups
}

// ── internal helpers: bisimulation & quotient isomorphism ─────────────────────

// shareCommonLumpableQuotient reports whether k1 and k2 both project onto a
// common strongly lumpable quotient Q of size m ≥ 2 up to isomorphism.
func shareCommonLumpableQuotient(k1, k2 *markov.Kernel, tol float64) bool {
	maxM := min(k1.NumStates(), k2.NumStates())
	for m := maxM; m >= 2; m-- {
		q1s := allStronglyLumpableQuotients(k1, m, tol)
		if len(q1s) == 0 {
			continue
		}
		q2s := allStronglyLumpableQuotients(k2, m, tol)
		if anyPairIsomorphic(q1s, q2s, tol) {
			return true
		}
	}
	return false
}

// anyPairIsomorphic reports whether any kernel in q1s is isomorphic to any
// kernel in q2s within tol.
func anyPairIsomorphic(q1s, q2s []*markov.Kernel, tol float64) bool {
	for _, q1 := range q1s {
		for _, q2 := range q2s {
			if areIsomorphic(q1, q2, tol) {
				return true
			}
		}
	}
	return false
}

// allStronglyLumpableQuotients enumerates all m-block partitions of k using
// restricted growth strings and returns the lumped kernels for those partitions
// that are strongly lumpable within tol.
func allStronglyLumpableQuotients(k *markov.Kernel, m int, tol float64) []*markov.Kernel {
	n := k.NumStates()
	if m > n || m <= 0 {
		return nil
	}
	if m == n {
		return []*markov.Kernel{k}
	}
	var out []*markov.Kernel
	enumerateSetPartitions(n, m, func(p markov.Partition) {
		if q := lumpIfStronglyLumpable(k, p, tol); q != nil {
			out = append(out, q)
		}
	})
	return out
}

// lumpIfStronglyLumpable returns Lump(k, p) if k is strongly lumpable on p
// within tol, or nil otherwise.
func lumpIfStronglyLumpable(k *markov.Kernel, p markov.Partition, tol float64) *markov.Kernel {
	ok, err := IsStronglyLumpable(k, p, tol)
	if err != nil || !ok {
		return nil
	}
	q, err := Lump(k, p, nil, nil, tol)
	if err != nil {
		return nil
	}
	return q
}

// enumerateSetPartitions generates all set partitions of {0, …, n-1} into
// exactly m non-empty blocks and invokes visit for each partition.
func enumerateSetPartitions(n, m int, visit func(markov.Partition)) {
	var blocks [][]int
	var step func(s int)
	step = func(s int) {
		remaining := n - s
		emptyNeeded := m - len(blocks)
		if remaining < emptyNeeded {
			return
		}
		if s == n {
			visit(clonePartition(markov.Partition(blocks)))
			return
		}
		if remaining > emptyNeeded {
			for i := range blocks {
				blocks[i] = append(blocks[i], s)
				step(s + 1)
				blocks[i] = blocks[i][:len(blocks[i])-1]
			}
		}
		if len(blocks) < m {
			blocks = append(blocks, []int{s})
			step(s + 1)
			blocks = blocks[:len(blocks)-1]
		}
	}
	step(0)
}

// areIsomorphic reports whether there exists a state permutation σ ∈ S_n such
// that |k1.P[i, j] - k2.P[σ(i), σ(j)]| ≤ tol for all i, j.
func areIsomorphic(k1, k2 *markov.Kernel, tol float64) bool {
	n := k1.NumStates()
	used := make([]bool, n)
	sigma := make([]int, n)

	var search func(i int) bool
	search = func(i int) bool {
		if i == n {
			return true
		}
		for cand := 0; cand < n; cand++ {
			if used[cand] || !isConsistentBijectionStep(k1, k2, sigma, i, cand, tol) {
				continue
			}
			used[cand] = true
			sigma[i] = cand
			if search(i + 1) {
				return true
			}
			used[cand] = false
		}
		return false
	}
	return search(0)
}

// isConsistentBijectionStep checks whether mapping state i of k1 to candidate
// state cand of k2 (σ(i) = cand) preserves self-loop and pairwise transitions
// with all previously mapped states 0..i-1 within tol.
func isConsistentBijectionStep(k1, k2 *markov.Kernel, sigma []int, i, cand int, tol float64) bool {
	if !markov.NearlyEqual(k1.P.At(i, i), k2.P.At(cand, cand), tol) {
		return false
	}
	for prev := 0; prev < i; prev++ {
		mappedPrev := sigma[prev]
		if !markov.NearlyEqual(k1.P.At(i, prev), k2.P.At(cand, mappedPrev), tol) ||
			!markov.NearlyEqual(k1.P.At(prev, i), k2.P.At(mappedPrev, cand), tol) {
			return false
		}
	}
	return true
}

// isQuotientOf reports whether there exists a surjective block assignment
// φ: {0..n-1} ↠ {0..m-1} from large onto small satisfying the Kemeny-Snell
// commutation identity |Σ_{t∈φ⁻¹(j)} P_large[s, t] - P_small[φ(s), j]| ≤ tol.
func isQuotientOf(large, small *markov.Kernel, tol float64) bool {
	n := large.NumStates()
	m := small.NumStates()
	assign := make([]int, n)
	counts := make([]int, m)

	var search func(s int, emptyBlocks int) bool
	search = func(s int, emptyBlocks int) bool {
		if n-s < emptyBlocks {
			return false
		}
		if s == n {
			return satisfiesQuotientCommutation(large, small, assign, tol)
		}
		for b := 0; b < m; b++ {
			assign[s] = b
			nextEmpty := emptyBlocks
			if counts[b] == 0 {
				nextEmpty--
			}
			counts[b]++
			if search(s+1, nextEmpty) {
				return true
			}
			counts[b]--
		}
		return false
	}
	return search(0, m)
}

// satisfiesQuotientCommutation checks whether the surjective state assignment
// φ = assign satisfies |(P_large · V_φ)[i, j] - (V_φ · P_small)[i, j]| ≤ tol
// for every micro-state i and macro-block j.
func satisfiesQuotientCommutation(large, small *markov.Kernel, assign []int, tol float64) bool {
	n := large.NumStates()
	m := small.NumStates()
	for i := 0; i < n; i++ {
		row := large.P.RawRowView(i)
		bi := assign[i]
		for j := 0; j < m; j++ {
			if !markov.NearlyEqual(assignedBlockMass(row, assign, j), small.P.At(bi, j), tol) {
				return false
			}
		}
	}
	return true
}

// assignedBlockMass returns Σ_{t : assign[t] == blockIdx} row[t].
func assignedBlockMass(row []float64, assign []int, blockIdx int) float64 {
	var sum float64
	for t, b := range assign {
		if b == blockIdx {
			sum += row[t]
		}
	}
	return sum
}
