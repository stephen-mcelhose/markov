# markov

A finite-state Markov chain solver, model-reduction, inference, and control library for Go, backed by [Gonum](https://gonum.org).

`markov` treats stochastic processes as first-class algebraic objects—combining exact linear-algebraic solvers with principled state-space reduction operators that preserve mathematical invariants.

## Why `markov`?

Most Markov packages in Go are n-gram text generators (`map[string][]string`), leaving finite-state linear-algebraic Markov analysis without an idiomatic Go home. `markov` fills that gap with a light dependency footprint (Go standard library + `gonum`) and three core commitments:

* **Dual Ergonomics:** Mathematical operators are exposed both as **methods on domain types** (`d.Project(...)`, `p.IndicatorMatrix(...)`, `m.ValueIteration(...)`) for idiomatic Go chaining and as **top-level functions** (`reduce.Trace(k, ...)`, `reduce.Lump(k, ...)`) for algebraic readability.
* **Principled, Auditable Compression:** Reduction operators either guarantee exact behavioral preservation within tolerance `tol` (Schur complement censoring and Kemeny–Snell quotienting) or expose explicit residual and information-loss metrics.
* **Executable Mathematical Specification:** Every exported type, operator, domain precondition, literature citation, and algebraic invariant is encoded in a machine-walkable JSON-LD knowledge graph ([docs/surface/surface.jsonld](./docs/surface/surface.jsonld)) validated against W3C SHACL shapes and checked against the Go AST at build time.

## Package Layout

```text
markov/
├── kernel.go        # Kernel, NewKernel, NormalizeRows, Submatrix, NearlyEqual
├── distribution.go  # Distribution validation, RestrictDistribution, ProjectDistribution
├── partition.go     # Partition, Validate, IndicatorMatrix, DistributorMatrix
├── reduce/          # Temporal & spatial reduction: Trace, IsTraceOf, Lump, IsStronglyLumpable, LumpabilityResidual
├── solve/           # Dynamics & decomposition: Stationary (GTH), FundamentalMatrix, MFPT, ClassifyStates, EntropyRate
├── infer/           # Stream estimation: MLE/Dirichlet counts, EMA decay, MarkovOrderTest (G²)
├── control/         # Controlled chains: MDPs, Value/Policy Iteration, MDPBisimulation
├── docs/            # Architecture knowledge graph, JSON-LD/SHACL surface schema, mathematical wiki, and personas
└── tools/           # Executable SHACL + Go AST surface validator (tools/surface) & RFC 2119 spec reporter (tools/testspec)
```

## Mathematical Foundations

* **Temporal Censoring (`reduce.Trace`):** Projects a transition matrix $P$ onto a watched subset $A$ by folding away intermediate excursions through $A^c$ via the Schur complement:
  $$P_A = a + b(I - c)^{-1}d$$
* **Spatial Quotienting (`reduce.Lump`):** Merges disjoint equivalence classes of a partition $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ via Kemeny–Snell projection:
  $$\hat{P} = U P V$$
  where $V$ is the $n \times m$ partition indicator matrix and $U$ is the $m \times n$ row-stochastic distributor matrix ($U V = I_m$). Strong lumpability holds within tolerance $\varepsilon$ when $R(P, \mathcal{B}) = \max_{B_i} \max_{s,s' \in B_i} \max_{B_j} |(PV)_{s,j} - (PV)_{s',j}| \le \varepsilon$ (with exact commutation $P V = V \hat{P}$ at $\varepsilon = 0$).
* **Subtraction-Free Ergodic Solvers (`solve.Stationary`, `solve.FundamentalMatrix`, `solve.MFPT`):** Computes steady-state distributions $\pi P = \pi$ via Grassmann–Taksar–Heyman (GTH) direct state reduction—eliminating catastrophic cancellation on nearly-disconnected chains—and derives the Kemeny–Snell fundamental matrix $Z = (I - P + \mathbf{1}\pi)^{-1}$ and mean first-passage times $M_{i,j} = (Z_{j,j} - Z_{i,j})/\pi_j$.

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/reduce"
	"gonum.org/v1/gonum/mat"
)

