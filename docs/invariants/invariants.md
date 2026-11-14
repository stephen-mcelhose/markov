# Mathematical Invariants

This document specifies the algebraic and probabilistic invariants that **MUST** hold across `markov` and its subpackages within floating-point tolerance $\varepsilon = \text{tol}$.

Every property-based test (`pgregory.net/rapid`) and fuzz target **MUST** assert the relevant invariants below on all generated inputs and outputs.


This document MUST be udpated to reflect as new invariants are added to the codebase.
---

## 1. Core Kernel & Partition Invariants (`markov`)

### 1.1 Row Stochasticity
Every valid `Kernel` with transition matrix $P \in \mathbb{R}^{n \times n}$ represents a right-stochastic transition operator:
$$P_{i, j} \ge -\varepsilon \quad \forall i, j \in \{0, \dots, n-1\}$$
$$\left| \sum_{j=0}^{n-1} P_{i, j} - 1 \right| \le \varepsilon \quad \forall i \in \{0, \dots, n-1\}$$

### 1.2 Distribution Conservation
Every valid probability distribution $\pi \in \mathbb{R}^n$ satisfies $\pi_i \ge -\varepsilon$ and $\left|\sum_{i=0}^{n-1} \pi_i - 1\right| \le \varepsilon$. Both `RestrictDistribution` and `ProjectDistribution` **MUST** output a valid probability distribution summing to $1 \pm \varepsilon$.

### 1.3 Collector–Distributor Left-Inverse Identity
For any valid partition $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ of $\{0, \dots, n-1\}$ with $n \times m$ indicator matrix $V$ and any valid $m \times n$ distributor matrix $U$:
$$U \cdot V = I_m$$
Equivalently, $\|(U \cdot V) - I_m\|_\infty \le \varepsilon$.

### 1.4 Permutation Equivariance
Let $\Pi \in \{0, 1\}^{n \times n}$ be any permutation matrix relabeling states $0, \dots, n-1$. Transforming $P \mapsto \Pi P \Pi^\top$ and relabeling subsets/partitions consistently **MUST** preserve eigenvalues, entropy rate, MFPTs, and induced reduced kernels up to the corresponding index permutation.

---

## 2. Temporal Censoring Invariants (`markov/reduce` — `Trace`)

Let $A \subseteq \{0, \dots, n-1\}$ be a non-empty subset with complement $A^c = \{0, \dots, n-1\} \setminus A$, and let $P_A = a + b(I - c)^{-1}d$ be the induced trace kernel whenever $(I - c)$ is invertible.

### 2.1 Full-Space Identity
When $A = \{0, \dots, n-1\}$ ($A^c = \emptyset$):
$$P_A = P$$

### 2.2 Tower Law (Transitivity of Censoring)
For any nested chain of non-empty subsets $B \subseteq A \subseteq \{0, \dots, n-1\}$, censoring $A^c$ first and then censoring $A \setminus B$ **MUST** equal censoring $B^c$ directly:
$$\text{Trace}(\text{Trace}(P, A), B) = \text{Trace}(P, B)$$

### 2.3 Stationary Distribution Restriction
If $\pi$ is a stationary distribution of $P$ ($\pi P = \pi$) with strictly positive mass on $A$ ($\sum_{s \in A} \pi_s > 0$), then the restricted distribution $\hat{\pi} = \text{RestrictDistribution}(\pi, A)$ is stationary for $P_A$:
$$\hat{\pi} P_A = \hat{\pi}$$

### 2.4 Hitting-Probability Preservation
For any starting state $s \in A$ and target state $t \in A$, the probability of ever reaching $t$ from $s$ (and the relative hitting probabilities among states in $A$) in $P_A$ is identical to that in $P$.

### 2.5 Excursion Reward Preservation (MRPs)
When step rewards/costs $r = [r_a; r_c]$ are folded via $r_A = r_a + b(I - c)^{-1}r_c$, the total expected accumulated reward before absorption in $(P_A, r_A)$ **MUST** equal the total expected accumulated reward in $(P, r)$ for all starting states $s \in A$.

