---
type: concept
title: "Catrace Reference Implementation"
description: "The reference implementation in Go that provides trace-chain calculations and serves as the baseline for the markov library."
resource: "https://github.com/stephen-mcelhose/catrace"
tags: ['go', 'catrace', 'reference-implementation', 'trace-chain']
timestamp: "2026-11-14T06:43:02Z"
---

# Catrace Reference Implementation

`catrace` is the original reference implementation developed by Stephen McElhose. The `markov` library is a direct extraction, modernization, and generalization of `catrace`'s capabilities. 
Key architectural properties inherited from `catrace`:
- Matrix operations backed by `gonum/mat`.
- State space reduction utilizing Schur complements (Trace-chains).
- Row-stochastic validation and threshold normalization.

## Key Points

- Canonical mathematical reference supporting `github.com/stephen-mcelhose/markov` capabilities.
- Highly relevant for state reduction, matrix solvers, or statistical estimation routines in Go.
- Synthesized and cross-referenced with related pages in the wiki database.

## Sources

- [Catrace Reference Implementation](https://github.com/stephen-mcelhose/catrace)
