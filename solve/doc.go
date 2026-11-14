// Package solve provides exact structural and ergodic solvers for finite-state
// Markov chains.
//
// # Core operators
//
//   - [ClassifyStates] — decomposes the state space into strongly connected
//     components (SCCs), closed recurrent classes, transient states, and
//     per-state periods via Tarjan's linear-time algorithm and cycle-length GCD.
//   - [Stationary] — computes the unique stationary distribution π (πP = π) for
//     unichain Markov kernels using the Grassmann-Taksar-Heyman (GTH) direct
//     state-reduction algorithm, eliminating subtractive cancellation.
//   - [FundamentalMatrix] — computes Kemeny & Snell's ergodic fundamental matrix
//     Z = (I - P + 1π)⁻¹.
//   - [AbsorbingFundamental] — computes the absorbing fundamental matrix
//     N = (I - Q)⁻¹ over a transient subset T.
//   - [MFPT], [MFPTToTarget], [MFPTDelta] — mean first-passage times and convex
//     row-perturbation sensitivities.
//   - [EntropyRate], [KLDivergenceRate] — Shannon entropy rate H(P) and
//     Kullback-Leibler divergence rate D_KL(P ‖ Q) in bits per step.
//
// # Resolved design choices
//
//   - Reducible chains with multiple closed recurrent classes (|Recurrent| > 1)
//     do not have a unique stationary distribution; [Stationary],
//     [FundamentalMatrix], [EntropyRate], and [KLDivergenceRate] return
//     [github.com/stephen-mcelhose/markov.ErrSingularOperator] when the chain
//     is not unichain.
//   - [MFPTDelta] applies the convex row perturbation
//     P̃[from, ·] = (1 - δ)P[from, ·] + δ·e_toᵀ for δ ∈ [0, 1], returning the
//     change in mean first-passage time Δm[from, target].
//   - Spectral gap and mixing-time bounds are deferred to a dedicated spectral
//     analysis extension.
package solve
