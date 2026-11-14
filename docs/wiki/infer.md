---
type: concept
title: infer — MLE, Dirichlet Estimation, EMA Decay & Markov Order Testing
description: The infer package estimates Markov transition matrices from observed sequences, with support for Dirichlet priors, exponential moving average decay, and likelihood-ratio tests for chain order.
tags: [mle, dirichlet, ema, markov-order, inference, estimation, infer]
timestamp: 2026-11-14T17:05:32Z
---

# infer — MLE, Dirichlet Estimation, EMA Decay & Markov Order Testing

The `infer` package estimates a Markov kernel $P$ from observed state sequences. It bridges empirical transition counts and the stochastic matrix types defined in [core.md](core.md).

**Primary sources**: Anderson & Goodman (1957), Billingsley (1961), Murphy (2012). Full citations: [references.md](references.md).

---

## 1. Transition Count Estimator — `Estimator`

### Mathematical Object

Given a sequence of states $(X_0, X_1, \dots, X_T)$ over state space $\{0, \dots, n-1\}$, the transition count matrix $C \in \mathbb{R}_{\ge 0}^{n \times n}$ accumulates:

$$C_{i,j} = \#\{t : X_t = i,\; X_{t+1} = j\}$$

The **maximum likelihood estimate** (MLE) of $P$ is:

$$\hat{P}_{i,j} = \frac{C_{i,j}}{\sum_r C_{i,r}} \quad \text{when } \sum_r C_{i,r} > 0$$

*(Anderson & Goodman 1957, Thm. 2.1)*

### Dirichlet–Multinomial Prior (Laplace Smoothing)

A **symmetric Dirichlet prior** with concentration $\alpha \ge 0$ gives the MAP estimate:

$$\hat{P}_{i,j} = \frac{C_{i,j} + \alpha}{\sum_r (C_{i,r} + \alpha)}$$

$\alpha = 0$: pure MLE (undefined for unobserved rows unless a fallback is used).  
$\alpha = 1$: Laplace smoothing (uniform additive pseudocounts).  
$\alpha > 0$: ensures $\hat{P}_{i,j} > 0$ for all $i,j$ regardless of observations.

**Uniform prior identity**: $C = 0 \land \alpha > 0 \Rightarrow \hat{P}_{i,j} = 1/n$ (uniform kernel).  
*(Murphy 2012, §3.4)*

### EMA Decay — Exponential Moving Average

For non-stationary processes, recent transitions should be weighted more heavily. The **EMA decay** scheme with half-life $h > 0$ applies decay factor $\gamma = 2^{-1/h}$ per observed transition:

$$C_{i,j}^{(t)} = \gamma C_{i,j}^{(t-1)} + \mathbf{1}_{(X_t = i,\; X_{t+1} = j)}$$

This is an exponential moving average with time constant $h$ transitions. After $h$ new observations, the weight of old data is halved.

$h = 0$ (no decay): $C^{(T)} = \sum_t \mathbf{1}_{(X_t=i,\; X_{t+1}=j)}$ — equivalent to standard counts.

**Decay formula**: at time $T-t$, observation at time $t$ receives weight $\gamma^{T-t}$. The sequence estimator $\hat{P}$ from `FromSequence` applies this weighting.

### Key Operations

| Function / Method | Formula | Note |
| ----------------- | ------- | ---- |
| `Observe(from, to)` | $C_{i,j} \mathrel{+}= 1$ (with optional decay) | Atomic count update |
| `ObserveSequence(states)` | $\text{Observe}(s_t, s_{t+1})$ for all $t$ | Sequential composition |
| `Kernel(tol)` | $\hat{P}_{i,j} = (C_{i,j}+\alpha)/\sum_r(C_{i,r}+\alpha)$ | Returns `ErrNotStochastic` on zero row |
| `FromTransitions(pairs, n, prior, tol)` | Batch count then `Kernel` | Stateless convenience wrapper |
| `FromSequence(states, n, prior, h, tol)` | EMA-weighted counts then `Kernel` | Full non-stationary estimator |

