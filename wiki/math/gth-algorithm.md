---
type: concept
title: "GTH Algorithm"
description: "Grassmann, Taksar, and Heyman (1985) algorithm for numerically stable stationary distribution calculations in finite Markov chains."
resource: "https://doi.org/10.1287/opre.33.5.1107"
tags: ['math', 'algorithms', 'gth', 'stationary-distribution']
timestamp: "2026-11-14T06:43:02Z"
---

# GTH Algorithm

The GTH (Grassmann-Taksar-Heyman) algorithm (1985) is a numerically stable modification of Gaussian elimination for finding the stationary distribution of a Markov chain.
### Numerical Properties
- Avoids the subtraction of positive numbers, preventing catastrophic cancellation.
- Ensures all computed values are non-negative.
- Guaranteed stability even for chains with near-unit eigenvalues.
- In `github.com/stephen-mcelhose/markov`, the `solve.Stationary` function implements the GTH algorithm as the default solver.

## Key Points

- Canonical mathematical reference supporting `github.com/stephen-mcelhose/markov` capabilities.
- Highly relevant for state reduction, matrix solvers, or statistical estimation routines in Go.
- Synthesized and cross-referenced with related pages in the wiki database.

## Sources

- [GTH Algorithm](https://doi.org/10.1287/opre.33.5.1107)
