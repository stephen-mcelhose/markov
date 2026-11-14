// Package markov provides finite-state Markov chain primitives and operators for Go.
//
// It treats stochastic processes as first-class algebraic objects, combining
// exact linear-algebraic solvers with principled state-space reduction that
// preserves mathematical invariants.
//
// # Core types
//
//   - [Kernel] — a validated square row-stochastic transition matrix P ∈ ℝⁿˣⁿ
//   - [Distribution] — a validated probability vector π ∈ ℝⁿ
//   - [Partition] — a set partition 𝒫 = {B₀, …, Bₘ₋₁} of {0, …, n-1}
//
// # Resolved design choices
//
//   - Default validation tolerance is [DefaultTolerance] = 1e-9; near-zero
//     negative entries within tolerance (v ∈ [-tol, 0)) are clamped to 0.
//   - Continuous-time Markov chain generators (Q) and uniformization are
//     deferred to a dedicated ctmc subpackage rather than overloading [Kernel].
//   - Matrices are represented using dense Gonum [*gonum.org/v1/gonum/mat.Dense]
//     in v1; sparse matrix representations (CSR/DOK) are deferred to v2.
//
// # Dual ergonomics
//
// Operators are exposed both as methods on domain types (for idiomatic Go
// chaining) and as top-level functions (for algebraic readability):
//
//	// method form
//	proj, err := d.Project(partition, tol)
//
//	// top-level form
//	proj, err := markov.ProjectDistribution(d, partition, tol)
//
// # Error handling
//
// Constructors and potentially singular operations return (T, error).
// Must* helpers are provided for use when preconditions are guaranteed:
//
//	k := markov.MustNewKernel(P, names)
//
// # Subpackages
//
//   - [github.com/stephen-mcelhose/markov/reduce] — Trace (temporal censoring), Lump (spatial quotienting), and bisimulation
//   - [github.com/stephen-mcelhose/markov/solve] — Stationary, FundamentalMatrix, MFPT, ClassifyStates, EntropyRate
//   - [github.com/stephen-mcelhose/markov/infer] — MLE/Dirichlet estimation, EMA decay, MarkovOrderTest
//   - [github.com/stephen-mcelhose/markov/control] — MDPs, ValueIteration, PolicyIteration, MDPBisimulation
package markov
