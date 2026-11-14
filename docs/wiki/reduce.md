---
type: concept
title: reduce — Temporal Censoring, Spatial Quotienting, Bisimulation
description: The reduce package implements Schur complement trace (temporal censoring), Kemeny–Snell lumping (spatial quotienting), and Larsen–Skou probabilistic bisimulation.
tags: [trace, lump, bisimulation, schur, kemeny-snell, partition-refinement, reduce]
timestamp: 2026-11-14T17:05:32Z
---

# reduce — Temporal Censoring, Spatial Quotienting & Bisimulation

The `reduce` package compresses a Markov kernel in two orthogonal ways:

| Operator | Removes | Preserves | File |
| -------- | ------- | --------- | ---- |
| `Trace`  | Unobserved states (temporal censoring) | Transition probabilities among observed states | `trace.go` |
| `Lump`   | Fine-grained states within macro-blocks (spatial quotienting) | Aggregate block-to-block probabilities | `lump.go` |
| `CoarsestPartition` / `AreBisimilar` | Redundant distinguishable structure | Minimal bisimulation quotient | `bisim.go` |

Both `Trace` and `Lump` produce exact (not approximate) reduced kernels when their preconditions hold.
See [references.md](references.md) and [core.md](core.md) for foundational types.

---

## 1. Trace — Temporal Censoring via Schur Complement

### Mathematical Object

Given kernel $P$ on $S = A \cup B$ (disjoint), block-partition as:

$$P = \begin{pmatrix} a & b \\ d & c \end{pmatrix} \quad \begin{array}{l} \text{rows/cols } A \\ \text{rows/cols } B \end{array}$$

The **trace kernel** (watched/censored chain on $A$) is:

$$P_A = a + b(I - c)^{-1}d$$

where $(I-c)^{-1} = \sum_{k=0}^\infty c^k$ is the fundamental matrix for the sub-chain on $B$, converging when $B$ is **transient** ($\rho(c) < 1$). Each entry $(P_A)_{i,j}$ gives the probability that, starting from $i \in A$, the first return to $A$ is to state $j$. *(Kemeny & Snell Ch. III §3.2)*

### Existence Condition

$(I - c)$ is invertible iff $B$ contains **no closed recurrent communicating class**. If $B$ has a recurrent class, the chain may never leave $B$, and $P_A$ does not exist as a stochastic matrix — `Trace` returns `ErrSingularOperator`.

### Reward Folding — `TraceReward`

When per-state rewards $r$ are defined, the expected cumulative reward from state $i \in A$ (including reward at $i$ plus excursion rewards through $B$) is:

$$r_A[i] = r_a[i] + \bigl(b(I-c)^{-1}r_c\bigr)[i]$$

where $r_a = r|_A$, $r_c = r|_B$. This folds the Poisson-equation accumulated reward into the trace kernel, enabling exact model reduction for Markov Reward Processes.

### Verification — `IsTraceOf`

Checks $\|k.P - P_A\|_\infty \le \varepsilon$ by recomputing the Schur complement. Use to validate that a provided kernel was derived from a given parent.

### Key Invariants (from `docs/invariants/invariants.md` §2)

| Invariant | Statement |
| --------- | --------- |
| Full-space identity | $A = S \Rightarrow P_A = P$ |
| Tower law (transitivity of censoring) | $\text{Trace}(\text{Trace}(P, A), B) = \text{Trace}(P, B)$ for $B \subseteq A$ |
| Stationary restriction | $\pi P = \pi \Rightarrow \hat{\pi} P_A = \hat{\pi}$ where $\hat{\pi} = \text{Restrict}(\pi, A)$ |
| Excursion reward preservation | Total expected reward before absorption matches between $(P, r)$ and $(P_A, r_A)$ |

### Numerical Notes

The solve is `X.Solve(I - c, d)` using gonum's LU decomposition. This is well-conditioned when $B$ is transient with spectral radius $\rho(c)$ well below 1. Near-singular cases (e.g., slowly mixing transient chains) may accumulate error — verify with `IsTraceOf`.

---

## 2. Lump — Spatial Quotienting via Kemeny–Snell Projection

### Mathematical Object

