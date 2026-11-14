// Package control provides Markov Decision Processes (MDPs), exact dynamic
// programming solvers, and reward-aware bisimulation minimization.
//
// # Core types and operators
//
//   - [MDP] — a finite-state, finite-action discounted Markov Decision Process
//     (S, A, {Pᵃ}_{a∈A}, R, γ) with γ ∈ [0, 1).
//   - [MDP.InducedKernel] — extracts the Markov transition kernel P^π induced
//     by a deterministic stationary policy π: S → A.
//   - [ValueIteration] / [MDP.ValueIteration] — computes the optimal policy π*
//     and value vector V* via Bellman optimality fixed-point iteration.
//   - [PolicyIteration] / [MDP.PolicyIteration] — computes the optimal policy π*
//     and value vector V* via Howard's exact policy evaluation + improvement loop.
//   - [MDPBisimulation] / [MDP.CoarsestPartition] — computes the Givan-Dean-Greig
//     coarsest bisimulation partition refining reward equivalence across all
//     action transition kernels.
//
// # Resolved design choices
//
//   - Rewards are represented as an |S|×|A| matrix R(s, a) of expected immediate
//     state-action rewards; transition-dependent rewards R(s, a, s') can be
//     pre-integrated via R(s, a) = Σ_{s'} Pᵃ[s, s'] R(s, a, s').
//   - Discount factor γ is strictly in [0, 1) so the Bellman operator is a
//     γ-contraction in ℓ∞ and (I - γP^π) is strictly diagonally dominant;
//     undiscounted average-reward MDPs (γ = 1) are deferred to a future extension.
//   - Every action a ∈ A is available in every state s ∈ S.
package control
