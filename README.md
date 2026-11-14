# markov

A finite-state Markov chain solver, model-reduction, inference, and control library for Go, backed by [Gonum](https://gonum.org).

`markov` treats stochastic processes as first-class algebraic objects—combining exact linear-algebraic solvers with principled state-space reduction operators that preserve mathematical invariants.

> [!WARNING]
> **Charter vs. Reality:** This README serves as the architectural charter and target specification for `markov`. The repository is currently in initial bootstrap (`AGENTS.md` and package layout defined); packages and APIs below are actively being ported from [`catrace`](https://github.com/stephen-mcelhose/catrace) and implemented.

## Why `markov`?

Most Markov packages in Go are n-gram text generators (`map[string][]string`), leaving finite-state linear-algebraic Markov analysis without an idiomatic Go home. `markov` fills that gap with a light dependency footprint (Go standard library + `gonum`) and two core commitments:

* **Dual Ergonomics:** Mathematical operators are exposed both as **methods on domain types** (`d.Project(...)`, `p.IndicatorMatrix(...)`, `m.ValueIteration(...)`) for idiomatic Go chaining and as **top-level functions** (`reduce.Trace(k, ...)`, `reduce.Lump(k, ...)`) for algebraic readability.
* **Principled, Auditable Compression:** Reduction operators either guarantee exact behavioral preservation within tolerance `tol` (Schur complement censoring and Kemeny–Snell quotienting) or expose explicit residual and information-loss metrics.

## Package Layout

```text
markov/
├── kernel.go        # Kernel, NewKernel, NormalizeRows, Submatrix, NearlyEqual
├── distribution.go  # Distribution validation, RestrictDistribution, ProjectDistribution
├── partition.go     # Partition, Validate, IndicatorMatrix, DistributorMatrix
├── reduce/          # Temporal & spatial reduction: Trace, IsTraceOf, Lump, IsStronglyLumpable, LumpabilityResidual
├── solve/           # Dynamics & decomposition: Stationary (GTH/power), FundamentalMatrix, MFPT, SCCs, EntropyRate
├── infer/           # Stream estimation: MLE/Dirichlet counts, EMA decay, Markov order tests
└── control/         # Controlled chains: MDPs, Value/Policy Iteration, MDP bisimulation
```

## Mathematical Foundations

* **Temporal Censoring (`reduce.Trace`):** Projects a transition matrix $P$ onto a watched subset $A$ by folding away intermediate excursions through $A^c$ via the Schur complement:
  $$P_A = a + b(I - c)^{-1}d$$
* **Spatial Quotienting (`reduce.Lump`):** Merges disjoint equivalence classes of a partition $\mathcal{B} = \{B_0, \dots, B_{m-1}\}$ via Kemeny–Snell projection:
  $$\hat{P} = U P V$$
  where $V$ is the $n \times m$ partition indicator matrix and $U$ is the $m \times n$ row-stochastic distributor matrix ($U V = I_m$). Strong lumpability holds within tolerance $\varepsilon$ when $\|P V - V \hat{P}\|_\infty \le \varepsilon$.

## Target Usage

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

	k, err := markov.NewKernel(P, []string{"a0", "a1", "b0", "b1"}, 1e-9)
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

## Development & Conventions

Engineering discipline, RFC 2119 testing specs (`pgregory.net/rapid` property tests + table-driven tests), and documentation standards are governed by [AGENTS.md](./AGENTS.md). See [docs/invariants/invariants.md](./docs/invariants/invariants.md) for the mathematical invariants enforced across packages.

## License

MIT © Stephen McElhose