Given kernel $P$ on $n$ states and partition $\mathcal{B} = \{B_0,\dots,B_{m-1}\}$, with indicator matrix $V \in \{0,1\}^{n \times m}$ and distributor matrix $U \in \mathbb{R}^{m \times n}$ (see [core.md](core.md)), the **lumped kernel** is:

$$\hat{P} = UPV$$

$\hat{P}$ is always row-stochastic when $P$ is row-stochastic and $U$ rows sum to 1 — algebraically, no numerical normalisation is required.

**Uniform weights**: $U_{i,s} = 1/|B_i|$ for $s \in B_i$. Correct for any strongly lumpable $(P, \mathcal{B})$.

**Stationary weights**: $U_{i,s} = \pi_s / \sum_{r \in B_i} \pi_r$ for $s \in B_i$. Minimises KL-divergence information loss for weakly lumpable chains (Deng, Mehta & Meyn 2011). Pass the stationary distribution as the `weights` argument.

*(Kemeny & Snell Ch. VI §6.3)*

### Strong Lumpability — `IsStronglyLumpable` / `LumpabilityResidual`

**Strong lumpability** (Kemeny & Snell §6.3): $P$ is strongly lumpable on $\mathcal{B}$ iff for every pair $s, s' \in B_i$ and every block $B_j$:

$$(PV)_{s,j} = (PV)_{s',j}$$

equivalently $PV = V\hat{P}$. When this holds, $\hat{P}$ is independent of the choice of $U$.

The **lumpability residual** quantifies deviation:

$$R(P, \mathcal{B}) = \max_{B_i} \max_{s,s' \in B_i} \max_{B_j} |(PV)_{s,j} - (PV)_{s',j}|$$

`IsStronglyLumpable` returns `R(P, \mathcal{B}) \le \varepsilon`.

### Key Invariants (from `docs/invariants/invariants.md` §3)

| Invariant | Statement |
| --------- | --------- |
| Discrete partition identity | $\mathcal{B} = \{\{0\},\dots,\{n-1\}\} \Rightarrow \hat{P} = P$ |
| Single-block identity | $\mathcal{B} = \{\{0,\dots,n-1\}\} \Rightarrow \hat{P} = [1]$ |
| Strong lumpability equivalence | $R(P,\mathcal{B}) \le \varepsilon \iff \|PV - V\hat{P}\|_\infty \le \varepsilon$ |
| Distributor weight invariance | Strong lumpable $\Rightarrow$ $\hat{P}$ same for any valid $U$ |
| Multi-step commutation | $P^t V = V\hat{P}^t$ when $PV = V\hat{P}$ |
| Stationary projection | $\pi P = \pi \land PV = V\hat{P} \Rightarrow \hat{\pi}\hat{P} = \hat{\pi}$ where $\hat{\pi} = \pi V$ |

### Spatial–Temporal Commutativity (Invariant 4.1)

When $\mathcal{B}$ is a strongly lumpable partition and $\mathcal{I} \subseteq \{0,\dots,m-1\}$ indexes a subset of macro-blocks with expanded micro-state set $A = \bigcup_{i \in \mathcal{I}} B_i$:

$$\text{Trace}(\text{Lump}(P, \mathcal{B}), \mathcal{I}) = \text{Lump}(\text{Trace}(P, A), \mathcal{B}|_A)$$

Lumping then tracing commutes with tracing then lumping. Tested in `TestLumpTraceCommutativity`.

---

## 3. Bisimulation — Coarsest Partition & Probabilistic Equivalence

### Mathematical Object

**Probabilistic bisimulation** (Larsen & Skou 1991): two Markov kernels $P_1$ and $P_2$ are bisimilar iff their minimal quotient kernels $\hat{P}_1$ and $\hat{P}_2$ (induced by their respective coarsest strongly lumpable partitions) are isomorphic up to state relabelling:

$$k_1 \sim k_2 \iff \exists \sigma \text{ permutation: } |\hat{P}_1[i,j] - \hat{P}_2[\sigma(i),\sigma(j)]| \le \varepsilon \; \forall i,j$$

### Algorithm — `CoarsestPartition`

Implements **iterative Larsen–Skou / Paige–Tarjan partition refinement**:

1. Start from initial partition (or trivial single block $\{S\}$ when `initial = nil`).
2. For each block, compute each state's **block-transition signature**: $\text{sig}[s][j] = \sum_{t \in B_j} P_{s,t}$.
3. Split any block where two states' signatures disagree by more than $\varepsilon$.
4. Repeat until stable (fixed point = coarsest strongly lumpable partition $\mathcal{B}^*$).

The fixed-point partition satisfies `IsStronglyLumpable(k, B*, tol) == true`.

**Starting from nil** (trivial partition) gives the **unique minimal probabilistic bisimulation quotient** of $k$ — the coarsest possible strongly lumpable partition, independent of any prior grouping.

**Complexity**: $O(n^2 m)$ per iteration in the current implementation (signature computation is $O(n \cdot m)$ per state). The theoretically optimal Paige–Tarjan algorithm achieves $O(nm \log n)$ overall; see [references.md](references.md).

### Algorithm — `AreBisimilar`

1. Compute $\mathcal{B}_1^* = \text{CoarsestPartition}(k_1, \text{nil}, \varepsilon)$ and $\mathcal{B}_2^* = \text{CoarsestPartition}(k_2, \text{nil}, \varepsilon)$.
2. If $|\mathcal{B}_1^*| \neq |\mathcal{B}_2^*|$: return `false`.
3. Project: $\hat{P}_1 = \text{Lump}(k_1, \mathcal{B}_1^*)$, $\hat{P}_2 = \text{Lump}(k_2, \mathcal{B}_2^*)$.
4. Check **matrix isomorphism** via backtracking with row-multiset signatures for pruning: find permutation $\sigma$ such that $|\hat{P}_1[i,j] - \hat{P}_2[\sigma(i),\sigma(j)]| \le \varepsilon$.

**Isomorphism check**: $O(m!)$ worst case, but row-multiset pruning (sorted row distributions as invariants) makes it fast in practice for small $m$. Chains with large minimal quotients ($m \gg 10$) may be slow.

### Key Invariants

| Invariant | Statement |
| --------- | --------- |
| Reflexivity | `AreBisimilar(k, k, tol) == true` always |
| Symmetry | `AreBisimilar(k1, k2, tol) == AreBisimilar(k2, k1, tol)` |
| Transitivity | `AreBisimilar(k1,k2) && AreBisimilar(k2,k3) ⇒ AreBisimilar(k1,k3)` |
| Quotient bisimilarity | $k$ is always bisimilar to `Lump(k, CoarsestPartition(k, nil))` |
| Idempotence | `CoarsestPartition(Lump(k, B*), nil)` = discrete partition $\{\{0\},\dots,\{m-1\}\}$ |
| Strong lumpability at fixed point | `IsStronglyLumpable(k, CoarsestPartition(k, nil), tol) == true` |

---

## Sources

- Kemeny, J. G. & Snell, J. L. (1976). *Finite Markov Chains*. Springer-Verlag. Ch. III §3.2, Ch. VI §6.3 pp. 123–140.
- Larsen, K. G. & Skou, A. (1991). Bisimulation through probabilistic testing. *Information and Computation*, 94(1), 1–28.
- Paige, R. & Tarjan, R. E. (1987). Three partition refinement algorithms. *SIAM J. Comput.*, 16(6), 973–989.
- Deng, X., Mehta, P. G. & Meyn, S. P. (2011). Optimal KL aggregation via spectral theory of Markov chains. *IEEE Trans. Autom. Control*, 56(12), 2793–2808.
- `docs/invariants/invariants.md` §2–4.

---

## Situational Maxims (Dreyfus Stage 2 Examples)

See [../personas/dreyfus-stage-2.md](../personas/dreyfus-stage-2.md) for the full Advanced Beginner example roadmap:

- **`01_retry_folding/` (`Trace` & `TraceReward`)**: *"When internal retry states clutter your metrics, use `reduce.Trace` and `reduce.TraceReward` to collapse them into exact end-to-end success probabilities and latency."*
- **`02_symmetric_workers/` (`CoarsestPartition`, `Lump`, `LumpabilityResidual`)**: *"When states look duplicated across zones, run `reduce.CoarsestPartition` + `reduce.Lump` to verify if they are behaviorally identical — and check `reduce.LumpabilityResidual` to catch silent zone skew."*