---

## 3. Spatial Quotienting Invariants (`markov/reduce` — `Lump`)

Let $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ be a valid partition of $\{0, \dots, n-1\}$ with indicator matrix $V$, distributor matrix $U$, and projected kernel $\hat{P} = U P V$.

### 3.1 Trivial Partition Identities
* **Discrete partition** ($\mathcal{B} = \{\{0\}, \{1\}, \dots, \{n-1\}\}$): $\hat{P} = P$.
* **Single-block partition** ($\mathcal{B} = \{\{0, \dots, n-1\}\}$): $\hat{P} = [1]$.

### 3.2 Kemeny–Snell Strong Lumpability Equivalence
The strong lumpability residual
$$R(P, \mathcal{B}) = \max_{B_i} \max_{s, s' \in B_i} \max_{B_j} \left| (PV)_{s, j} - (PV)_{s', j} \right|$$
satisfies $R(P, \mathcal{B}) \le \varepsilon$ if and only if:
$$\|P V - V \hat{P}\|_\infty \le \varepsilon$$

### 3.3 Distributor Weight Invariance Under Strong Lumpability
When $P$ is strongly lumpable on $\mathcal{B}$ ($R(P, \mathcal{B}) = 0$), $\hat{P} = U P V$ **MUST** be invariant to the choice of valid block weights in $U$.

### 3.4 Multi-Step Chapman–Kolmogorov Commutation
When $P$ is strongly lumpable on $\mathcal{B}$ ($P V = V \hat{P}$), spatial projection commutes with time evolution for every horizon $t \ge 1$:
$$P^t V = V \hat{P}^t$$

### 3.5 Stationary Distribution Projection
If $\pi P = \pi$ and $P$ is strongly lumpable on $\mathcal{B}$ (or weakly lumpable using $U$ weighted by $\pi$), then $\hat{\pi} = \text{ProjectDistribution}(\pi, \mathcal{B}) = \pi V$ is stationary for $\hat{P}$:
$$\hat{\pi} \hat{P} = \hat{\pi}$$

---

## 4. Compositional & Cross-Package Invariants

### 4.1 Spatial–Temporal Commutativity (`Lump` $\circ$ `Trace` $=$ `Trace` $\circ$ `Lump`)
Let $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ be a partition on which $P$ is strongly lumpable, and let $\mathcal{I} \subseteq \{0, \dots, m-1\}$ be a subset of macro-blocks with expanded micro-state set $A = \bigcup_{i \in \mathcal{I}} B_i$. Then spatial lumping and temporal tracing commute:
$$\text{Trace}(\text{Lump}(P, \mathcal{B}), \mathcal{I}) = \text{Lump}(\text{Trace}(P, A), \mathcal{B}|_A)$$
where $\mathcal{B}|_A$ is the restriction of $\mathcal{B}$ to the blocks indexed by $\mathcal{I}$.

### 4.2 Solver Consistency (`markov/solve`)
* **Stationary Eigen-Identity:** Every returned stationary vector $\pi$ satisfies $\|\pi P - \pi\|_\infty \le \varepsilon$.
* **Fundamental Matrix Identity:** For ergodic chains, $Z = (I - P + \mathbf{1}\pi)^{-1}$ satisfies $(I - P + \mathbf{1}\pi) Z = I$. For absorbing chains with transient submatrix $Q$, $N = (I - Q)^{-1}$ satisfies $(I - Q) N = I$.
* **Data Processing Inequality (Entropy Rate):** Under any deterministic state projection $Y_t = \phi(X_t)$, the entropy rate of the reduced process cannot exceed the entropy rate of the fine process: $H(\hat{P}) \le H(P) + \varepsilon$.
Use code with caution.
Ready for the repository organization discussion whenever you are—let me know what directories, Makefile targets, runbook templates, or skills you want to structure next!