func main() {
	P := mat.NewDense(4, 4, []float64{
		0.2, 0.3, 0.4, 0.1,
		0.1, 0.4, 0.2, 0.3,
		0.3, 0.0, 0.5, 0.2,
		0.1, 0.2, 0.6, 0.1,
	})

	k, err := markov.NewKernelWithTol(P, []string{"a0", "a1", "b0", "b1"}, 1e-9)
	if err != nil {
		log.Fatal(err)
	}

	// Spatial quotienting via Kemeny–Snell projection: P_hat = U * P * V
	partition := markov.Partition{{0, 1}, {2, 3}}
	ok, err := reduce.IsStronglyLumpable(k, partition, 1e-9)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Strongly lumpable:", ok)

	lumped, err := reduce.Lump(k, partition, []string{"A", "B"}, nil, 1e-9)
	if err != nil {
		log.Fatal(err)
	}

	// Temporal censoring via Schur complement trace: P_A = a + b(I - c)^(-1)d
	traced, err := reduce.Trace(lumped, []int{0}, 1e-9)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Reduced kernel:\n%v\n", mat.Formatted(traced.P))
}
```

## Executable Knowledge Graph, SHACL & AST Constraints

Rather than letting mathematical documentation, API signatures, and property tests drift apart, `markov` binds them together in an executable verification loop:

1. **Architecture Knowledge Graph ([docs/architecture.md](./docs/architecture.md) $\to$ [docs/surface/surface.jsonld](./docs/surface/surface.jsonld)):**
   Every exported symbol across all 5 packages is modeled as an IRI-addressed node (`urn:markov:sym:<pkg>.<Symbol>`) carrying typed predicates:
   * `def` — Formal LaTeX mathematical definition ($P_A = a + b(I-c)^{-1}d$, $\hat{P} = UPV$, $Z = (I - P + \mathbf{1}\pi)^{-1}$).
   * `req` — Domain preconditions required for well-posedness (e.g., spectral radius $\rho(Q) < 1$, unichain $|\text{Recurrent}(P)| = 1$).
   * `err` — Sentinel error contract when preconditions or numerical conditioning fail (`ErrNotStochastic`, `ErrSingularOperator`, `ErrInvalidDistribution`, `ErrInvalidPartition`).
   * `src` — Primary literature citation (Kemeny & Snell 1976, Grassmann–Taksar–Heyman 1985, Larsen & Skou 1991, Buchholz 1994, Puterman 1994, Givan–Dean–Greig 2003).
   * `invariants` (`⊢`) — Tolerance-bounded algebraic and probabilistic theorems (`urn:markov:inv:...`) verified by `pgregory.net/rapid` property tests.

2. **SHACL Shape Validation (`make surface`):**
   `tools/surface` validates `docs/surface/surface.jsonld` against [docs/surface/context.jsonld](./docs/surface/context.jsonld) and [docs/surface/surface.shacl.ttl](./docs/surface/surface.shacl.ttl) (`sh:NodeShape` / `sh:PropertyShape`), enforcing IRI patterns, LaTeX math delimiters on every definition and invariant, valid Go declaration syntax, referential integrity across `receiverType`/`delegatesTo`/`references` edges, package DAG acyclicity, and [docs/personas/persona.shacl.ttl](./docs/personas/persona.shacl.ttl) constraints on pedagogical documentation.

3. **Go AST & Policy Verification (`make surface-ast`):**
   `go run ./tools/surface ast` parses the live Go source tree via `go/ast` and verifies:
   * **Exact Signature Parity:** Every symbol in `surface.jsonld` exists in its declared `.go` file with the exact normalized Go type signature.
   * **Clean Godoc Math:** Every package has a `doc.go` and every exported identifier has a Go doc comment using readable Unicode math (`P_A = a + b(I - c)⁻¹d`) with zero raw LaTeX backslashes, linking to [docs/wiki/](./docs/wiki/index.md) for full derivations.
   * **Dreyfus Stage 2 Runnable Examples:** Every `func Example*()` in `example_test.go` leads with a situational-maxim doc comment (see [docs/personas/dreyfus-stage-2.md](./docs/personas/dreyfus-stage-2.md)) and ends with a verified `// Output:` block.

4. **Executable RFC 2119 Specification (`make spec`):**
   Table-driven sub-tests use RFC 2119 keywords (`MUST`, `MUST NOT`, `SHOULD`) and are rendered into a live behavioral specification report via `tools/testspec`, paired with randomized `pgregory.net/rapid` generators asserting the `⊢` invariants in [docs/invariants/invariants.md](./docs/invariants/invariants.md).

```sh
make check       # Runs fmt-check, vet, surface (SHACL), surface-ast (Go AST), and unit/property/example tests
make spec        # Runs all tests and renders the RFC 2119 specification report
make surface-emit # Regenerates docs/surface/surface.jsonld from docs/architecture.md and validates
```

## Documentation & Mathematical Wiki

* **[docs/architecture.md](./docs/architecture.md)** — Full package/file/symbol knowledge graph with `def:`, `req:`, `err:`, `src:`, and `⊢` invariants.
* **[docs/invariants/invariants.md](./docs/invariants/invariants.md)** — Normative algebraic and probabilistic invariants across packages.
* **[docs/wiki/index.md](./docs/wiki/index.md)** — Mathematical derivations, numerical analysis notes (e.g., why GTH beats LU on nearly-reducible chains), and master bibliography ([docs/wiki/references.md](./docs/wiki/references.md)).
* **[docs/personas/dreyfus-stage-2.md](./docs/personas/dreyfus-stage-2.md)** — Target persona, vocabulary bridge, and situational maxims for runnable Go examples.
* **[AGENTS.md](./AGENTS.md)** — Maintainer discipline, API dual-surface rules, and 4-step verification pipeline.

## License

MIT © Stephen McElhose