### Invariants

| Invariant | Statement |
| --------- | --------- |
| Non-negativity of counts | $C_{i,j}^{(t)} \ge 0$ always |
| Row stochasticity of output | $\hat{P}_{i,j} \ge 0 \land \|\sum_j \hat{P}_{i,j} - 1\| \le \varepsilon$ |
| Uniform prior identity | $C = 0 \land \alpha > 0 \Rightarrow \hat{P}_{i,j} = 1/n$ |
| Scale invariance (infinite memory, $\alpha=0$) | $\text{Kernel}(c \cdot C) = \text{Kernel}(C)$ for $c > 0$ |
| Sequential composition | $\text{ObserveSequence}([s_0,\dots,s_k,\dots,s_T]) \equiv \text{ObserveSequence}([s_0,\dots,s_k]) \circ \text{ObserveSequence}([s_k,\dots,s_T])$ |
| Asymptotic consistency | $T \to \infty, h = 0 \Rightarrow \hat{P} \xrightarrow{\text{a.s.}} P_\text{true}$ *(Billingsley 1961 Thm. 1.1)* |

---

## 2. Markov Order Test — `MarkovOrderTest`

### Mathematical Object

Given a sequence $(X_0, \dots, X_T)$, the **likelihood-ratio $G^2$ test** for Markov order tests $H_0$: the sequence is first-order Markov against $H_1$: it is second-order (requires knowing the previous two states).

Let $N_{i,j,k}$ = count of triples $(X_{t-1}=i, X_t=j, X_{t+1}=k)$ and define:

$$\hat{P}_{i,j,k} = \frac{N_{i,j,k}}{N_{i,j,\cdot}}, \qquad \hat{P}_{j,k} = \frac{N_{\cdot,j,k}}{N_{\cdot,j,\cdot}}$$

The test statistic is:

$$G^2 = 2 \sum_{i,j,k} N_{i,j,k} \ln \frac{\hat{P}_{i,j,k}}{\hat{P}_{j,k}} \;\sim\; \chi^2_{n(n-1)^2} \quad \text{under } H_0$$

*(Anderson & Goodman 1957, §3.2 p. 99; Billingsley 1961 Thm. 3.1)*

The degrees of freedom $n(n-1)^2$ arise because: for each of $n$ current states $j$, the conditional distribution over $k$ has $n-1$ free parameters given the history $i$, and $H_0$ constrains this to be independent of $i$.

### Invariants

| Invariant | Statement |
| --------- | --------- |
| Non-negativity | $G^2 \ge 0$ |
| Valid p-value | $p \in [0, 1]$ |
| First-order identity | $\hat{P}_{i,j,k} = \hat{P}_{j,k} \; \forall i,j,k \Rightarrow G^2 = 0 \land p = 1$ |

### Known Unknowns

- Handling zero counts: $N_{i,j,\cdot} = 0$ requires either skipping those triples or applying a small prior.
- Higher-order tests ($k$th order vs. $(k+1)$th order) require lifting the state space to $S^k$ — not yet in scope.

---

## Sources

- Anderson, T. W. & Goodman, L. A. (1957). Statistical inference about Markov chains. *Ann. Math. Statist.*, 28(1), 89–110. Thm. 2.1, §3.2 p. 99.
- Billingsley, P. (1961). *Statistical Inference for Markov Processes*. University of Chicago Press. Thm. 1.1, Thm. 3.1.
- Murphy, K. P. (2012). *Machine Learning: A Probabilistic Perspective*. MIT Press. §3.4 (Dirichlet-multinomial model).

---

## Situational Maxims (Dreyfus Stage 2 Examples)

See [../personas/dreyfus-stage-2.md](../personas/dreyfus-stage-2.md) for the full Advanced Beginner example roadmap:

- **`04_log_drift_detector/` (`FromSequence` & `MarkovOrderTest`)**: *"When streaming logs might have hidden memory or regime shifts, check `infer.MarkovOrderTest` ($G^2$) to see if your state labels are missing history."*
