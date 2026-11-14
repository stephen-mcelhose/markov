---
type: concept
title: Core — Kernel, Distribution, Partition
description: The fundamental algebraic data types of the markov package — the row-stochastic transition matrix, probability simplex, and state-space partition.
tags: [kernel, distribution, partition, stochastic-matrix, core]
timestamp: 2026-11-14T17:05:32Z
---

# Core — Kernel, Distribution, Partition

The `markov` root package defines three algebraic objects that every subpackage (`reduce`, `solve`, `infer`, `control`) operates on. They encode the mathematical prerequisites for all downstream computations.

**Source**: Kemeny & Snell, *Finite Markov Chains* (1976), Ch. I–VI. See [references.md](references.md).

---

## 1. Kernel — Row-Stochastic Transition Matrix

### Mathematical Object

A **Markov kernel** is a square matrix $P \in \mathbb{R}^{n \times n}$ satisfying:

$$P_{i,j} \ge 0 \quad \forall i, j \in \{0, \dots, n-1\}$$

$$\sum_{j=0}^{n-1} P_{i,j} = 1 \quad \forall i \in \{0, \dots, n-1\}$$

$P_{i,j}$ is the one-step transition probability from state $i$ to state $j$. The $n$-step distribution is $\pi P^n$ where $\pi$ is an initial distribution. *(Kemeny & Snell, Ch. I §1.1)*

### Go Type

```go
type Kernel struct {
    P          *mat.Dense  // n×n row-stochastic matrix
    StateNames []string    // len = n, always non-nil after construction
}
```

### Constructors

| Function                          | Tolerance         | Behaviour                                                                   |
| --------------------------------- | ----------------- | --------------------------------------------------------------------------- |
| `NewKernel(P, names)`             | `tol = 1e-9`      | Clamps near-zero negatives $|v| \le \varepsilon$ to 0, then validates.      |
| `NewKernelWithTol(P, names, tol)` | user-supplied     | Same with explicit tolerance.                                                |
| `MustNewKernel(P, names)`         | `tol = 1e-9`      | Panics on error; use only when preconditions are guaranteed by construction. |

**Clamping rule**: entries in $(-\varepsilon, 0)$ are silently zeroed before the row-sum check. Entries $\le -\varepsilon$ produce `ErrNotStochastic`.

### Key Operations

#### `NormalizeRows(tol)`
Rescales each row to sum to 1:

$$\tilde{P}_{i,j} = \frac{P_{i,j}}{\sum_r P_{i,r}}$$

Returns `ErrNotStochastic` if any row sum $\le \varepsilon$ (all-zero row). Mutates in place, re-validates.

#### `Submatrix(rows, cols)`

$$P_{I,J} = [P_{i,j}]_{i \in I,\; j \in J}$$

Extracts the submatrix indexed by `rows` and `cols`. Used internally by `reduce.Trace` to form the Schur blocks $a, b, c, d$.

#### `NearlyEqual(a, b, tol)`

$$|a - b| \le \varepsilon$$

Absolute tolerance comparison. For probability values in $[0,1]$, absolute tolerance is appropriate.

### Invariants (from `docs/invariants/invariants.md`)

| Invariant                  | Formula                                                                        |
| -------------------------- | ------------------------------------------------------------------------------ |
| Non-negativity             | $P_{i,j} \ge -\varepsilon$                                                     |
| Row stochasticity          | $\left|\sum_j P_{i,j} - 1\right| \le \varepsilon$                             |
| Permutation equivariance   | Relabeling states via $\Pi$ preserves eigenvalues, entropy rate, MFPTs         |

---

## 2. Distribution — Probability Simplex

### Mathematical Object

A **probability distribution** over $n$ states is a vector $\pi \in \mathbb{R}^n$ in the standard simplex $\Delta^{n-1}$:

$$\pi_i \ge 0, \quad \sum_{i=0}^{n-1} \pi_i = 1$$

*(Kemeny & Snell, Ch. I §1.1)*

### Go Type

```go
type Distribution []float64
```

### Core Operations

#### `Restrict(subset, tol)` — Conditional Distribution on Subset

Given a non-empty subset $A \subseteq \{0,\dots,n-1\}$ with positive mass $\sum_{s \in A} \pi_s > \varepsilon$:

$$\pi_A[i] = \frac{\pi_{A_i}}{\sum_{s \in A} \pi_s}$$

This is the conditional distribution $\pi(\cdot \mid A)$. Returns `ErrInvalidDistribution` if the mass on $A$ is negligible.

