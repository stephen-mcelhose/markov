---
type: decision
title: "Lumpability Implementation and State-Space Reduction Debate"
description: "Architectural debate and design of the state partitioning, lumping, and dual-surface reduction leading to the creation of the markov repository."
resource: "https://github.com/stephen-mcelhose/catrace/issues/59"
tags: ['lumpability', 'partition', 'refinement', 'decision-log', 'catrace']
timestamp: "2026-11-14T06:43:02Z"
---

# Lumpability Implementation and State-Space Reduction Debate

This issue is the defining design event that birthed the `markov` repository. It contains the complete initial implementations of `lump.go` and `partition.go`, along with the `lump_test.go` test suite.
### The Core Debate: PROS & CONS
- **PRO**: Unifies state-space reduction by offering spatial lumping (Kemeny-Snell quotient projection: $\hat{P} = UPV$) as the dual counterpart to temporal reduction (Schur complement censoring / Trace-chains: $P_A = a + b(I-c)^{-1}d$).
- **CON**: Adding this directly to `catrace` risked bloating an already specialized trace library. The decision was made to split these foundational Markov chain operations into `github.com/stephen-mcelhose/markov`.

## Key Points

- Canonical mathematical reference supporting `github.com/stephen-mcelhose/markov` capabilities.
- Highly relevant for state reduction, matrix solvers, or statistical estimation routines in Go.
- Synthesized and cross-referenced with related pages in the wiki database.

## Sources

- [Lumpability Implementation and State-Space Reduction Debate](https://github.com/stephen-mcelhose/catrace/issues/59)
