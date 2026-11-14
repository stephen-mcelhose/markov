---
type: decision
title: Initial Public Surface Design
description: Complete file tree with every exported type, method, and top-level function for all packages — the target API surface for markov v0.1.
resource: https://github.com/stephen-mcelhose/catrace/issues/59
tags: [design, api, surface, kernel, reduce, solve, infer, control]
timestamp: 2026-11-14T06:44:10Z
---

# Initial Public Interface

We are building a math library, it is very easy to at least estimate our surface design.

Here is the full repository tree with every expected .go file and its exported types, methods, and top-level functions hanging directly off each file—respecting Go's single-package method rule (Kernel, Distribution, and Partition have methods in root markov, while subpackages expose top-level functions plus methods on their own domain types like MDP and Estimator).
```text
markov/
├── go.mod / go.sum
├── Makefile
├── LICENSE
├── README.md
├── AGENTS.md
├── doc.go
├── errors.go
│   ├── var ErrUnimplemented error
│   ├── var ErrNotStochastic error
│   ├── var ErrInvalidDistribution error
│   ├── var ErrInvalidPartition error
│   └── var ErrSingularOperator error
├── kernel.go
│   ├── type Kernel struct { P *mat.Dense; StateNames []string }
│   ├── func NewKernel(P *mat.Dense, names []string) (*Kernel, error)
│   │   └── $P \in \mathbb{R}^{n \times n},\; P_{i,j} \ge 0,\; \sum_j P_{i,j} = 1$
│   ├── func NewKernelWithTol(P *mat.Dense, names []string, tol float64) (*Kernel, error)
│   │   └── $P_{i,j} \ge -\varepsilon,\; \left|\sum_j P_{i,j} - 1\right| \le \varepsilon$
│   ├── func MustNewKernel(P *mat.Dense, names []string) *Kernel
│   ├── func (k *Kernel) NumStates() int
│   │   └── $n = |S|$
│   ├── func (k *Kernel) Clone() *Kernel
│   ├── func (k *Kernel) Submatrix(rows, cols []int) *mat.Dense
│   │   └── $P_{I, J} = [P_{i,j}]_{i \in I,\, j \in J}$
│   ├── func (k *Kernel) NormalizeRows(tol float64) error
│   │   └── $\tilde{P}_{i,j} = \frac{P_{i,j}}{\sum_r P_{i,r}}$
│   ├── func Submatrix(P *mat.Dense, rows, cols []int) *mat.Dense
│   │   └── $P_{I, J} = [P_{i,j}]_{i \in I,\, j \in J}$
│   ├── func NormalizeRows(P *mat.Dense, tol float64) error
│   │   └── $\tilde{P}_{i,j} = \frac{P_{i,j}}{\sum_r P_{i,r}}$
│   ├── func CanonicalSubset(subset []int) []int
│   │   └── $\text{sort}(\text{unique}(A))$
│   └── func NearlyEqual(a, b, tol float64) bool
│       └── $|a - b| \le \varepsilon$
├── distribution.go
│   ├── type Distribution []float64
│   ├── func NewDistribution(probs []float64, tol float64) (Distribution, error)
│   │   └── $\pi \in \Delta^{n-1}:\; \pi_i \ge 0,\; \sum_{i=0}^{n-1} \pi_i = 1$
│   ├── func MustNewDistribution(probs []float64, tol float64) Distribution
│   ├── func (d Distribution) Validate(tol float64) error
│   │   └── $\pi_i \ge -\varepsilon,\; \left|\sum_i \pi_i - 1\right| \le \varepsilon$
│   ├── func (d Distribution) Restrict(subset []int, tol float64) (Distribution, error)
│   │   └── $\pi_A[i] = \frac{\pi_{A_i}}{\sum_{s \in A} \pi_s}$
│   ├── func (d Distribution) Project(p Partition, tol float64) (Distribution, error)
│   │   └── $\hat{\pi}[j] = \sum_{s \in B_j} \pi_s = (\pi V)_j$
│   ├── func ValidateDistribution(pi Distribution, tol float64) error
│   │   └── $\pi_i \ge -\varepsilon,\; \left|\sum_i \pi_i - 1\right| \le \varepsilon$
│   ├── func RestrictDistribution(pi Distribution, subset []int, tol float64) (Distribution, error)
│   │   └── $\pi_A[i] = \frac{\pi_{A_i}}{\sum_{s \in A} \pi_s}$
│   └── func ProjectDistribution(pi Distribution, p Partition, tol float64) (Distribution, error)
│       └── $\hat{\pi}[j] = \sum_{s \in B_j} \pi_s = (\pi V)_j$
├── partition.go
│   ├── type Partition [][]int
│   ├── func (p Partition) Validate(n int) error
│   │   └── $B_i \neq \emptyset,\; B_i \cap B_j = \emptyset \; (i \neq j),\; \bigcup_{i=0}^{m-1} B_i = \{0,\dots,n-1\}$
│   ├── func (p Partition) IndicatorMatrix(n int) (*mat.Dense, error)
│   │   └── $V \in \{0,1\}^{n \times m},\; V_{s,j} = \mathbf{1}_{B_j}(s)$
│   ├── func (p Partition) DistributorMatrix(n int, weights Distribution, tol float64) (*mat.Dense, error)
│   │   └── $U \in \mathbb{R}^{m \times n},\; U_{i,s} = \frac{w_s \mathbf{1}_{B_i}(s)}{\sum_{r \in B_i} w_r},\; U V = I_m$
│   ├── func ValidatePartition(p Partition, n int) error
│   │   └── $B_i \neq \emptyset,\; B_i \cap B_j = \emptyset \; (i \neq j),\; \bigcup_{i=0}^{m-1} B_i = \{0,\dots,n-1\}$
│   ├── func IndicatorMatrix(p Partition, n int) (*mat.Dense, error)
│   │   └── $V \in \{0,1\}^{n \times m},\; V_{s,j} = \mathbf{1}_{B_j}(s)$
│   └── func DistributorMatrix(p Partition, n int, weights Distribution, tol float64) (*mat.Dense, error)
│       └── $U \in \mathbb{R}^{m \times n},\; U_{i,s} = \frac{w_s \mathbf{1}_{B_i}(s)}{\sum_{r \in B_i} w_r},\; U V = I_m$
├── known_unknowns.list
│   ├── Exact field/method signatures in uninspected catrace/kernel.go (default tolerance constant, validation rules)
│   ├── Whether continuous-time Generator (Q) and Uniformize(Q, gamma) belong in root markov or a separate ctmc package
│   └── Whether sparse matrix support (CSR/DOK) is needed alongside dense *mat.Dense or deferred to v2
├── docs/
│   └── invariants.md
├── reduce/
│   ├── doc.go
│   ├── trace.go
│   │   ├── func Trace(k *markov.Kernel, subset []int, tol float64) (*markov.Kernel, error)
│   │   │   └── $P_A = a + b(I - c)^{-1}d$
│   │   ├── func MustTrace(k *markov.Kernel, subset []int, tol float64) *markov.Kernel
│   │   │   └── $P_A = a + b(I - c)^{-1}d$
│   │   ├── func IsTraceOf(k, parent *markov.Kernel, subset []int, tol float64) (bool, error)
│   │   │   └── $\left\| P_k - \left(a + b(I - c)^{-1}d\right) \right\|_\infty \le \varepsilon$
│   │   └── func TraceReward(k *markov.Kernel, rewards []float64, subset []int, tol float64) ([]float64, error)
│   │       └── $r_A = r_a + b(I - c)^{-1}r_c$
│   ├── lump.go
│   │   ├── func Lump(k *markov.Kernel, p markov.Partition, names []string, weights markov.Distribution, tol float64) (*markov.Kernel, error)
│   │   │   └── $\hat{P} = U P V$
│   │   ├── func MustLump(k *markov.Kernel, p markov.Partition, names []string, weights markov.Distribution, tol float64) *markov.Kernel
│   │   │   └── $\hat{P} = U P V$
│   │   ├── func IsStronglyLumpable(k *markov.Kernel, p markov.Partition, tol float64) (bool, error)
│   │   │   └── $\|P V - V \hat{P}\|_\infty \le \varepsilon$
│   │   └── func LumpabilityResidual(k *markov.Kernel, p markov.Partition) (float64, error)
│   │       └── $R(P, \mathcal{B}) = \max_{B_i} \max_{s, s' \in B_i} \max_{B_j} |(PV)_{s,j} - (PV)_{s',j}|$
│   ├── bisim.go
│   │   ├── func CoarsestPartition(k *markov.Kernel, initial markov.Partition, tol float64) (markov.Partition, error)
│   │   │   └── $\mathcal{B}^* = \sup \{ \mathcal{B} \preceq \mathcal{B}_0 : P V_{\mathcal{B}} = V_{\mathcal{B}} \hat{P} \}$
│   │   └── func AreBisimilar(k1, k2 *markov.Kernel, tol float64) (bool, error)
│   │       └── $\exists \mathcal{B}_1, \mathcal{B}_2:\; \hat{P}_1 \cong \hat{P}_2$
│   └── known_unknowns.list
│       ├── Spectral & metastable aggregation algorithms (PCCA+, Simon–Ando NCD decomposition)
│       ├── Continuous bisimulation metrics (Kantorovich / Wasserstein pseudometric distance)
│       └── Weak lumpability verification (checking pi * P^k * V = pi_hat * P_hat^k across horizons)
├── solve/
│   ├── doc.go
│   ├── stationary.go
│   │   ├── func Stationary(k *markov.Kernel, tol float64) (markov.Distribution, error)
│   │   │   └── $\pi P = \pi,\; \sum_i \pi_i = 1,\; \pi_i \ge 0$
│   │   └── func MustStationary(k *markov.Kernel, tol float64) markov.Distribution
│   │       └── $\pi P = \pi,\; \sum_i \pi_i = 1,\; \pi_i \ge 0$
│   ├── fundamental.go
│   │   ├── func FundamentalMatrix(k *markov.Kernel, tol float64) (*mat.Dense, error)
│   │   │   └── $Z = (I - P + \mathbf{1}\pi)^{-1}$
│   │   └── func AbsorbingFundamental(k *markov.Kernel, transient []int, tol float64) (*mat.Dense, error)
│   │       └── $N = (I - Q)^{-1} = \sum_{t=0}^\infty Q^t$
│   ├── mfpt.go
│   │   ├── func MFPT(k *markov.Kernel, tol float64) (*mat.Dense, error)
│   │   │   └── $M_{i,j} = \frac{Z_{j,j} - Z_{i,j}}{\pi_j} \; (i \neq j),\; M_{i,i} = 0$
│   │   ├── func MFPTToTarget(k *markov.Kernel, target int, tol float64) ([]float64, error)
│   │   │   └── $m_i = 1 + \sum_{j \neq \text{target}} P_{i,j} m_j$
│   │   └── func MFPTDelta(k *markov.Kernel, from, to, target int, eps, tol float64) (float64, error)
│   │       └── $\Delta m = m(P + \Delta P_{\text{from},\text{to}}(\varepsilon)) - m(P)$
│   ├── classes.go
│   │   ├── type StateClasses struct { SCCs [][]int; Recurrent [][]int; Transient []int; Periods map[int]int }
│   │   └── func ClassifyStates(k *markov.Kernel, tol float64) (*StateClasses, error)
│   │       └── $i \leftrightarrow j \iff (P^t)_{i,j} > 0 \land (P^s)_{j,i} > 0,\; d(i) = \gcd\{t \ge 1 : (P^t)_{i,i} > 0\}$
│   ├── entropy.go
│   │   ├── func EntropyRate(k *markov.Kernel, tol float64) (float64, error)
│   │   │   └── $H(P) = -\sum_{i,j} \pi_i P_{i,j} \log_2 P_{i,j}$
│   │   └── func KLDivergenceRate(p, q *markov.Kernel, tol float64) (float64, error)
│   │       └── $D_{\text{KL}}(P \,\|\, Q) = \sum_{i,j} \pi_i^{(P)} P_{i,j} \log_2 \frac{P_{i,j}}{Q_{i,j}}$
│   └── known_unknowns.list
│       ├── Exact function names and return structs in catrace for MFPT, perturbation ranking (soka-cli knobs), SCCs, and entropy rate
│       ├── Multi-class / reducible stationary distribution solver behavior (returning basis of stationary distributions vs. error)
│       └── Spectral gap, relaxation time, and mixing-time bound functions
├── infer/
│   ├── doc.go
│   ├── estimate.go
│   │   ├── type Estimator struct { ... }
│   │   ├── func NewEstimator(n int, names []string, prior, halflife float64) (*Estimator, error)
│   │   ├── func (e *Estimator) Observe(from, to int) error
│   │   │   └── $C_{i,j}^{(t)} = \gamma C_{i,j}^{(t-1)} + \mathbf{1}_{(i,j)},\; \gamma = 2^{-1/h}$
│   │   ├── func (e *Estimator) ObserveSequence(states []int) error
│   │   ├── func (e *Estimator) Kernel(tol float64) (*markov.Kernel, error)
│   │   │   └── $\hat{P}_{i,j} = \frac{C_{i,j} + \alpha}{\sum_r (C_{i,r} + \alpha)}$
│   │   ├── func FromTransitions(pairs []int, n int, prior float64, tol float64) (*markov.Kernel, error)
│   │   │   └── $\hat{P}_{i,j} = \frac{N_{i,j} + \alpha}{\sum_r (N_{i,r} + \alpha)}$
│   │   └── func FromSequence(states []int, n int, prior, halflife, tol float64) (*markov.Kernel, error)
│   │       └── $\hat{P}_{i,j} = \frac{\sum_t \gamma^{T-t} \mathbf{1}_{(X_t=i, X_{t+1}=j)} + \alpha}{\sum_r \left(\sum_t \gamma^{T-t} \mathbf{1}_{(X_t=i, X_{t+1}=r)} + \alpha\right)}$
│   ├── order.go
│   │   └── func MarkovOrderTest(states []int, n int) (chi2, pValue float64, err error)
│   │       └── $G^2 = 2 \sum_{i,j,k} N_{i,j,k} \ln \frac{\hat{P}_{i,j,k}}{\hat{P}_{j,k}} \sim \chi^2_{n(n-1)^2}$
│   └── known_unknowns.list
│       ├── Exact EMA half-life weighting formula used in soka-cli (--halflife)
│       ├── Handling unseen states (zero-out-degree rows) during MLE estimation without Laplace prior
│       └── Higher-order Markov chain lifting (converting k-th order chain to 1st-order product state space)
└── control/
├── doc.go
├── mdp.go
│   ├── type MDP struct { Actions []string; Transitions []*markov.Kernel; Rewards *mat.Dense; Discount float64 }
│   ├── func NewMDP(actions []string, transitions []*markov.Kernel, rewards *mat.Dense, discount, tol float64) (MDP, error)
│   │   └── $(S, A, {P^a}{a \in A}, R, \gamma),; \gamma \in [0, 1)$
│   ├── func (m *MDP) NumStates() int
│   │   └── $|S|$
│   ├── func (m *MDP) NumActions() int
│   │   └── $|A|$
│   └── func (m *MDP) InducedKernel(policy []int, tol float64) (*markov.Kernel, error)
│       └── $P^\pi{s, s'} = P^{\pi(s)}{s, s'}$
├── dp.go
│   ├── func (m *MDP) ValueIteration(tol float64) (policy []int, values []float64, err error)
│   │   └── $V{k+1}(s) = \max_{a \in A} \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V_k(s') \right]$
│   ├── func (m MDP) PolicyIteration(tol float64) (policy []int, values []float64, err error)
│   │   └── $V^{\pi_k} = (I - \gamma P^{\pi_k})^{-1} R^{\pi_k},; \pi_{k+1}(s) = \arg\max_a \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V^{\pi_k}(s') \right]$
│   ├── func ValueIteration(m MDP, tol float64) (policy []int, values []float64, err error)
│   │   └── $V^(s) = \max_{a \in A} \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V^(s') \right]$
│   └── func PolicyIteration(m MDP, tol float64) (policy []int, values []float64, err error)
│       └── $\pi^(s) = \arg\max_{a \in A} \left[ R(s,a) + \gamma \sum_{s'} P^a_{s,s'} V^{\pi^}(s') \right]$
├── bisim.go
│   ├── func (m *MDP) CoarsestPartition(tol float64) (markov.Partition, error)
│   │   └── $s \sim s' \iff \forall a \in A:; R(s,a) = R(s',a) ;\land; \forall B \in \mathcal{B},; \sum_{t \in B} P^a_{s,t} = \sum_{t \in B} P^a_{s',t}$
│   └── func MDPBisimulation(m *MDP, tol float64) (markov.Partition, error)
│       └── $s \sim s' \iff \forall a \in A:; R(s,a) = R(s',a) ;\land; \forall B \in \mathcal{B},; \sum_{t \in B} P^a_{s,t} = \sum_{t \in B} P^a_{s',t}$
└── known_unknowns.list
├── State-action dependent rewards R(s, a, s') vs. state-action rewards R(s, a) representation
├── Average-reward (undiscounted, discount = 1.0) MDP solver support via relative value iteration
└── Action-dependent state availability (when not every action is valid in every state)
```

