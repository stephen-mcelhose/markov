package solve

import (
	"fmt"
	"sort"

	"github.com/stephen-mcelhose/markov"
)

// StateClasses holds the structural decomposition of a Markov chain's state space.
// For acyclic transient states with no positive-probability return cycles,
// Periods[s] is 0.
type StateClasses struct {
	SCCs      [][]int
	Recurrent [][]int
	Transient []int
	Periods   map[int]int
}

// ClassifyStates decomposes k into strongly connected components (SCCs),
// closed recurrent communicating classes, transient states, and state periods.
//
// An edge i → j is present when k.P[i, j] > tol.
// Returns [markov.ErrNotStochastic] if k is nil or tol < 0.
func ClassifyStates(k *markov.Kernel, tol float64) (*StateClasses, error) {
	if k == nil || k.P == nil {
		return nil, fmt.Errorf("%w: nil kernel", markov.ErrNotStochastic)
	}
	if tol < 0 {
		return nil, fmt.Errorf("%w: negative tolerance %g", markov.ErrNotStochastic, tol)
	}

	adj := transitionAdjacencyGraph(k, tol)
	sccs := tarjanSCCs(adj)
	sccOf := componentIndexByState(sccs, k.NumStates())

	var recurrent [][]int
	var transient []int
	periods := make(map[int]int, k.NumStates())

	for idx, comp := range sccs {
		period := computeComponentPeriod(comp, adj)
		for _, s := range comp {
			periods[s] = period
		}
		if isClosedRecurrentClass(comp, idx, adj, sccOf) {
			recurrent = append(recurrent, append([]int(nil), comp...))
		} else {
			transient = append(transient, comp...)
		}
	}
	sort.Ints(transient)

	return &StateClasses{
		SCCs:      sccs,
		Recurrent: recurrent,
		Transient: transient,
		Periods:   periods,
	}, nil
}

// AsPartition returns the SCC decomposition as a [markov.Partition].
func (sc *StateClasses) AsPartition() markov.Partition {
	if sc == nil || len(sc.SCCs) == 0 {
		return nil
	}
	out := make(markov.Partition, len(sc.SCCs))
	for i, blk := range sc.SCCs {
		out[i] = append([]int(nil), blk...)
	}
	return out
}

// ── internal helpers ──────────────────────────────────────────────────────────

// transitionAdjacencyGraph builds the directed support graph G = (S, E) of
// kernel k, where directed edge (i → j) ∈ E iff P[i, j] > tol.
func transitionAdjacencyGraph(k *markov.Kernel, tol float64) [][]int {
	n := k.NumStates()
	adj := make([][]int, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if k.P.At(i, j) > tol {
				adj[i] = append(adj[i], j)
			}
		}
	}
	return adj
}

// tarjanState holds the depth-first search book-keeping arrays for Tarjan's
// linear-time strongly connected components (SCC) algorithm.
type tarjanState struct {
	adj     [][]int
	index   int
	indices []int
	lowlink []int
	onStack []bool
	stack   []int
	sccs    [][]int
}

// tarjanSCCs partitions the vertices of adj into maximal communicating classes
// (strongly connected components) using Tarjan's (1972) DFS algorithm, sorted
// deterministically by each component's minimum state index.
func tarjanSCCs(adj [][]int) [][]int {
	n := len(adj)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = -1
	}
	ts := &tarjanState{
		adj:     adj,
		indices: indices,
		lowlink: make([]int, n),
		onStack: make([]bool, n),
	}
	for v := 0; v < n; v++ {
		if ts.indices[v] == -1 {
			ts.strongConnect(v)
		}
	}
	sort.Slice(ts.sccs, func(i, j int) bool {
		return ts.sccs[i][0] < ts.sccs[j][0]
	})
	return ts.sccs
}

// strongConnect visits vertex v in Tarjan's DFS, updating lowlink values and
// popping a strongly connected component whenever v is a component root.
func (ts *tarjanState) strongConnect(v int) {
	ts.indices[v] = ts.index
	ts.lowlink[v] = ts.index
	ts.index++
	ts.stack = append(ts.stack, v)
	ts.onStack[v] = true

	for _, w := range ts.adj[v] {
		ts.visitNeighbor(v, w)
	}
	if ts.lowlink[v] == ts.indices[v] {
		ts.popComponent(v)
	}
}

// visitNeighbor updates lowlink[v] along directed edge (v → w) during Tarjan's DFS.
func (ts *tarjanState) visitNeighbor(v, w int) {
	if ts.indices[w] == -1 {
		ts.strongConnect(w)
		if ts.lowlink[w] < ts.lowlink[v] {
			ts.lowlink[v] = ts.lowlink[w]
		}
	} else if ts.onStack[w] && ts.indices[w] < ts.lowlink[v] {
		ts.lowlink[v] = ts.indices[w]
	}
}

// popComponent pops vertices from the Tarjan DFS stack down to root v to form
// a single sorted strongly connected component.
func (ts *tarjanState) popComponent(v int) {
	var scc []int
	for {
		w := ts.stack[len(ts.stack)-1]
		ts.stack = ts.stack[:len(ts.stack)-1]
		ts.onStack[w] = false
		scc = append(scc, w)
		if w == v {
			break
		}
	}
	sort.Ints(scc)
	ts.sccs = append(ts.sccs, scc)
}

// componentIndexByState maps each state s ∈ {0, …, n-1} to the index of its
// strongly connected component in sccs.
func componentIndexByState(sccs [][]int, n int) []int {
	sccOf := make([]int, n)
	for idx, comp := range sccs {
		for _, s := range comp {
			sccOf[s] = idx
		}
	}
	return sccOf
}

// isClosedRecurrentClass reports whether strongly connected component comp
// (with index compIdx) is a closed recurrent communicating class: i.e., it has
// at least one internal transition and no outgoing transition to any state
// outside comp.
func isClosedRecurrentClass(comp []int, compIdx int, adj [][]int, sccOf []int) bool {
	hasInternalEdge := false
	for _, u := range comp {
		for _, v := range adj[u] {
			if sccOf[v] != compIdx {
				return false
			}
			hasInternalEdge = true
		}
	}
	return hasInternalEdge
}

// computeComponentPeriod computes the period d = gcd{t ≥ 1 : (Pᵗ)[s,s] > 0}
// of a strongly connected component using Balcer & Veinott's (1973) BFS
// spanning-tree depth-difference algorithm: for every edge u → v inside comp,
// d divides |depth[u] - depth[v] + 1|.
func computeComponentPeriod(comp []int, adj [][]int) int {
	inComp := make(map[int]bool, len(comp))
	for _, s := range comp {
		inComp[s] = true
	}
	depth := map[int]int{comp[0]: 0}
	queue := []int{comp[0]}
	g := 0

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		g = visitComponentEdges(u, adj[u], inComp, depth, &queue, g)
	}
	return g
}

// visitComponentEdges inspects outgoing edges from state u within the component,
// enqueuing unvisited tree edges and accumulating the GCD of cycle chord lengths.
func visitComponentEdges(u int, neighbors []int, inComp map[int]bool, depth map[int]int, queue *[]int, g int) int {
	for _, v := range neighbors {
		if !inComp[v] {
			continue
		}
		if dv, seen := depth[v]; !seen {
			depth[v] = depth[u] + 1
			*queue = append(*queue, v)
		} else {
			g = gcd(g, depth[u]-dv+1)
		}
	}
	return g
}

// gcd returns the non-negative greatest common divisor of a and b.
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}
