# Wiki Index

| Page | Summary | Tags |
| ---- | ------- | ---- |
| [Catrace API Surface Area](api/catrace-api.md) | Analysis of the original catrace API surface area and design patterns that inform the markov library. | api, go, catrace, design-patterns |
| [Lumpability Implementation and State-Space Reduction Debate](decisions/lumpability-implementation-and-state-reduction-debate.md) | Architectural debate and design of the state partitioning, lumping, and dual-surface reduction leading to the creation of the markov repository. | lumpability, partition, refinement, decision-log, catrace |
| [Origin and Design Lineage](decisions/origin-and-design-lineage.md) | Why markov exists as a separate repo from catrace, and the design decisions captured in catrace issue #59 that define its surface area. | design, lumpability, dual-surface-api, catrace, origin |
| [Absorbing Markov Chain](math/absorbing-markov-chain.md) | Properties and analysis of absorbing states, transient matrices, and fundamental matrices. | math, absorbing-states, fundamental-matrix |
| [Anderson-Goodman Statistical Inference](math/anderson-goodman-test.md) | Statistical inference about Markov chains, including likelihood-ratio tests for the Markov order. | math, statistics, hypothesis-testing, markov-order |
| [Bellman Equation](math/bellman-equation.md) | Dynamic programming optimality conditions and value/policy iteration updates in MDPs. | math, dynamic-programming, bellman-equation, mdp |
| [Bisimulation](math/bisimulation.md) | State equivalence and partition refinement concepts underlying bisimilarity in state-transition systems. | math, bisimulation, partition-refinement, equivalence |
| [Dirichlet-Multinomial Distribution](math/dirichlet-multinomial-distribution.md) | Conjugate prior probability model for estimating transition probabilities under uncertainty. | math, bayesian-inference, dirichlet-multinomial, estimator |
| [Entropy Rate](math/entropy-rate.md) | The asymptotic rate of information generation of a stationary Markov process. | math, information-theory, entropy-rate |
| [Exponential Smoothing](math/exponential-smoothing.md) | EMA count decay and recursive filtering for time-varying transition estimation. | math, filtering, exponential-smoothing, estimation |
| [Givan-Dean-Greig MDP Bisimulation](math/givan-dean-greig-mdp-homomorphism.md) | Equivalence notions and model minimization for MDPs via bisimulation (Artificial Intelligence, 2003). | math, mdp, bisimulation, minimization |
| [GTH Algorithm](math/gth-algorithm.md) | Grassmann-Taksar-Heyman (1985) numerically stable stationary distribution solver. | math, algorithms, gth, stationary-distribution |
| [Joint Kernels and Coupling](math/joint-kernels-and-coupling.md) | Product-space kernels, Kronecker factoring, and coupled multi-agent chains. | coupling, joint-kernel, stochastic |
| [Kullback-Leibler Divergence](math/kullback-leibler-divergence.md) | Relative entropy and divergence rate measures for Markov chain transitions. | math, information-theory, kl-divergence |
| [Lumpability of Markov Chains](math/lumpability-of-markov-chains.md) | Strong and weak lumpability conditions for Kemeny-Snell quotienting. | math, lumpability, state-reduction |
| [Markov Chain Foundations](math/markov-chain-foundations.md) | Row-stochastic invariants, stationary distributions, communicating classes, and ergodicity. | markov, stochastic, stationary, communicating-classes |
| [Markov Decision Process](math/markov-decision-process.md) | Stochastic control framework extending Markov chains with actions and rewards. | math, control, mdp |
| [Math Summary (catrace)](math/math-summary.md) | Mathematical summary of matrix operations, trace derivations, and steady-state properties from catrace. | mathematics, linear-algebra, summary |
| [Mean First-Passage Time](math/mean-first-passage-time.md) | Expected steps to reach a target state; linear system formulation and perturbation analysis. | math, mfpt, hitting-time |
| [Paige-Tarjan Partition Refinement](math/paige-tarjan-algorithm.md) | O(m log n) algorithm for the coarsest bisimulation partition (SIAM J. Comput., 1987). | math, algorithms, paige-tarjan, partition-refinement |
| [Partition of a Set](math/partition-of-a-set.md) | Disjoint non-empty blocks covering the state space; mathematical foundation of Lump and CoarsestPartition. | math, set-theory, partition |
| [Probability Simplex](math/probability-simplex.md) | Geometric space of probability distributions; home of the Distribution type. | math, geometry, probability, simplex |
| [Schur Complement](math/schur-complement.md) | Block-matrix factorization underlying the Trace (censoring) formula $P_A = a + b(I-c)^{-1}d$. | math, linear-algebra, schur-complement |
| [Stochastic Matrix](math/stochastic-matrix.md) | Row-stochastic, column-stochastic, and doubly-stochastic transition matrices. | math, linear-algebra, stochastic-matrix |
| [Tarjan SCC Algorithm](math/tarjan-scc-algorithm.md) | Linear-time SCC decomposition for communicating-class and recurrence detection. | math, algorithms, tarjan, scc |
| [Trace Chain and Schur Censoring](math/trace-chain.md) | Schur complement censoring of subset excursions; the temporal reduction at the heart of reduce.Trace. | trace-chain, censoring, schur-complement |
| [Traces of Consciousness (source paper)](math/traces-of-consciousness-schur-censoring.md) | Hoffman, Prakash & Chattopadhyay (2024) — mathematical origin of catrace's trace-chain construction. | math, censoring, schur-complement, trace-chain |
| [Catrace Glossary](practice/catrace-glossary.md) | Definitions of terms in Markov analysis, traces, lumping, and stochastic modeling. | glossary, terminology, markov |
| [Catrace Reference Implementation](practice/catrace-reference-implementation.md) | Go reference implementation providing trace-chain calculations; baseline for markov library. | go, catrace, reference-implementation |
| [Catrace Source Writeup](practice/source-writeup.md) | Implementation notes and architecture of the original catrace library. | source-notes, architecture, go |
