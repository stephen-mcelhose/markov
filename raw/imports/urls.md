# Raw Import URLs
# One URL per line. Lines starting with # are comments.
# Add URLs here; run `ingest raw/imports/urls.md` to batch-synthesize into wiki/.

# =============================================================================
# Reference implementation — catrace
# =============================================================================

# catrace repo — the reference implementation markov extracts and extends
https://github.com/stephen-mcelhose/catrace

# Issue #59 — the design event that spawned markov. Contains full implementations
# of lump.go, partition.go (methods + top-level dual surface), lump_test.go,
# and the PROS/CONS debate that resolved to creating this repo instead.
https://github.com/stephen-mcelhose/catrace/issues/59

# Source paper for catrace's trace-chain construction (Schur complement censoring)
# Hoffman, Prakash & Chattopadhyay — "Traces of Consciousness", Preprints 2024 (CC BY 4.0)
https://www.preprints.org/manuscript/202410.1305/v1

# catrace wiki — synthesized knowledge from the reference implementation
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/wiki/markov-chain-foundations.md
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/wiki/trace-chain.md
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/wiki/joint-kernels-and-coupling.md
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/wiki/catrace-api.md
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/wiki/catrace-glossary.md
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/source_writeup.md
https://raw.githubusercontent.com/stephen-mcelhose/catrace/main/docs/math_summary.md

# =============================================================================
# Root package — Kernel, Distribution, Partition
# =============================================================================

# Stochastic matrix (Kernel)
https://en.wikipedia.org/wiki/Stochastic_matrix

# Probability simplex (Distribution)
https://en.wikipedia.org/wiki/Simplex

# Partition of a set (Partition)
https://en.wikipedia.org/wiki/Partition_of_a_set

# =============================================================================
# reduce/ — Trace, Lump, Bisimulation
# =============================================================================

# Schur complement — the linear algebra behind reduce.Trace
https://en.wikipedia.org/wiki/Schur_complement

# Lumpability — Kemeny-Snell strong lumpability, reduce.Lump
https://en.wikipedia.org/wiki/Lumpability

# Bisimulation — reduce.CoarsestPartition, reduce.AreBisimilar
https://en.wikipedia.org/wiki/Bisimulation

# Paige & Tarjan (1987) — "Three Partition Refinement Algorithms", SIAM J. Comput.
# Partition refinement algorithm underlying reduce.CoarsestPartition
https://doi.org/10.1137/0216062

# =============================================================================
# solve/ — Stationary, FundamentalMatrix, MFPT, ClassifyStates, EntropyRate
# =============================================================================

# GTH algorithm — Grassmann, Taksar & Heyman (1985), Operations Research 33(5)
# Numerically stable stationary distribution solver used in solve.Stationary
https://doi.org/10.1287/opre.33.5.1107

# Absorbing Markov chain — covers both fundamental matrices N=(I-Q)^{-1} and Z=(I-P+1π)^{-1}
https://en.wikipedia.org/wiki/Absorbing_Markov_chain

# Mean first-passage time — solve.MFPT, solve.MFPTToTarget
https://en.wikipedia.org/wiki/Mean_first-passage_time

# Tarjan's SCC algorithm — solve.ClassifyStates
https://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm

# Entropy rate — solve.EntropyRate
https://en.wikipedia.org/wiki/Entropy_rate

# Kullback-Leibler divergence — solve.KLDivergenceRate
https://en.wikipedia.org/wiki/Kullback%E2%80%93Leibler_divergence

# =============================================================================
# infer/ — Estimator, MarkovOrderTest
# =============================================================================

# Dirichlet-multinomial distribution — MLE/Dirichlet estimation in infer.Estimator
https://en.wikipedia.org/wiki/Dirichlet-multinomial_distribution

# Exponential smoothing — EMA count decay in infer.Estimator
https://en.wikipedia.org/wiki/Exponential_smoothing

# Anderson & Goodman (1957) — "Statistical Inference about Markov Chains", JASA
# The G² likelihood-ratio Markov order test in infer.MarkovOrderTest
https://www.jstor.org/stable/2237025

# =============================================================================
# control/ — MDP, ValueIteration, PolicyIteration, Bisimulation
# =============================================================================

# Markov decision process
https://en.wikipedia.org/wiki/Markov_decision_process

# Bellman equation — value iteration
https://en.wikipedia.org/wiki/Bellman_equation

# Givan, Dean & Greig (2003) — "Equivalence Notions and Model Minimization in MDPs"
# Artificial Intelligence 147(1-2): 163-223 — control.CoarsestPartition (MDP bisimulation)
https://doi.org/10.1016/S0004-3702(03)00047-1
