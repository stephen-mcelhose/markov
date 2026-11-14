---
type: concept
title: solve — Stationary Distributions, Fundamental Matrices, MFPT, SCCs & Entropy
description: The solve package implements the full suite of Markov chain analysis — from numerically stable stationary solvers (GTH) to mean first-passage times, state classification, and information-theoretic quantities.
tags: [stationary, gth, fundamental-matrix, mfpt, scc, entropy, solve]
timestamp: 2026-11-14T17:05:32Z
---

# solve — Stationary Distributions, Fundamental Matrices, MFPT, SCCs & Entropy

The `solve` package computes **dynamical and analytical properties** of a Markov kernel: steady-state behaviour, expected hitting times, structural decomposition, and information content.

**Primary sources**: Kemeny & Snell (1976), Stewart (1994), Norris (1997), Cover & Thomas (2006). Full citations: [references.md](references.md).

---

## 1. Stationary Distribution — `Stationary` / `MustStationary`

### Mathematical Object

A **stationary distribution** $\pi$ of kernel $P$ satisfies:

$$\pi P = \pi, \quad \sum_{i=0}^{n-1} \pi_i = 1, \quad \pi_i \ge 0$$

For an **ergodic** (irreducible, aperiodic) chain, $\pi$ is unique and $\pi_i = 1 / m_{i,i}$ where $m_{i,i}$ is the mean return time to state $i$. *(Kemeny & Snell Ch. IV §4.1)*

### Algorithm — GTH Elimination (Do Not Use Naive LU)

The standard naïve approach — solve $(P^T - I)\pi = 0$ subject to $\sum \pi_i = 1$ via LU or QR — suffers **catastrophic cancellation** on nearly-reducible chains where eigenvalues of $P$ cluster near 1 (ill-conditioned $I - P$).

**GTH (Grassmann–Taksar–Heyman) elimination** (1985) is **subtraction-free**: it exploits the stochastic structure to accumulate only additions during Gaussian elimination, completely avoiding the cancellation that plagues general LU. It is provably more accurate on stochastic matrices.

The algorithm:
1. Form the generator matrix $Q = P - I$ (off-diagonal entries are non-negative; diagonal entries are non-positive).
2. Perform Gaussian elimination on $Q$ **without subtraction** in the pivot step — each pivot uses only the complementary row sums.
3. Back-substitute to recover $\pi$; normalise.

*(Grassmann, Taksar & Heyman 1985; Stewart 1994 Ch. 2 §2.3)*

### Fallback — Power Iteration

For very large chains or when GTH's $O(n^2)$ memory is prohibitive, power iteration $\pi^{(t+1)} = \pi^{(t)} P$ converges to $\pi$ when the chain is ergodic. Stewart (1994) Ch. 4 covers Arnoldi/GMRES variants for stiff chains.

### Invariants

| Invariant | Formula |
| --------- | ------- |
| Left-eigenvector fixed point | $\|\pi P - \pi\|_\infty \le \varepsilon$ |
| Multi-step invariance | $\|\pi P^t - \pi\|_\infty \le \varepsilon$ for all $t \ge 1$ |
| Doubly stochastic uniformity | $\sum_i P_{i,j} = 1 \; \forall j \Rightarrow \pi_i = 1/n$ |
| Trace restriction compatibility | Restricted $\hat\pi = \text{Restrict}(\pi, A)$ is stationary for $P_A = \text{Trace}(P, A)$ |

### Reducible Chains

For non-ergodic chains with multiple recurrent communicating classes, there is a **basis** of stationary distributions — one per recurrent class. The solver behaviour (return all basis elements vs. error) is a **known unknown**; see the `known_unknowns.list` in the surface plan.

---

## 2. Fundamental Matrix — `FundamentalMatrix` / `AbsorbingFundamental`

### Ergodic Fundamental Matrix $Z$

For an ergodic chain with stationary distribution $\pi$, the **fundamental matrix** is:

$$Z = (I - P + \mathbf{1}\pi)^{-1}$$

where $\mathbf{1}\pi$ is the $n \times n$ matrix with every row equal to $\pi$. The additive correction makes the system uniquely solvable: $(I - P + \mathbf{1}\pi)$ has rank $n$ even though $(I - P)$ is singular. *(Kemeny & Snell Thm. 4.3.5)*

$Z$ can be interpreted as a regularised Green's function: $Z_{i,j} = \sum_{t=0}^\infty \bigl[(P^t)_{i,j} - \pi_j\bigr]$.

