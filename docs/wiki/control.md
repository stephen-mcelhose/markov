---
type: concept
title: control — MDPs, Value Iteration, Policy Iteration & MDP Bisimulation
description: The control package implements Markov Decision Processes with discounted reward — Bellman optimality via Value and Policy Iteration, and exact MDP bisimulation for model minimisation.
tags: [mdp, bellman, value-iteration, policy-iteration, bisimulation, control]
timestamp: 2026-11-14T17:05:32Z
---

# control — MDPs, Value Iteration, Policy Iteration & MDP Bisimulation

The `control` package adds **decision-making** to the Markov machinery: a controller selects actions to maximise expected discounted reward. The underlying chain structure is defined in [core.md](core.md); the bisimulation tools extend the state-minimisation ideas from [reduce.md](reduce.md).

**Primary sources**: Puterman (1994), Howard (1960), Givan, Dean & Greig (2003). Full citations: [references.md](references.md).

---

## 1. MDP — Markov Decision Process

### Mathematical Object

A **discounted MDP** is a tuple $\mathcal{M} = (S, A, \{P^a\}_{a \in A}, R, \gamma)$ where:

- $S = \{0, \dots, n-1\}$: finite state space ($n$ states)
- $A = \{0, \dots, k-1\}$: finite action space ($k$ actions)
- $P^a \in \mathbb{R}^{n \times n}$: row-stochastic transition kernel for action $a$
- $R \in \mathbb{R}^{n \times k}$: reward matrix, $R(s,a)$ = expected one-step reward in state $s$ under action $a$
- $\gamma \in [0, 1)$: discount factor

*(Puterman 1994, Ch. 1–3)*

**Note**: This formulation uses expected state-action rewards $R(s,a)$. The more general $R(s,a,s')$ formulation can be reduced to this by pre-computing $R(s,a) = \sum_{s'} P^a_{s,s'} r(s,a,s')$.

### Go Type

```go
type MDP struct {
    Actions     []string          // len = k
    Transitions []*markov.Kernel  // len = k; Transitions[a] is P^a
    Rewards     *mat.Dense        // n×k matrix
    Discount    float64           // γ ∈ [0,1)
}
```

### Induced Kernel — `InducedKernel`

A **deterministic stationary policy** $\pi: S \to A$ selects one action per state. The induced transition kernel is:

$$P^\pi_{s,s'} = P^{\pi(s)}_{s,s'}$$

`InducedKernel(policy, tol)` builds this $n \times n$ row-stochastic kernel, enabling evaluation of a policy using `solve.Stationary` and `solve.MFPT`.

---

## 2. Value Iteration — `ValueIteration`

### Mathematical Object

The **Bellman optimality equation** for discounted reward:

