---
type: concept
title: References — Canonical Mathematical Sources
description: Master citation map linking every package, algorithm, and formula in the markov library to its authoritative textbook or paper.
tags: [citations, bibliography, markov, numerical-methods]
timestamp: 2026-11-14T17:05:32Z
---

# References — Canonical Mathematical Sources

This page is the authoritative anchor for every formula, algorithm, and invariant in the library.
Each implementation file **MUST** cite the entry below in its `doc.go` or leading comment.
When a naive textbook formula and a numerically stable variant diverge, the stable variant governs.

---

## Primary References by Package

### `markov/` — Core Types

| Algorithm / Concept                     | Source                                                                                                                                                 | Location                       |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------ |
| Row-stochastic transition matrix $P$    | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                         | Ch. I §1.1                     |
| Probability simplex $\Delta^{n-1}$      | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                         | Ch. I §1.1                     |
| Partition / equivalence classes         | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                         | Ch. VI §6.3 pp. 123–140        |
| Indicator matrix $V$, distributor $U$   | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                         | Ch. VI §6.3 pp. 123–140        |

---

### `reduce/` — Temporal Censoring & Spatial Quotienting

| Algorithm / Concept                             | Source                                                                                                                                                       | Location                             |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------ |
| Schur complement trace $P_A = a + b(I-c)^{-1}d$ | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                               | Ch. III §3.2 (absorbing fundamental) |
| Strong lumpability $PV = V\hat{P}$               | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                               | Ch. VI §6.3 pp. 123–140              |
| Weak lumpability $\hat{P} = UPV$                 | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                                                               | Ch. VI §6.3 pp. 123–140              |
| Probabilistic bisimulation                       | Larsen & Skou, "Bisimulation through Probabilistic Testing," *Information and Computation* 94(1):1–28 (1991)                                                | Def. 2, Thm. 3                       |
| Coarsest partition (partition refinement)        | Paige & Tarjan, "Three Partition Refinement Algorithms," *SIAM J. Comput.* 16(6):973–989 (1987)                                                            | Algorithm 3 ($O(m \log n)$)          |
| KL-divergence aggregation bounds                 | Deng, Mehta & Meyn, "Optimal Kullback–Leibler Aggregation via Spectral Theory of Markov Chains," *IEEE Trans. Autom. Control* 56(12):2793–2808 (2011)      | Thm. 1–3                             |

---

### `solve/` — Dynamics, Decomposition & Information

| Algorithm / Concept                                    | Source                                                                                                                 | Location                           |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| Stationary distribution $\pi P = \pi$                  | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                         | Ch. IV §4.1                        |
| **GTH elimination** (numerically stable stationary)    | Grassmann, Taksar & Heyman, "Regenerative Analysis and Steady State Distributions for Markov Chains," *Oper. Res.* 33(5):1107–1116 (1985) | Algorithm 1 |
| GTH implementation guidance                            | Stewart, *Introduction to the Numerical Solution of Markov Chains* (Princeton UP, 1994)                               | Ch. 2 §2.3                         |
| Power iteration / Arnoldi                              | Stewart, *Introduction to the Numerical Solution of Markov Chains* (Princeton UP, 1994)                               | Ch. 4                              |
| Fundamental matrix $Z = (I - P + \mathbf{1}\pi)^{-1}$ | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                         | Ch. IV §4.3 Thm. 4.3.5             |
| Absorbing fundamental $N = (I-Q)^{-1}$                | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                         | Ch. III §3.2                       |
| Mean first-passage time $M_{i,j}$                      | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                         | Thm. 4.4.7                         |
| Kemeny's constant $K(P) = \sum_j M_{i,j}\pi_j$        | Kemeny & Snell, *Finite Markov Chains* (1976)                                                                         | Cor. 4.4.8                         |
| SCCs / communication classes / periodicity             | Norris, *Markov Chains* (Cambridge UP, 1997)                                                                          | Ch. 1 §1.4–1.5                     |
| Tarjan's SCC algorithm                                 | Tarjan, "Depth-First Search and Linear Graph Algorithms," *SIAM J. Comput.* 1(2):146–160 (1972)                       | Algorithm                          |
| Entropy rate $H(P)$                                    | Cover & Thomas, *Elements of Information Theory* 2nd ed. (Wiley, 2006)                                                | Thm. 4.2.4                         |
| KL divergence rate $D_\text{KL}(P \| Q)$              | Cover & Thomas, *Elements of Information Theory* 2nd ed. (Wiley, 2006)                                                | Ch. 2 & Ch. 4                      |
| Spectral gap $1 - |\lambda_2|$, mixing time            | Levin, Peres & Wilmer, *Markov Chains and Mixing Times* (AMS, 2009)                                                   | Ch. 12–13                          |

---

### `infer/` — Statistical Estimation

| Algorithm / Concept                                    | Source                                                                                                   | Location            |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- | ------------------- |
| MLE of transition matrix $\hat{P}_{ij} = N_{ij}/N_{i\cdot}$ | Anderson & Goodman, "Statistical Inference about Markov Chains," *Ann. Math. Statist.* 28(1):89–110 (1957) | Thm. 2.1       |
| Dirichlet–multinomial (Laplace prior) MAP estimate     | Anderson & Goodman (1957); Murphy, *Machine Learning: A Probabilistic Perspective* (MIT Press, 2012)     | §3.4 (Murphy)       |
| Asymptotic distribution of count estimators            | Billingsley, *Statistical Inference for Markov Processes* (U. Chicago Press, 1961)                       | Ch. 1 Thm. 1.1      |
| Likelihood-ratio $G^2$ test for Markov order           | Anderson & Goodman (1957)                                                                                | §3.2 p. 99          |
| $G^2 \sim \chi^2_{n(n-1)^2}$ asymptotic distribution  | Billingsley (1961)                                                                                        | Thm. 3.1            |
| EMA exponential decay $\gamma = 2^{-1/h}$             | Murphy, *Machine Learning: A Probabilistic Perspective* (MIT Press, 2012)                                | §3.4 (Dirichlet-multinomial decay) |