**Invariants**:

| Invariant | Formula |
| --------- | ------- |
| Two-sided inverse | $\|(I - P + \mathbf{1}\pi) Z - I\|_\infty \le \varepsilon$ and $\|Z(I - P + \mathbf{1}\pi) - I\|_\infty \le \varepsilon$ |
| Unit row sums | $Z\mathbf{1} = \mathbf{1}$ |
| Stationary left-annihilation | $\pi Z = \pi$ |

### Absorbing Fundamental Matrix $N$

For an absorbing chain with transient states $T$ and absorbing states $A = S \setminus T$, let $Q = P_{T,T}$ be the transient–transient sub-block. The **absorbing fundamental matrix** is:

$$N = (I - Q)^{-1} = \sum_{t=0}^\infty Q^t$$

$N_{i,j}$ is the expected number of visits to transient state $j$ before absorption, starting from transient state $i$. Convergence requires $\rho(Q) < 1$, guaranteed when $T$ is purely transient. *(Kemeny & Snell Ch. III §3.2)*

**Schur connection**: when $T = A^c$ (complement of the trace subset $A$), the Schur complement formula gives $bNd = P_A - a$ — directly linking `AbsorbingFundamental` to `reduce.Trace`.

**Invariants**:

| Invariant | Formula |
| --------- | ------- |
| Two-sided inverse | $\|(I - Q)N - I\|_\infty \le \varepsilon$ |
| Entrywise non-negativity | $N_{i,j} \ge -\varepsilon$ |
| Schur excursion equivalence | $bNd = P_A - a$ when $T = A^c$ |

---

## 3. Mean First-Passage Time — `MFPT` / `MFPTToTarget` / `MFPTDelta`

### Mathematical Object

For an ergodic chain, the **mean first-passage time** from $i$ to $j$ is the expected number of steps to reach $j$ for the first time starting from $i$:

$$M_{i,j} = \frac{Z_{j,j} - Z_{i,j}}{\pi_j} \quad (i \neq j), \qquad M_{i,i} = 0$$

*(Kemeny & Snell Thm. 4.4.7)*

$M_{i,j}$ satisfies the **first-step recurrence**:

$$M_{i,j} = 1 + \sum_{r \neq j} P_{i,r} M_{r,j} \quad (i \neq j)$$

The full $n \times n$ MFPT matrix is computed via the fundamental matrix $Z$.

### Kemeny's Constant

A striking identity: the quantity

$$K(P) = \sum_{j=0}^{n-1} M_{i,j} \pi_j = \text{tr}(Z) - 1$$

is **independent of the starting state $i$** — it depends only on the chain, not the initial state. *(Kemeny & Snell Cor. 4.4.8)* `MFPT` should assert this as a property test.

### Single-Target Solver — `MFPTToTarget`

For a fixed target $j^*$, solve the linear system:

$$m_{j^*} = 0, \quad m_i = 1 + \sum_{r \neq j^*} P_{i,r} m_r \quad (i \neq j^*)$$

equivalently: $m_T = N\mathbf{1}$ where $T = S \setminus \{j^*\}$ and $N = (I - P_{T,T})^{-1}$. This is cheaper than computing the full MFPT matrix.

### Perturbation Sensitivity — `MFPTDelta`

Computes the MFPT change under a row perturbation $\Delta P_{\text{from},\text{to}}(\varepsilon)$ (shifting probability mass $\varepsilon$ from row `from` to column `to`):

$$\Delta m = m(P + \Delta P) - m(P)$$

**Invariant**: $\varepsilon = 0 \Rightarrow \Delta m = 0$ (zero perturbation identity). **Invariant**: `to == target` and $\varepsilon > 0 \Rightarrow \Delta m_\text{from} \le 0$ (adding probability toward the target shortens MFPT from that state).

---

## 4. State Classification — `ClassifyStates`

### Mathematical Object

For states $i, j \in S$:

- **Communicate** ($i \leftrightarrow j$): $(P^t)_{i,j} > 0$ and $(P^s)_{j,i} > 0$ for some $t, s \ge 1$.
- **Strongly connected component (SCC)**: maximal communicating class.
- **Recurrent SCC**: closed communicating class (no outgoing transitions). Every state in a recurrent SCC is recurrent.
- **Transient state**: state in a non-closed SCC; probability of eventual return $< 1$.
- **Period** $d(i)$: $\gcd\{t \ge 1 : (P^t)_{i,i} > 0\}$; constant within a communicating class.

