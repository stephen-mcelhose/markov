// Package infer provides statistical inference, stream estimation, and order
// testing for finite-state Markov chains.
//
// # Core operators
//
//   - [Estimator] — an online Dirichlet-multinomial transition count estimator
//     with optional exponential moving average (EMA) half-life decay.
//   - [FromTransitions] — batch maximum-likelihood / MAP kernel estimation from
//     interleaved (from, to) state pairs.
//   - [FromSequence] — batch kernel estimation from a sequential state trajectory
//     with Dirichlet prior α and EMA half-life h.
//   - [MarkovOrderTest] — Anderson-Goodman / Billingsley log-likelihood ratio
//     test (G²) for first-order vs. second-order Markov dependence.
//
// # Resolved design choices
//
//   - EMA half-life decay applies per-event across the entire count matrix C:
//     each observed transition (from, to) scales C ← γ·C + 1_{(from,to)} where
//     γ = 1 when h = 0 and γ = 2^(-1/h) when h > 0.
//   - When prior α = 0 and a state i has zero observed outgoing transitions
//     (row sum ≤ tol), [Estimator.Kernel] returns
//     [github.com/stephen-mcelhose/markov.ErrNotStochastic] rather than silently
//     fabricating a self-loop.
//   - Higher-order k-th order product-space lifting (Sᵏ) is deferred to a future
//     extension.
package infer