$$V^*(s) = \max_{a \in A} \left[ R(s,a) + \gamma \sum_{s' \in S} P^a_{s,s'} V^*(s') \right]$$

$V^*$ is the unique fixed point of the **Bellman optimality operator** $\mathcal{T}^*$, which is a $\gamma$-contraction in the $\ell^\infty$ norm. *(Puterman Thm. 6.2.7)*

### Algorithm

Value Iteration applies $\mathcal{T}^*$ repeatedly:

$$V_{k+1}(s) = \max_{a \in A} \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V_k(s') \right]$$

starting from $V_0 = 0$. The convergence criterion is $\|V_{k+1} - V_k\|_\infty \le \varepsilon(1-\gamma)/(2\gamma)$, which guarantees $\|V_k - V^*\|_\infty \le \varepsilon/2$. *(Puterman Thm. 6.3.1)*

The greedy policy at convergence:

$$\pi^*(s) = \arg\max_{a \in A} \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V^*(s') \right]$$

### Invariants

| Invariant | Statement |
| --------- | --------- |
| Bellman optimality fixed point | $\|V^* - \mathcal{T}^* V^*\|_\infty \le \varepsilon$ |
| Myopic horizon identity | $\gamma = 0 \Rightarrow V^*(s) = \max_a R(s,a)$ |
| Policy-value consistency | $(I - \gamma P^{\pi^*})V^* = R^{\pi^*} \pm \varepsilon$ |

---

## 3. Policy Iteration — `PolicyIteration`

### Algorithm

Policy Iteration alternates between **policy evaluation** and **policy improvement**:

1. **Evaluate** current policy $\pi_k$: solve the linear system

   $$V^{\pi_k} = (I - \gamma P^{\pi_k})^{-1} R^{\pi_k}$$

   where $R^{\pi_k}_s = R(s, \pi_k(s))$. The matrix $(I - \gamma P^{\pi_k})$ is invertible for $\gamma < 1$ since $\rho(\gamma P^{\pi_k}) = \gamma < 1$.

2. **Improve**: update policy greedily

   $$\pi_{k+1}(s) = \arg\max_{a \in A} \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V^{\pi_k}(s') \right]$$

Policy Iteration terminates in a finite number of iterations (at most $k^n$ policies) and converges faster than Value Iteration in practice. *(Howard 1960; Puterman Ch. 6.4)*

### Invariants

| Invariant | Statement |
| --------- | --------- |
| Monotonic policy improvement | $V^{\pi_{k+1}}(s) \ge V^{\pi_k}(s) - \varepsilon$ for all $s$ |
| Value Iteration equivalence | $\|V^*_\text{PI} - V^*_\text{VI}\|_\infty \le \varepsilon$ |

### Numerical Note

Policy evaluation requires solving an $n \times n$ linear system at each iteration. For small-to-medium $n$ ($\le 10^3$), LU decomposition is reliable. For large $n$, iterative solvers (GMRES) or block structure exploitation may be needed. $(I - \gamma P^{\pi_k})$ is well-conditioned for $\gamma$ away from 1; near $\gamma = 1$ the system approaches singularity.

---

## 4. MDP Bisimulation — `CoarsestPartition` / `MDPBisimulation`

### Mathematical Object

**MDP bisimulation** (Givan, Dean & Greig 2003): states $s \sim s'$ iff for every action $a \in A$:

1. **Reward equivalence**: $|R(s,a) - R(s',a)| \le \varepsilon$
2. **Transition equivalence**: for every block $B \in \mathcal{B}^*$,
   $$\left|\sum_{t \in B} P^a_{s,t} - \sum_{t \in B} P^a_{s',t}\right| \le \varepsilon$$

This extends the Kemeny–Snell strong lumpability condition ([reduce.md](reduce.md)) across **all** action kernels simultaneously, with the additional reward-matching constraint.

### Algorithm

Joint partition refinement across all action kernels:

1. Start from the coarsest partition consistent with rewards: initial blocks group states with identical reward vectors $R(s, \cdot)$.
2. Iteratively split blocks whenever two states disagree on aggregate transition probabilities into some block, **for any action** $a \in A$.
3. Terminate at fixed point $\mathcal{B}^*$.

This is equivalent to running `reduce.CoarsestPartition` simultaneously on all $k$ action kernels $\{P^a\}$ with the reward-induced initial partition.

### Invariants

| Invariant | Statement |
| --------- | --------- |
| Action-wise strong lumpability | `reduce.IsStronglyLumpable(P^a, B*, tol) == true` for all $a \in A$ |
| Optimal value invariance | $s, s' \in B_i \Rightarrow |V^*(s) - V^*(s')| \le 2\varepsilon/(1-\gamma)$ |
| Passive chain degeneration | $|A|=1 \land R \equiv 0 \Rightarrow \mathcal{B}^* = \text{reduce.CoarsestPartition}(P^{a_0}, \text{nil}, \varepsilon)$ |

The value invariance bound $2\varepsilon/(1-\gamma)$ shows that the approximation error from bisimulation-based model reduction blows up as $\gamma \to 1$ — a fundamental limitation of discounted-reward MDP compression. *(Givan, Dean & Greig 2003 Thm. 1)*

---

## Relationship to `reduce`

`control.CoarsestPartition` is a **multi-action generalisation** of `reduce.CoarsestPartition`:

| Aspect | `reduce.CoarsestPartition` | `control.CoarsestPartition` |
| ------ | -------------------------- | --------------------------- |
| Input | Single kernel $P$ | Tuple $\{P^a\}_{a \in A}$ + reward $R$ |
| Initial partition | User-supplied or trivial | Reward-induced grouping |
| Split criterion | Disagree on aggregate trans. probs | Disagree on agg. probs for ANY action OR differ in rewards |
| Output | Coarsest strongly lumpable partition | Coarsest MDP bisimulation partition |

---

## Sources

- Puterman, M. L. (1994). *Markov Decision Processes: Discrete Stochastic Dynamic Programming*. Wiley. Ch. 1–3, Thm. 6.2.7, Thm. 6.3.1, Ch. 6.4.
- Howard, R. A. (1960). *Dynamic Programming and Markov Processes*. MIT Press.
- Givan, R., Dean, T. & Greig, M. (2003). Equivalence notions and model minimization in Markov decision processes. *Artificial Intelligence*, 147(1–2), 163–223. Def. 2, Thm. 1.

---

## Situational Maxims (Dreyfus Stage 2 Examples)

See [../personas/dreyfus-stage-2.md](../personas/dreyfus-stage-2.md) for the full Advanced Beginner example roadmap:

- **`05_adaptive_throttler/` (`NewMDP` & `PolicyIteration`)**: *"When you have both transition probabilities and operating costs, wrap them in `control.NewMDP` and let `control.PolicyIteration` compute the provably optimal rule."*