*(Norris Ch. 1 §1.4–1.5; Kemeny & Snell Ch. I §1.2)*

SCCs are found via **Tarjan's DFS algorithm** (1972) in $O(n + \text{edges})$ time.

### Return Type

```go
type StateClasses struct {
    SCCs      [][]int        // all SCCs as a valid Partition
    Recurrent [][]int        // recurrent SCCs only
    Transient []int          // transient states
    Periods   map[int]int    // period d(i) for each state i
}
```

`AsPartition()` casts `SCCs` as a `markov.Partition`.

### Invariants

| Invariant | Statement |
| --------- | --------- |
| Partition coverage | `SCCs` is a valid `markov.Partition` of $\{0,\dots,n-1\}$ |
| Recurrent class closure | $\forall C \in \text{Recurrent}, \forall i \in C, \forall j \notin C: P_{i,j} \le \varepsilon$ |
| Constant period | $i \leftrightarrow j \Rightarrow d(i) = d(j)$ |
| Stationary support | $\pi_i > \varepsilon \Rightarrow i \notin \text{Transient}$ |

---

## 5. Entropy Rate — `EntropyRate` / `KLDivergenceRate`

### Entropy Rate

For an ergodic chain at stationarity, the **entropy rate** measures the asymptotic uncertainty per step:

$$H(P) = -\sum_{i=0}^{n-1} \pi_i \sum_{j=0}^{n-1} P_{i,j} \log_2 P_{i,j} \qquad (0 \log_2 0 := 0)$$

*(Cover & Thomas, Thm. 4.2.4)*

**Bounds**: $0 \le H(P) \le \log_2 n$. Zero iff $P$ is a permutation matrix (deterministic); $\log_2 n$ iff $P_{i,j} = 1/n$ (uniform). **Data processing inequality**: $H(\hat{P}) \le H(P) + \varepsilon$ for any lumped kernel $\hat{P}$ — lumping can only reduce entropy.

### KL Divergence Rate

The **KL divergence rate** between kernels $P$ and $Q$ (sharing the same state space, with $P$'s stationary distribution $\pi^{(P)}$):

$$D_\text{KL}(P \| Q) = \sum_{i=0}^{n-1} \pi_i^{(P)} \sum_{j=0}^{n-1} P_{i,j} \log_2 \frac{P_{i,j}}{Q_{i,j}}$$

*(Cover & Thomas, Ch. 2 & Ch. 4)*

**Invariants**: Gibbs' inequality $D_\text{KL}(P \| Q) \ge -\varepsilon$; identity of indiscernibles $D_\text{KL}(P \| Q) = 0 \iff P_{i,\cdot} = Q_{i,\cdot}$ for all $i$ with $\pi_i^{(P)} > 0$. Undefined when $P_{i,j} > 0$ and $Q_{i,j} = 0$ (absolute continuity failure).

---

## Sources

- Kemeny, J. G. & Snell, J. L. (1976). *Finite Markov Chains*. Ch. III §3.2, Ch. IV §4.1–4.4, Thm. 4.3.5, 4.4.7, Cor. 4.4.8.
- Grassmann, W. K., Taksar, M. I. & Heyman, D. P. (1985). Regenerative analysis and steady state distributions for Markov chains. *Operations Research*, 33(5), 1107–1116.
- Stewart, W. J. (1994). *Introduction to the Numerical Solution of Markov Chains*. Ch. 2 §2.3, Ch. 4.
- Norris, J. R. (1997). *Markov Chains*. Ch. 1 §1.4–1.5.
- Tarjan, R. E. (1972). Depth-first search and linear graph algorithms. *SIAM J. Comput.*, 1(2), 146–160.
- Cover, T. M. & Thomas, J. A. (2006). *Elements of Information Theory* (2nd ed.). Thm. 4.2.4, Ch. 2.
- `docs/invariants/invariants.md` §4.2 (Solver Consistency).

---

## Situational Maxims (Dreyfus Stage 2 Examples)

See [../personas/dreyfus-stage-2.md](../personas/dreyfus-stage-2.md) for the full Advanced Beginner example roadmap:

- **`03_recovery_bottleneck/` (`MFPTToTarget` & `MFPTDelta`)**: *"Don't use naive averages on cyclic graphs; use `solve.MFPTToTarget` for exact recovery steps and `solve.MFPTDelta` to find the single edge fix with the highest ROI."*
