---
type: concept
title: "Traces of Consciousness: Schur Complement Censoring"
description: "Mathematical preprint detailing trace-chain construction via Schur complement censoring of intermediate excursions."
resource: "https://www.preprints.org/manuscript/202410.1305/v1"
tags: ['math', 'censoring', 'schur-complement', 'excursions', 'trace-chain']
timestamp: "2026-11-14T06:43:02Z"
---

# Traces of Consciousness: Schur Complement Censoring

The preprint "Traces of Consciousness" by Hoffman, Prakash, and Chattopadhyay (2024) provides the formal mathematical grounding for trace-chain construction.
### Key Insights
- Under censoring, any Markov chain $P$ projected onto a subset of states $A$ retains row-stochastic structure through the Schur complement.
- The censored transition matrix is given by $P_A = P_{AA} + P_{AB}(I - P_{BB})^{-1}P_{BA}$, where $B = A^c$ is the censored state set.
- This represents the exact multi-step excursion probability between states in $A$, bypassing intermediate visits to $B$.

## Key Points

- Canonical mathematical reference supporting `github.com/stephen-mcelhose/markov` capabilities.
- Highly relevant for state reduction, matrix solvers, or statistical estimation routines in Go.
- Synthesized and cross-referenced with related pages in the wiki database.

## Sources

- [Traces of Consciousness: Schur Complement Censoring](https://www.preprints.org/manuscript/202410.1305/v1)
