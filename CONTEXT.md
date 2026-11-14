# Markov Chain Analysis, Reduction, Inference & Control

Finite-state discrete-time Markov chain operators, state-space reduction (temporal censoring and spatial quotienting), exact ergodic and absorbing solvers, empirical stream inference, and discounted Markov Decision Process control.

## Language

### Core Algebraic Primitives

**Kernel**:
A validated $n \times n$ right-stochastic transition matrix $P \in \mathbb{R}^{n \times n}$ ($P_{i,j} \ge 0$, $\sum_j P_{i,j} = 1$) paired with state names.
_Avoid_: TransitionTable, Graph, ChainMatrix

**Distribution**:
A validated probability vector $\pi \in \Delta^{n-1}$ ($\pi_i \ge 0$, $\sum_i \pi_i = 1$) over the $n$ states of a Markov chain.
_Avoid_: ProbSlice, WeightsVector

**Partition**:
A family of disjoint, non-empty equivalence classes (macro-blocks) $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ whose union is the full state space $\{0, \dots, n-1\}$.
_Avoid_: Clustering, StateGroups, Buckets

**IndicatorMatrix (Collector Matrix $V$)**:
The $n \times m$ binary matrix $V_{s,j} = \mathbf{1}_{B_j}(s)$ that sums micro-state transition probabilities into macro-blocks via right-multiplication ($P V$).
_Avoid_: AggregatorMatrix, MembershipMatrix

**DistributorMatrix ($U$)**:
The $m \times n$ row-stochastic matrix $U_{i,s} = w_s \mathbf{1}_{B_i}(s) / \sum_{r \in B_i} w_r$ that distributes macro-block mass across constituent micro-states and satisfies $U V = I_m$.
_Avoid_: Disaggregator, RestrictionMatrix

### Model Order Reduction

**Trace (Censored / Watched Chain)**:
The induced kernel $P_A = a + b(I - c)^{-1}d$ on a watched subset $A \subseteq S$ obtained by integrating out excursions through the transient complement $A^c$ via the Schur complement.
_Avoid_: SliceKernel, Subchain, FilteredKernel

**Lump (Kemeny–Snell Quotient)**:
The $m \times m$ quotient kernel $\hat{P} = U P V$ on the macro-blocks of a partition $\mathcal{B}$.
_Avoid_: MergeStates, CollapseKernel, Coarsen

**LumpabilityResidual**:
The maximum intra-block disagreement $R(P, \mathcal{B}) = \max_{B_i} \max_{s,s' \in B_i} \max_{B_j} |(PV)_{s,j} - (PV)_{s',j}|$; vanishes when the chain is strongly lumpable ($P V = V \hat{P}$).
_Avoid_: LumpError, ProjectionLoss

**CoarsestPartition (Bisimulation Quotient)**:
The largest (coarsest) partition $\mathcal{B}^* \preceq \mathcal{B}_0$ refining an initial partition under which a kernel (or MDP) is strongly lumpable, computed via Paige–Tarjan / Larsen–Skou block splitting.
_Avoid_: MinimalPartition,MergedBlocks

### Structural & Ergodic Solvers

**CommunicatingClass (SCC)**:
A maximal subset of states that can reach one another via positive-probability directed paths; closed communicating classes are **Recurrent**, while states that can permanently exit their class are **Transient**.
_Avoid_: Component, Cluster, NodeGroup

**StationaryDistribution**:
A left eigenvector $\pi P = \pi$ in the probability simplex $\Delta^{n-1}$, unique when the chain is unichain ($|\text{Recurrent}(P)| = 1$) and computed subtraction-free via Grassmann–Taksar–Heyman (GTH) state elimination.
_Avoid_: SteadyState, EquilibriumVector

**FundamentalMatrix ($Z$ / $N$)**:
For ergodic/unichain kernels, Kemeny & Snell's matrix $Z = (I - P + \mathbf{1}\pi)^{-1}$; for absorbing chains on transient subset $T$, the expected visit-count matrix $N = (I - Q)^{-1}$ where $Q = P_{T,T}$.
_Avoid_: GreenMatrix, Resolvent

**MeanFirstPassageTime (MFPT)**:
The expected number of steps $M_{i,j} = (Z_{j,j} - Z_{i,j})/\pi_j$ ($i \neq j$, $M_{i,i} = 0$) to reach target state $j$ for the first time starting from state $i$.
_Avoid_: HittingTime, CommuteTime

### Inference & Optimal Control

**Estimator**:
An online Dirichlet-multinomial transition counter $C^{(t)}_{i,j} = \gamma C^{(t-1)}_{i,j} + \mathbf{1}_{(i,j)}$ with prior concentration $\alpha \ge 0$ and exponential moving average half-life $h \ge 0$ ($\gamma = 2^{-1/h}$).
_Avoid_: Counter, Accumulator

**MarkovOrderTest ($G^2$)**:
The Anderson–Goodman / Billingsley log-likelihood ratio test statistic $G^2 = 2 \sum_{i,j,k} N_{i,j,k} \ln(\hat{P}_{i,j,k}/\hat{P}_{j,k}) \sim \chi^2_{n(n-1)^2}$ testing first-order vs. second-order Markov dependence.
_Avoid_: ChiSquareTest, IndependenceTest

**MDP (Markov Decision Process)**:
A controlled stochastic process $(S, A, \{P^a\}_{a \in A}, R, \gamma)$ with action transition kernels $P^a$, expected state-action rewards $R(s,a)$, and discount factor $\gamma \in [0, 1)$.
_Avoid_: Controller, Environment