**Key use**: after computing a trace kernel $P_A = a + b(I-c)^{-1}d$, the restricted distribution $\pi_A$ is stationary for $P_A$ whenever $\pi$ is stationary for $P$ (invariant 2.3 in `invariants.md`).

#### `Project(partition, tol)` — Aggregate over Macro-Blocks

Given a partition $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ with indicator matrix $V$:

$$\hat{\pi}[j] = \sum_{s \in B_j} \pi_s = (\pi V)_j$$

The projected distribution $\hat{\pi}$ sums to 1 algebraically (no renormalization needed for valid inputs). **Key use**: if $\pi$ is stationary for a strongly lumpable kernel $P$ on $\mathcal{B}$, then $\hat{\pi}$ is stationary for the lumped kernel $\hat{P}$ (invariant 3.5 in `invariants.md`).

### Invariants

| Invariant              | Formula                                            |
| ---------------------- | -------------------------------------------------- |
| Non-negativity         | $\pi_i \ge 0$                                      |
| Normalization          | $\left|\sum_i \pi_i - 1\right| \le \varepsilon$    |
| Restrict output valid  | $\pi_A$ satisfies both above                        |
| Project output valid   | $\hat{\pi}$ satisfies both above                    |

---

## 3. Partition — Equivalence Classes of States

### Mathematical Object

A **partition** of $\{0,\dots,n-1\}$ into $m$ blocks is a family $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ satisfying:

$$B_i \neq \emptyset, \quad B_i \cap B_j = \emptyset \; (i \neq j), \quad \bigcup_{i=0}^{m-1} B_i = \{0,\dots,n-1\}$$

*(Kemeny & Snell, Ch. VI §6.3)*

### Go Type

```go
type Partition [][]int   // Partition[i] = block Bᵢ, elements are state indices
```

### Algebraic Matrices

The partition induces two key matrices used throughout `reduce`:

#### Indicator (Collector) Matrix $V \in \{0,1\}^{n \times m}$

$$V_{s,j} = \mathbf{1}_{B_j}(s) = \begin{cases} 1 & s \in B_j \\ 0 & \text{otherwise} \end{cases}$$

Right-multiplying $P$ by $V$ sums each row's transition probabilities into macro-blocks: $(PV)_{s,j} = \sum_{t \in B_j} P_{s,t}$.

#### Distributor Matrix $U \in \mathbb{R}^{m \times n}$

$$U_{i,s} = \frac{w_s \, \mathbf{1}_{B_i}(s)}{\sum_{r \in B_i} w_r}$$

where $w$ is a weight vector (uniform $w_s = 1$ gives $U_{i,s} = 1/|B_i|$; a stationary distribution gives the minimum-information-loss projector).

**Left-inverse identity**: $U \cdot V = I_m$ (invariant 1.3 in `invariants.md`). This holds for *any* valid weight vector — it is a purely algebraic consequence of the partition structure.

#### Lumped Kernel $\hat{P} = UPV$

The composition $\hat{P} = UPV$ always produces a row-stochastic $m \times m$ matrix whenever $P$ is row-stochastic and $U, V$ are built from a valid partition. No renormalization is needed.

**Strong lumpability** (Kemeny & Snell §6.3): $P$ is strongly lumpable on $\mathcal{B}$ iff $PV = V\hat{P}$, equivalently:

$$\max_{B_i} \max_{s,s' \in B_i} \max_{B_j} |(PV)_{s,j} - (PV)_{s',j}| \le \varepsilon$$

When this holds, $\hat{P}$ is independent of the choice of weight vector $w$ in $U$.

### Invariants

| Invariant                      | Formula                                                           |
| ------------------------------ | ----------------------------------------------------------------- |
| Left-inverse identity          | $\|UV - I_m\|_\infty \le \varepsilon$                             |
| Discrete partition identity    | $\hat{P} = P$ when $\mathcal{B} = \{\{0\},\dots,\{n-1\}\}$      |
| Single-block identity          | $\hat{P} = [1]$ when $\mathcal{B} = \{\{0,\dots,n-1\}\}$         |

---

## Related Pages

- [reduce.md](reduce.md) — `Trace` and `Lump` consume `Kernel` and `Partition`
- [solve.md](solve.md) — `Stationary` returns a `Distribution`; `FundamentalMatrix` takes a `Kernel`
- [references.md](references.md) — full citation details

## Sources

- Kemeny, J. G. & Snell, J. L. (1976). *Finite Markov Chains*. Springer-Verlag. Ch. I §1.1, Ch. VI §6.3 pp. 123–140.
- `docs/invariants/invariants.md` §1 (Core Kernel & Partition Invariants).
