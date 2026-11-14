---
type: persona
title: Dreyfus Stage 2 (Advanced Beginner) Go Engineer
description: Defines the Advanced Beginner (Dreyfus Stage 2) target persona for markov example_test.go files — combining concrete Go systems stories, situational maxims, and verified // Output: assertions.
stage: Dreyfus Stage 2 (Advanced Beginner)
resource: https://en.wikipedia.org/wiki/Dreyfus_model_of_skill_acquisition
tags: ['persona', 'dreyfus-model', 'examples', 'example-test', 'situational-maxims']
timestamp: 2026-11-14T19:08:00Z
---

# Dreyfus Stage 2 (Advanced Beginner) Go Engineer

In the [Dreyfus model of skill acquisition](https://en.wikipedia.org/wiki/Dreyfus_model_of_skill_acquisition), **Novices (Stage 1)** rely on rigid, context-free syntax rules ("`NewKernel` takes a `*mat.Dense`"), whereas **Advanced Beginners (Stage 2)** learn through **situational maxims** and **narrative stories** that connect a recognizable operational problem to the right mathematical operator and its concrete engineering payoff.

All runnable examples in `markov` (`example_test.go` files across all packages) MUST target this persona.

---

## 1. Skill Stage

* **Stage**: Dreyfus Stage 2 — Advanced Beginner (competent in Go systems engineering; building intuition for stochastic linear algebra).
* **How This Stage Learns**: Through concrete operational situations paired with actionable rules of thumb (*"When in systems situation X, apply mathematical operator Y to preserve invariant Z"*), backed by a short 3-act narrative (*Setup the system* $\to$ *Apply the operator* $\to$ *Verify the operational payoff in `// Output:`*).

---

## 2. Prior Knowledge

* **Already Mastered (Do Not Over-Explain)**:
  - Idiomatic Go (`error` handling, slices, methods, `testing` package, `gonum.org/v1/gonum/mat`).
  - Distributed systems & backend state machines: RPC retry/backoff loops, multi-AZ worker pools, circuit breakers, streaming event logs, and queue backpressure controllers.
* **Intuition Being Built (Bridge Explicitly)**:
  - Why the Schur complement $P_A = a + b(I-c)^{-1}d$ folds away transient excursions without losing hitting probabilities or latency (`reduce.Trace`, `reduce.TraceReward`).
  - Why Kemeny–Snell projection $\hat{P} = UPV$ preserves multi-step dynamics when the block residual vanishes (`reduce.Lump`, `reduce.LumpabilityResidual`, `reduce.CoarsestPartition`).
  - Why GTH elimination avoids subtractive cancellation on nearly-disconnected chains (`solve.Stationary`) and how `solve.MFPTDelta` ranks the ROI of transition fixes.
  - Why streaming counts need EMA decay (`infer.NewEstimator`) and how the $G^2$ log-likelihood ratio (`infer.MarkovOrderTest`) detects hidden second-order memory.
  - How Bellman optimality (`control.PolicyIteration`, `control.ValueIteration`) turns transition matrices and SLA costs into an optimal control policy.

---

## 3. Mental Model & Vocabulary Bridge

| Go Systems / Production Symptom | Mathematical Object | `markov` Operator |
| :--- | :--- | :--- |
| Noisy internal retry/backoff states cluttering end-to-end SLA metrics | Schur complement censoring $P_A = a + b(I-c)^{-1}d$ & Poisson reward folding | `reduce.Trace`, `reduce.TraceReward` |
| Duplicated worker pods across availability zones (`us-east-1a`, `us-east-1b`) | Kemeny–Snell quotient $\hat{P} = UPV$ & strong lumpability residual $R(P,\mathcal{B})$ | `reduce.CoarsestPartition`, `reduce.Lump`, `reduce.LumpabilityResidual` |
| Cyclic service degradation (`Healthy -> Degraded -> CircuitOpen -> Rebooting`) | Absorbing fundamental matrix $N = (I-Q)^{-1}$ & convex row perturbation $\tilde{P}$ | `solve.MFPTToTarget`, `solve.MFPTDelta`, `solve.Stationary` |
| Streaming event logs with regime drift or unmodeled state history | Dirichlet-EMA count matrix $C^{(t)} = \gamma C^{(t-1)} + \mathbf{1}$ & $G^2 \sim \chi^2_{n(n-1)^2}$ | `infer.NewEstimator`, `infer.FromSequence`, `infer.MarkovOrderTest` |
| Queue backpressure controller trading throughput against tail-latency cost | Discounted MDP $(S, A, \{P^a\}, R, \gamma)$ & Bellman fixed point $V^*$ | `control.NewMDP`, `control.PolicyIteration`, `control.ValueIteration`, `control.MDPBisimulation` |

---

## 4. Situational Maxims

1. **Conditioning & Rollups (`markov`)**: *"When raw telemetry gives unnormalized transition counts or per-pod probabilities, normalize into a `markov.Kernel` and use `Distribution.Restrict` (subset conditioning) and `Distribution.Project` (zone rollups) instead of manual slice loops."*
2. **Temporal Censoring (`reduce.Trace`)**: *"When a state machine has noisy internal retry or backoff states that you do not want to monitor directly, use `reduce.Trace` and `reduce.TraceReward` to fold them away into exact end-to-end completion probabilities and expected latencies."*
3. **Spatial Quotienting (`reduce.Lump`)**: *"When worker or replica states look symmetric across zones, run `reduce.CoarsestPartition` + `reduce.Lump` to collapse them into a zone-level quotient model — and check `reduce.LumpabilityResidual` to catch silent configuration skew."*
4. **Recovery Bottleneck ROI (`solve.MFPTDelta`)**: *"Don't use naive averages on cyclic failure graphs; use `solve.MFPTToTarget` for exact mean recovery steps and `solve.MFPTDelta` to find the single transition improvement with the highest recovery ROI."*
5. **Stream Drift & Memory Testing (`infer.MarkovOrderTest`)**: *"When streaming logs might have non-stationary drift or hidden memory, weight transitions with an EMA half-life in `infer.NewEstimator` and check `infer.MarkovOrderTest` ($G^2$) to see if your state labels are missing history."*
6. **Optimal Control Synthesis (`control.PolicyIteration`)**: *"When you have both state transition probabilities and operating costs/rewards, wrap them in `control.NewMDP` and let `control.PolicyIteration` compute the provably optimal control policy."*

---

## 5. Tone, Do's & Don'ts

* **MUST**:
  - Place all examples in standard Go `example_test.go` files (`func Example...()` with deterministic `// Output:` blocks).
  - Lead every `Example...` doc comment with the **Situational Maxim** and a concrete Go systems story.
  - Use meaningful engineering state names (`"Start"`, `"Retry1"`, `"Healthy"`, `"CircuitOpen"`, `"Throttle"`) in package walkthroughs.
* **MUST NOT**:
  - Introduce raw Greek symbols in an example comment without first naming the systems quantity they represent.
  - Leave an `Example...` function without a verified `// Output:` assertion.

---

## 6. Examples Roadmap

| File | Package Story (`func Example()`) & Symbol Examples | Situational Maxim Demonstrated |
| :--- | :--- | :--- |
| `example_test.go` (`markov`) | **Canary Traffic Routing & Zone Rollup**: `Example`, `ExampleNewKernel`, `ExampleKernel_NormalizeRows`, `ExampleDistribution_Restrict`, `ExampleDistribution_Project`, `ExamplePartition_DistributorMatrix` | Maxim 1 — Conditioning & Rollups |
| `reduce/example_test.go` | **RPC Retry Folding & Multi-AZ Worker Quotienting**: `Example`, `ExampleTrace`, `ExampleTraceReward`, `ExampleLump`, `ExampleLumpabilityResidual`, `ExampleCoarsestPartition`, `ExampleAreBisimilar` | Maxim 2 & Maxim 3 — Temporal Censoring & Spatial Quotienting |
| `solve/example_test.go` | **Service Degradation & Recovery Bottleneck Analysis**: `Example`, `ExampleStationary`, `ExampleClassifyStates`, `ExampleMFPTToTarget`, `ExampleMFPTDelta`, `ExampleEntropyRate`, `ExampleKLDivergenceRate` | Maxim 4 — Recovery Bottleneck ROI |
| `infer/example_test.go` | **Live Event Stream Estimation & Hidden-Memory Detection**: `Example`, `ExampleNewEstimator`, `ExampleFromSequence`, `ExampleFromTransitions`, `ExampleMarkovOrderTest` | Maxim 5 — Stream Drift & Memory Testing |
| `control/example_test.go` | **Adaptive Backpressure & Queue Controller**: `Example`, `ExampleNewMDP`, `ExampleMDP_InducedKernel`, `ExamplePolicyIteration`, `ExampleValueIteration`, `ExampleMDPBisimulation` | Maxim 6 — Optimal Control Synthesis |

---

## 7. Persona Document Template

When adding a new `type: persona` document under `docs/personas/`, adhere to the following template (validated by `docs/personas/persona.shacl.ttl` via `make surface`):

```markdown
---
type: persona
title: <Persona Name & Role>
description: <One-sentence summary of who this persona is and what artifacts they govern>
stage: <Pedagogical or skill acquisition stage, e.g. Dreyfus Stage 2 (Advanced Beginner)>
resource: <Canonical reference URL>
tags: [persona, <tag2>, <tag3>]
timestamp: <ISO-8601 UTC>
---

# <Persona Title>

## 1. Skill Stage
## 2. Prior Knowledge
## 3. Mental Model & Vocabulary Bridge
## 4. Situational Maxims
## 5. Tone, Do's & Don'ts
## 6. Examples Roadmap
## Sources
```

---

## Sources

- Dreyfus, S. E. & Dreyfus, H. L. (1980). *A Five-Stage Model of the Mental Activities Involved in Directed Skill Acquisition*. Operations Research Center, UC Berkeley (ORC 80-2).
- [docs/wiki/core.md](../wiki/core.md), [docs/wiki/reduce.md](../wiki/reduce.md), [docs/wiki/solve.md](../wiki/solve.md), [docs/wiki/infer.md](../wiki/infer.md), [docs/wiki/control.md](../wiki/control.md).