---

### `control/` — Markov Decision Processes

| Algorithm / Concept                                    | Source                                                                                                                       | Location             |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- | -------------------- |
| MDP model $(S, A, \{P^a\}, R, \gamma)$                 | Puterman, *Markov Decision Processes* (Wiley, 1994)                                                                         | Ch. 1–3              |
| Bellman optimality equation                            | Puterman, *Markov Decision Processes* (Wiley, 1994)                                                                         | Thm. 6.2.7           |
| Value Iteration convergence, $\gamma$-contraction      | Puterman, *Markov Decision Processes* (Wiley, 1994)                                                                         | Thm. 6.3.1           |
| Policy Iteration, policy evaluation solve              | Howard, *Dynamic Programming and Markov Processes* (MIT Press, 1960); Puterman (1994)                                       | Puterman Ch. 6.4     |
| MDP bisimulation $s \sim s'$                           | Givan, Dean & Greig, "Equivalence Notions and Model Minimization in MDPs," *Artif. Intell.* 147(1–2):163–223 (2003)        | Def. 2 (exact bisim) |
| Approximate MDP bisimulation metric                    | Givan, Dean & Greig (2003)                                                                                                   | §4                   |

---

## Numerical Stability Notes

### Why GTH, not naive linear solve for stationary $\pi$?

The naïve approach solves $(P^T - I)\pi = 0$ or inverts $(I - P + \mathbf{1}\pi)$ directly.
Both suffer **catastrophic cancellation** on nearly-reducible chains where $P$ has eigenvalues near 1
(ill-conditioned $(I-P)$). GTH elimination (Grassmann, Taksar & Heyman 1985) avoids all subtractions
in the pivoting phase — it accumulates sums only by addition, exploiting the stochastic structure —
and is provably more accurate than LU decomposition on stochastic matrices.
See Stewart (1994) Ch. 2 §2.3 for the algorithm and conditioning analysis.

### Schur complement conditioning for `Trace`

$(I - P_{BB})^{-1}$ is singular whenever $B$ contains a closed recurrent communicating class
(the chain may never leave $B$). The implementation wraps the solve in `ErrSingularOperator`.
The error is not a numerical artifact — it is a mathematically correct signal that the trace
kernel does not exist for that $(P, A)$ pair.

### Fundamental matrix $Z$ vs. absorbing $N$

- **Ergodic chains**: use $Z = (I - P + \mathbf{1}\pi)^{-1}$. The additive correction $\mathbf{1}\pi$
  makes the system uniquely solvable (Kemeny & Snell Thm. 4.3.5). Verify $\pi Z = \pi$ and $Z\mathbf{1} = \mathbf{1}$.
- **Absorbing chains**: use $N = (I - Q)^{-1}$ where $Q = P_{TT}$ is the transient–transient block.
  $\rho(Q) < 1$ guarantees invertibility when $T$ contains only transient states.

---

## Sources

- Kemeny, J. G. & Snell, J. L. (1976). *Finite Markov Chains*. Springer-Verlag.
- Stewart, W. J. (1994). *Introduction to the Numerical Solution of Markov Chains*. Princeton University Press.
- Grassmann, W. K., Taksar, M. I. & Heyman, D. P. (1985). Regenerative analysis and steady state distributions for Markov chains. *Operations Research*, 33(5), 1107–1116.
- Norris, J. R. (1997). *Markov Chains*. Cambridge University Press.
- Levin, D. A., Peres, Y. & Wilmer, E. L. (2009). *Markov Chains and Mixing Times*. AMS.
- Larsen, K. G. & Skou, A. (1991). Bisimulation through probabilistic testing. *Information and Computation*, 94(1), 1–28.
- Paige, R. & Tarjan, R. E. (1987). Three partition refinement algorithms. *SIAM J. Comput.*, 16(6), 973–989.
- Deng, X., Mehta, P. G. & Meyn, S. P. (2011). Optimal Kullback–Leibler aggregation via spectral theory of Markov chains. *IEEE Trans. Autom. Control*, 56(12), 2793–2808.
- Tarjan, R. E. (1972). Depth-first search and linear graph algorithms. *SIAM J. Comput.*, 1(2), 146–160.
- Cover, T. M. & Thomas, J. A. (2006). *Elements of Information Theory* (2nd ed.). Wiley.
- Anderson, T. W. & Goodman, L. A. (1957). Statistical inference about Markov chains. *Ann. Math. Statist.*, 28(1), 89–110.
- Billingsley, P. (1961). *Statistical Inference for Markov Processes*. University of Chicago Press.
- Murphy, K. P. (2012). *Machine Learning: A Probabilistic Perspective*. MIT Press.
- Puterman, M. L. (1994). *Markov Decision Processes: Discrete Stochastic Dynamic Programming*. Wiley.
- Howard, R. A. (1960). *Dynamic Programming and Markov Processes*. MIT Press.
- Givan, R., Dean, T. & Greig, M. (2003). Equivalence notions and model minimization in Markov decision processes. *Artificial Intelligence*, 147(1–2), 163–223.
