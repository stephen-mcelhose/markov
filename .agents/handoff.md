# Agent Handoff — markov implementation

## Repo

`github.com/stephen-mcelhose/markov` — a first-class Go Markov chain analysis library.
Working branch: `feat/wiki-knowledge-base` (PR #1, open). Merge when you are done with wiki; then cut a new branch per work group below.

## State of play

- `AGENTS.md`, `README.md`, `Makefile`, `go.mod/sum`, `docs/invariants.md`, `.github/CODEOWNERS` — committed on `main`
- `wiki/` — 31 pages ingested (math, decisions, api, practice). **Read `wiki/decisions/surface-design.md` before writing any code.** It is the canonical API surface with every exported symbol and formula.
- `wiki/decisions/origin-and-design-lineage.md` — explains why this repo exists and all API principles.
- `wiki/decisions/lumpability-implementation-and-state-reduction-debate.md` — contains full reference implementations of `lump.go` and `lump_test.go` from catrace issue #59. **Start here for `reduce/`.**
- No `.go` source files exist yet. The repo is a clean slate.

## Reference implementations

- `catrace` repo: `gh api repos/stephen-mcelhose/catrace/contents/<file> --jq '.content' | base64 -d`
  - `kernel.go`, `trace.go`, `stationary.go`, `passage.go`, `graph.go`, `analysis.go`, `stream/estimator.go`
- `catrace issue #59`: `gh issue view 59 --repo stephen-mcelhose/catrace --comments`
  - Comment 2: `lump.go` + `lump_test.go` (methods + top-level dual surface)
  - Comment 3: `Partition` as first-class type with methods
  - Comment 4: dual functional/method API examples
- catrace patterns are reference, **not gospel**. Follow the conventions established here (see AGENTS.md).

## Process (from AGENTS.md)

For each work group below:

1. **Stubs first** — write all function signatures with `panic("unimplemented")` bodies and exported error vars. No logic yet.
2. **Tests second** — write table-driven tests per AGENTS.md §Tests before filling implementations:
   - `type testCase struct { name string; ... }` inside the test function
   - `testCase.name` MUST use RFC 2119 language for specable behaviors (`"MUST return error when kernel is nil"`)
   - Test function names must NOT use RFC 2119 language
   - Property tests via `pgregory.net/rapid` for algebraic invariants (see verification steps below)
3. **Implement** — fill in the stubs until all tests pass.
4. **Examples** — runnable `// Output:` examples for all public functions (may be deferred to documentation pass).
5. **`go test ./...`** must pass clean before committing.

### API rules (AGENTS.md §API)

- Dual surface: every operator as both a **method** on the domain type AND a **top-level function**.
- Constructors and singular solvers return `(T, error)`. Provide `Must*` helpers.
- Math operations (Add, Multiply, etc.) MUST NOT return errors — they MUST be "MATHable".
- Top-level functions accepting pointer types MUST nil-check before dereferencing.
- Errors MUST be exported `var Err... error` in `errors.go`.
- Each package MUST have a `doc.go`.
- Go doc comments: readable Unicode math in preformatted blocks, NOT raw LaTeX.

---

## Work groups

### Group 0 — Root scaffolding (prerequisite for everything)

**Files:** `errors.go`, `doc.go`, `kernel.go`, `distribution.go`, `partition.go`

These are the domain types. Everything else depends on them. Do this group first, on its own branch.

**Exported errors (errors.go):**
```go
var ErrNotStochastic     error
var ErrInvalidDistribution error
var ErrInvalidPartition  error
var ErrSingularOperator  error
```

**kernel.go surface:**
```
type Kernel struct { P *mat.Dense; StateNames []string }
NewKernel(P, names) (*Kernel, error)            // tol = 1e-9
NewKernelWithTol(P, names, tol) (*Kernel, error)
MustNewKernel(P, names) *Kernel
(k) NumStates() int
(k) Clone() *Kernel
(k) Submatrix(rows, cols []int) *mat.Dense
(k) NormalizeRows(tol) error
Submatrix(P, rows, cols) *mat.Dense             // top-level
NormalizeRows(P, tol) error                     // top-level
CanonicalSubset(subset []int) []int             // sort + dedup
NearlyEqual(a, b, tol) bool
```

**distribution.go surface:**
```
type Distribution []float64
NewDistribution(probs, tol) (Distribution, error)
MustNewDistribution(probs, tol) Distribution
(d) Validate(tol) error
(d) Restrict(subset, tol) (Distribution, error)   // π_A[i] = π[Aᵢ] / Σ_{s∈A} π[s]
(d) Project(p Partition, tol) (Distribution, error) // π̂[j] = Σ_{s∈Bⱼ} π[s]
ValidateDistribution(pi, tol) error              // top-level
RestrictDistribution(pi, subset, tol) (Distribution, error)
ProjectDistribution(pi, p, tol) (Distribution, error)
```

**partition.go surface:**
```
type Partition [][]int
(p) Validate(n) error                           // non-empty, disjoint, covering
(p) IndicatorMatrix(n) (*mat.Dense, error)      // V[s,j] = 1 iff s∈Bⱼ
(p) DistributorMatrix(n, weights Distribution, tol) (*mat.Dense, error)  // U[i,s] = w[s]/Σ_{r∈Bᵢ}w[r]
ValidatePartition(p, n) error                   // top-level
IndicatorMatrix(p, n) (*mat.Dense, error)        // top-level
DistributorMatrix(p, n, weights, tol) (*mat.Dense, error) // top-level
```

**Verification steps — Group 0:**
- `NewKernel`: rows sum to 1 ± tol, entries ≥ -tol → clamp to 0 → validate; return `ErrNotStochastic` on failure
- `Submatrix`: output[i][j] = P[rows[i]][cols[j]] exactly
- `NormalizeRows`: each row divided by its sum; error if sum ≤ tol
- `Distribution.Restrict`: slice out subset indices, divide by their sum; error if sum ≤ tol
- `Distribution.Project`: output[j] = Σ_{s∈Bⱼ} π[s]; must sum to 1.0
- `Partition.Validate`: all blocks non-empty, pairwise disjoint, union = {0..n-1}
- `Partition.IndicatorMatrix`: V is n×m; V·(all-ones m-vec) = all-ones n-vec; each row has exactly one 1
- `Partition.DistributorMatrix`: U is m×n; U·V = Iₘ (left inverse); each row sums to 1; nil weights → uniform
- Property test: for any valid `(P, partition)`, `U·V = Iₘ` holds within tol

---

### Group 1 — `reduce/` (start here, it is why this repo exists)

**Files:** `reduce/doc.go`, `reduce/trace.go`, `reduce/lump.go`, `reduce/bisim.go`

**Reference:** catrace `trace.go` (for Trace), catrace issue #59 comment 2 & 3 (for Lump), wiki `math/paige-tarjan-algorithm.md` (for CoarsestPartition).

**trace.go surface:**
```
Trace(k, subset, tol) (*Kernel, error)          // P_A = a + b(I-c)⁻¹d
MustTrace(k, subset, tol) *Kernel
IsTraceOf(k, parent, subset, tol) (bool, error) // ‖k - Trace(parent,subset)‖∞ ≤ tol
TraceReward(k, rewards, subset, tol) ([]float64, error) // r_A = r_a + b(I-c)⁻¹r_c
```

**lump.go surface:**
```
Lump(k, p, names, weights, tol) (*Kernel, error)  // P̂ = UPV
MustLump(k, p, names, weights, tol) *Kernel
IsStronglyLumpable(k, p, tol) (bool, error)        // ‖PV - VP̂‖∞ ≤ tol
LumpabilityResidual(k, p) (float64, error)         // max_{Bᵢ} max_{s,s'∈Bᵢ} max_{Bⱼ} |(PV)_{s,j}-(PV)_{s',j}|
```

**bisim.go surface:**
```
CoarsestPartition(k, initial, tol) (Partition, error)  // Paige-Tarjan partition refinement
AreBisimilar(k1, k2, tol) (bool, error)                // coarsest partition check
```

**Verification steps — Group 1:**

*Trace:*
- `P_A` is row-stochastic
- `IsTraceOf(Trace(parent, A), parent, A, tol)` = true
- stationary consistency: `π_A[i] = π[i] / Σ_{j∈A} π[j]` (use `solve.Stationary` once available; skip until Group 2 is done)
- `Trace` of singleton subset = identity (1×1 matrix with P=[[1]])
- returns `ErrSingularOperator` when (I-c) is singular (B contains recurrent states)
- Property: for any ergodic kernel and valid subset A, Trace produces a row-stochastic kernel

*Lump:*
- `P̂ = UPV` is row-stochastic
- multi-step commutation: `Pᵗ·V = V·P̂ᵗ` for t = 1,2,5,10 when strongly lumpable
- `Lump ∘ Trace = Trace ∘ Lump` commutativity when subset respects partition blocks (this is the key invariant from issue #59)
- `IsStronglyLumpable` is true iff `LumpabilityResidual ≤ tol`
- nil weights → uniform distributor U; non-nil weights → stationary-restricted distributor
- singleton partition → identity (lumped = original)
- fully merged partition (one block) → 1×1 kernel [[1]]
- Property: for any strongly lumpable `(P, B)`, `U·P·V` is invariant to choice of valid distributor U

*CoarsestPartition:*
- result is a valid partition (Validate passes)
- result coarsens initial partition (each result block is a union of initial blocks)
- Paige-Tarjan: stable under one refinement step (applying refinement again yields same partition)
- `AreBisimilar(k, k, tol)` = true always (reflexive)

---

### Group 2 — `solve/`

**Files:** `solve/doc.go`, `solve/stationary.go`, `solve/fundamental.go`, `solve/mfpt.go`, `solve/classes.go`, `solve/entropy.go`

**Reference:** catrace `stationary.go`, `passage.go`, `graph.go`, `analysis.go`. Wiki `math/gth-algorithm.md`, `math/absorbing-markov-chain.md`, `math/mean-first-passage-time.md`.

**Surface:**
```
Stationary(k, tol) (Distribution, error)              // GTH algorithm; πP = π
MustStationary(k, tol) Distribution
FundamentalMatrix(k, tol) (*mat.Dense, error)         // Z = (I - P + 1π)⁻¹
AbsorbingFundamental(k, transient, tol) (*mat.Dense, error)  // N = (I-Q)⁻¹
MFPT(k, tol) (*mat.Dense, error)                      // M[i,j] = (Z[j,j]-Z[i,j])/π[j]
MFPTToTarget(k, target, tol) ([]float64, error)       // linear system solve
MFPTDelta(k, from, to, target, eps, tol) (float64, error)  // perturbation
type StateClasses struct { SCCs, Recurrent [][]int; Transient []int; Periods map[int]int }
ClassifyStates(k, tol) (*StateClasses, error)         // Tarjan SCC + period GCD
EntropyRate(k, tol) (float64, error)                  // H = -Σᵢⱼ πᵢPᵢⱼ log₂Pᵢⱼ
KLDivergenceRate(p, q, tol) (float64, error)          // D_KL = Σᵢⱼ πᵢPᵢⱼ log(Pᵢⱼ/Qᵢⱼ)
```

**Verification steps — Group 2:**
- `Stationary`: πP = π within tol; Σπᵢ = 1; πᵢ ≥ 0; test ergodic 2×2 known-answer
- `FundamentalMatrix Z`: (I-P+1π)Z = I within tol; diagonal Z[i,i] ≥ 1/πᵢ
- `AbsorbingFundamental N`: (I-Q)N = I within tol; N[i,j] = expected visits to j before absorption starting from i
- `MFPT`: M[i,i] = 0; M[i,j] = (Z[j,j]-Z[i,j])/π[j]; M[i,j] = 1/π[j] for recurrent i=j
- `MFPTToTarget`: m[target] = 0; Σⱼ Pᵢⱼ·(1 + m[j]) = m[i] for i ≠ target
- `ClassifyStates`: SCCs partition all states; recurrent SCCs have no outgoing edges; period = GCD of cycle lengths
- `EntropyRate`: H ≥ 0; H = 0 iff deterministic; H ≤ log₂(n) for n-state chain
- `KLDivergenceRate`: D ≥ 0; D = 0 iff P = Q; returns error if Q[i,j]=0 where P[i,j]>0
- Property: `EntropyRate(k) = EntropyRate(Lump(k, partition))` only when strongly lumpable (counter-example otherwise)

---

### Group 3 — `infer/`

**Files:** `infer/doc.go`, `infer/estimate.go`, `infer/order.go`

**Reference:** catrace `stream/estimator.go`. Wiki `math/dirichlet-multinomial-distribution.md`, `math/exponential-smoothing.md`, `math/anderson-goodman-test.md`.

**Surface:**
```
type Estimator struct { ... }
NewEstimator(n, names, prior, halflife) (*Estimator, error)
(e) Observe(from, to) error                     // Cᵢⱼ ← γ·Cᵢⱼ + 1, γ = 2^(-1/h)
(e) ObserveSequence(states) error
(e) Kernel(tol) (*markov.Kernel, error)         // P̂ᵢⱼ = (Cᵢⱼ+α)/(Σᵣ(Cᵢᵣ+α))
FromTransitions(pairs, n, prior, tol) (*markov.Kernel, error)
FromSequence(states, n, prior, halflife, tol) (*markov.Kernel, error)
MarkovOrderTest(states, n) (chi2, pValue float64, err error)  // G² ~ χ²_{n(n-1)²}
```

**Verification steps — Group 3:**
- `Observe`: after k observations of (i,j), C[i][j] ≈ k (halflife=∞, prior=0)
- `Kernel`: output is row-stochastic; with large counts dominates prior; with zero counts returns prior-smoothed uniform
- EMA: after one half-life, old observations halved in weight
- `FromTransitions`: pairs is flat `[from, to, from, to, ...]` — length must be even
- `MarkovOrderTest`: G²=0 on a sequence generated from a true Markov chain (large n); G²>0 on a 2nd-order chain tested as 1st-order
- Property: `FromSequence` with halflife=∞ agrees with `FromTransitions` on same data

---

### Group 4 — `control/`

**Files:** `control/doc.go`, `control/mdp.go`, `control/dp.go`, `control/bisim.go`

**Reference:** Wiki `math/markov-decision-process.md`, `math/bellman-equation.md`, `math/givan-dean-greig-mdp-homomorphism.md`.

**Surface:**
```
type MDP struct { Actions []string; Transitions []*markov.Kernel; Rewards *mat.Dense; Discount float64 }
NewMDP(actions, transitions, rewards, discount, tol) (MDP, error)
(m) NumStates() int
(m) NumActions() int
(m) InducedKernel(policy, tol) (*markov.Kernel, error)   // P^π[s,s'] = P^{π(s)}[s,s']
(m) ValueIteration(tol) (policy []int, values []float64, err error)
(m) PolicyIteration(tol) (policy []int, values []float64, err error)
ValueIteration(m, tol) (policy, values, error)   // top-level
PolicyIteration(m, tol) (policy, values, error)  // top-level
(m) CoarsestPartition(tol) (markov.Partition, error)  // MDP bisimulation
MDPBisimulation(m, tol) (markov.Partition, error) // top-level
```

**Verification steps — Group 4:**
- `NewMDP`: all Transitions are square, same size; Rewards rows = |S|, cols = |A|; discount ∈ [0,1)
- `InducedKernel`: output[s,s'] = Transitions[policy[s]][s,s']; result is row-stochastic
- `ValueIteration`: converges (‖Vₖ₊₁ - Vₖ‖∞ ≤ tol); values satisfy Bellman optimality; policy is deterministic (argmax)
- `PolicyIteration`: same optimal policy as ValueIteration on same MDP; terminates in finite steps
- `CoarsestPartition`: s ~ s' implies R(s,a) = R(s',a) for all a AND Σ_{t∈B} P^a[s,t] = Σ_{t∈B} P^a[s',t] for all a, B
- `MDPBisimulation(m) = m.CoarsestPartition()` (dual-surface consistency)
- Property: `ValueIteration` and `PolicyIteration` produce same optimal value function on randomly generated MDPs

---

## Commit conventions

```
feat:   new user-visible functionality
fix:    bug fix affecting users
chore:  deps, configs, internal maintenance
test:   test additions/fixes
docs:   documentation only
```

Never `--no-verify`. Never `--approve` a PR (comment only).
