---
type: decision
title: Origin and Design Lineage
description: Why markov exists as a separate repo from catrace, and the design decisions captured in catrace issue #59 that define its surface area.
resource: https://github.com/stephen-mcelhose/catrace/issues/59
tags: [design, lumpability, dual-surface-api, catrace, origin]
timestamp: 2026-11-14T06:44:10Z
---

# Origin and Design Lineage

`markov` was born from [catrace issue #59](https://github.com/stephen-mcelhose/catrace/issues/59), a feature request to add Kemeny-Snell spatial quotienting (`Lump`) to catrace.

## The Decision

The issue recorded the tension explicitly:

> **PROS:** A very interesting feature.
> **CONS:** This starts to push us towards being a stochastic model reduction library, which might confuse original purpose.
> **Alternative:** create `stephen-mcelhose/markov` and purpose-build this.

catrace models autonomous-agent perception-decision-action loops. Adding general state-space reduction would dilute that focus. `markov` is the purpose-built home for the mathematics catrace was using but not exposing.

## What Issue #59 Contains

The issue's comments contain complete, tested reference implementations:

- **Comment 2:** Kemeny-Snell projection primer + full `lump.go` (methods on `Kernel`) + exhaustive `lump_test.go` including two algebraic invariant tests:
  - Multi-step commutation: $P^t V = V \hat{P}^t$ for all $t \ge 1$
  - Commutativity: `Lump ∘ Trace = Trace ∘ Lump` when the traced subset respects partition blocks
- **Comment 3:** "Alternative as Objects" — promotes `Partition` to a first-class type with methods (`IndicatorMatrix`, `DistributorMatrix`, `ProjectDistribution`, `IsStronglyLumpable`, `Lump`)
- **Comment 4:** Dual-surface API — both `k.Lump(...)` method and `Lump(k, ...)` top-level function; this became the core AGENTS.md API principle

## Design Principles Established Here

1. **Dual surface:** every operator exposed as both a method (idiomatic Go chaining) and a top-level function (algebraic readability). Encoded in AGENTS.md §API.
2. **`Partition` as a first-class type** with `Validate`, `IndicatorMatrix`, `DistributorMatrix`, `ProjectDistribution` methods.
3. **`Must*` helpers** for fluent composition when inputs are known valid.
4. **Tolerance threading:** every numerically sensitive operation accepts an explicit `tol float64`.
5. **Errors, not panics:** constructors and singular solvers return `(T, error)`.

## catrace as Reference Implementation

catrace covers: `Kernel`, `Trace` (Schur complement), `Stationary`, `MFPT`, `ClassifyStates`, `stream/Estimator`.

catrace does **not** cover: `Lump`, `CoarsestPartition` (bisimulation), `MarkovOrderTest`, anything in `control/`. These are net-new in `markov`.

## Sources

- [catrace issue #59](https://github.com/stephen-mcelhose/catrace/issues/59)
- [catrace repo](https://github.com/stephen-mcelhose/catrace)
