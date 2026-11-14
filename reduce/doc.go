// Package reduce provides state-space reduction operators for Markov chains.
//
// Two complementary operators are provided:
//
//   - [Trace] — temporal reduction (censoring): given a subset A of states,
//     returns the induced kernel P_A on A by integrating out excursions through
//     the complement B = S\A via the formula P_A = a + b(I-c)⁻¹d.
//
//   - [Lump] — spatial reduction (quotienting): given a partition 𝒫 of states,
//     returns the quotient kernel P̂ = UPV using Kemeny-Snell projection, where
//     V is the collector matrix and U is the distributor matrix.
//
//   - [CoarsestPartition] — bisimulation: computes the coarsest partition under
//     which the chain is strongly lumpable, using Paige-Tarjan partition refinement.
//
// # Resolved design choices
//
//   - Strong lumpability is verified via the exact Kemeny-Snell block diameter
//     residual [LumpabilityResidual] ≤ tol; spectral (PCCA+), Simon-Ando, and
//     Kantorovich continuous metrics are deferred to future extensions.
//   - Weak lumpability is supported in [Lump] by passing the stationary
//     distribution π as the distributor weights vector.
//
// # Dual ergonomics
//
// Every operator is a top-level function taking a [github.com/stephen-mcelhose/markov.Kernel]:
//
//	traced, err := reduce.Trace(k, subset, tol)
//	lumped, err := reduce.Lump(k, partition, nil, nil, tol)
//
// # Key invariants
//
// When strongly lumpable (LumpabilityResidual ≤ tol):
//
//	Pᵗ·V = V·P̂ᵗ   for all t ≥ 1   (Chapman-Kolmogorov commutation)
//
// Trace and Lump commute when the traced subset is a union of partition blocks:
//
//	Trace(Lump(k, 𝒫), Â, tol) ≅ Lump(Trace(k, A, tol), 𝒫_A, tol)
package reduce
