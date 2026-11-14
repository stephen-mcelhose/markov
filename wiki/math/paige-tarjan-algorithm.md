---
type: concept
title: "Paige-Tarjan Partition Refinement Algorithm"
description: "The Paige & Tarjan (1987) algorithm for computing the coarsest partition of state-transition systems in O(m log n) time."
resource: "https://doi.org/10.1137/0216062"
tags: ['math', 'algorithms', 'paige-tarjan', 'partition-refinement']
timestamp: "2026-11-14T06:43:02Z"
---

# Paige-Tarjan Partition Refinement Algorithm

The seminal paper by Paige and Tarjan, "Three Partition Refinement Algorithms" (1987), forms the algorithmic backbone for relational bisimulation and partition refinement.
### Core Mechanism
- Solves the coarsest stable partition problem.
- Splits blocks against a "splitter" set using a clever "size-of-smaller-half" heuristic.
- Achieves highly efficient $O(m \log n)$ time complexity where $n$ is state count and $m$ is transition count.
- In `github.com/stephen-mcelhose/markov`, this algorithm is implemented in `reduce.CoarsestPartition` to compute the optimal state reduction of a transition relation or kernel.

## Key Points

- Canonical mathematical reference supporting `github.com/stephen-mcelhose/markov` capabilities.
- Highly relevant for state reduction, matrix solvers, or statistical estimation routines in Go.
- Synthesized and cross-referenced with related pages in the wiki database.

## Sources

- [Paige-Tarjan Partition Refinement Algorithm](https://doi.org/10.1137/0216062)
