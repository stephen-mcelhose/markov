# Discipline

You are the maintainer of this markov Go library.

## API

1. Math SHOULD be exposed both as methods on domain types (for idiomatic Go chaining) and as top-level functions (for algebraic readability).
2. Constructors and operations that can fail numerically (e.g., singular matrix inversion, non-convergent solvers, invalid external inputs) MUST return `error` rather than panicking.
3. Core fallible operations SHOULD provide `Must*` counterparts (e.g., `MustNewKernel`, `MustTrace`, `MustLump`) to support clean mathematical composition when preconditions are guaranteed.
4. Top-level functions accepting pointer domain types MUST check for `nil` and return a descriptive error rather than panicking.
5. Math MUST be Mathable, don't return errors from math operations, they should be MATHable.

## Errors

1. Errors should be defined and exported.
2. Constructors and potentially singular solvers (NewKernel, Trace, Lump, Stationary) SHOULD return (T, error) and MUST NOT panic on invalid input or singular matrices.
  - Provide Must* helpers (MustNewKernel, MustTrace, MustLump) for fluent mathematical composition when inputs are known valid.
3. Math operations MUST NOT return errors; they MUST be MATHable.


## Tests

Test should be written for all public functions and methods.

1. Prefer table-driven tests.
2. Prefer in-test `type testCase{name, [setup...], inputs..., outputs..., [sideeffects...]} struct` to define test cases. Where `setup` is optional setup (the object under test), inputs are the inputs to the function, outputs are the expected outputs with names matching the function signature, and `sideeffects` are optional side effects (e.g. database updates, file writes, asserted calls to injectables etc.) that are asserted.
3. MUST use RFC2119 language for all specable behaviors.
4. MUST NOT use RFC2119 language for non-specable behaviors (e.g. examples, fuzzing, etc.)
5. MUST NOT name test functions using RFC2119 language (only testCase.name and/or t.Run(<name>))
6. Property-based tests SHOULD use pgregory.net/rapid with custom generators (e.g., row-stochastic kernels, partitions, strongly lumpable kernels) to verify algebraic invariants across randomized inputs and automatically minimize failing counterexamples, and MAY wrap generators with rapid.MakeFuzz for native go test -fuzz integration.
7. Every invariant listed in `docs/invariants/invariants.md` MUST have at least one corresponding `rapid.Check` property test. Property tests MUST be named `TestProperty_<SectionID>_<Name>` (e.g., `TestProperty_2_3_StationaryRestriction`) so invariant coverage is auditable by grep.


### Fuzzing

Fuzzing, or mathematical mutation testing, should be used to test the robustness of the library. Fuzzing should assert all invariants of the library. Use `pgregory.net/rapid` 

## Documentation

1. Each package MUST have a `doc.go` file.
2. Documentation MUST be written for all exported identifiers.
3. Markdown documentation (`.md`) MUST use GitHub-flavored LaTeX (`$...$` inline and `$$...$$` display blocks) for mathematical notation.
4. Go doc comments SHOULD use readable Unicode and indented preformatted blocks (`P_A = a + b(I - c)⁻¹d`) so `go doc` and `pkg.go.dev` render cleanly without raw LaTeX backslashes, linking to Markdown derivations where deeper LaTeX is warranted.
5. Documentation MUST otherwise follow standard Go doc comment conventions.

## Examples

1. Examples MUST be standard Go runnable examples (`example_test.go` using `func Example...()` and `// Output:` assertions) verified by `go test ./...`.
2. Examples SHOULD target the Dreyfus Stage 2 (Advanced Beginner) audience — Go engineers who write idiomatic Go and are building intuition for stochastic linear algebra — by leading each example's doc comment with a concrete systems scenario and situational maxim, and telling a cohesive story that shows the operational benefit.
3. Each package SHOULD include a package-level `func Example()` walkthrough telling an end-to-end systems story alongside symbol-level `func ExampleFunc()` / `func ExampleType_Method()` examples.
   - See [docs/personas/dreyfus-stage-2.md](docs/personas/dreyfus-stage-2.md) for the persona profile, narrative arc, and situational maxims across packages.
4. Examples MAY be deferred until a second-phase documentation pass.


## Knowledge Base (Wiki)

The repo maintains a compounding knowledge base in `docs/wiki/` following the llm-wiki pattern.
Raw import sources live in `raw/imports/urls.md` (one URL per line; `#` lines are comments).

### Structure

```
docs/wiki/
├── AGENTS.md        # wiki schema and domain definition
├── index.md         # catalog of all pages (updated on every write)
├── log.md           # append-only operation log
├── math/            # mathematical foundations and algorithm pages
├── practice/        # practitioner context and usage patterns
├── decisions/       # API and architectural decisions (ADRs)
└── api/             # surface area rationale per subpackage

raw/
└── imports/
    └── urls.md      # line-delimited URL import list
```

### Rules

1. **Raw sources are immutable.** Read `raw/imports/urls.md` and fetched URLs; never write to them.
2. **Ingest produces a wiki page.** For each source: fetch, synthesize a `docs/wiki/<group>/kebab-title.md` page with OKF frontmatter (`type`, `title`, `description`, `timestamp`, optional `resource`/`tags`), propagate insights to related pages, update `docs/wiki/index.md`, append to `docs/wiki/log.md`.
3. **Batch ingest.** When asked to ingest `raw/imports/urls.md`, process every non-comment line.
4. **Synthesize, don't transcribe.** Pages integrate and cross-reference; they do not copy sources verbatim.
5. **Index is always current.** Every wiki write MUST update `docs/wiki/index.md` before finishing.
6. **Log is append-only.** Format: `## [YYYY-MM-DD] <operation> | <detail>`. Never delete entries.
7. **Cross-references** use standard relative markdown links from the linking file. Never `[[wikilinks]]`.

### Operations

- `ingest <url>` — fetch, synthesize, propagate, update index + log
- `ingest raw/imports/urls.md` — batch ingest all URLs
- `query <question>` — synthesize answer from wiki pages; write back novel synthesis
- `lint` — audit orphans, contradictions, stale claims, missing OKF frontmatter


## Group Definition of Done

Before committing or starting the next work group, every change MUST pass the following validation pipeline in order:

1. **Surface & Persona Schema Sync (`make surface`)**:
   - If `docs/architecture.md` changed, run `make surface-emit` to regenerate `docs/surface/surface.jsonld`.
   - Verify `docs/surface/surface.jsonld` against `docs/surface/context.jsonld` and `docs/surface/surface.shacl.ttl`, and verify `docs/personas/*.md` against `docs/personas/persona.shacl.ttl`.
2. **AST & Policy Verification (`make surface-ast`)**:
   - Verify implemented Go packages match the exact file/symbol signatures in `docs/surface/surface.jsonld`, have `doc.go` and exported symbol comments without raw LaTeX backslashes, keep `func Test*` names free of RFC 2119 keywords, and include situational-maxim comments + `// Output:` blocks on all `func Example*` functions.
3. **Static Checks, Unit/Property/Example Tests & Spec Report (`make check && make spec`)**:
   - Runs `fmt-check`, `vet`, `surface`, `surface-ast`, `go test -count=1 ./...` (including `pgregory.net/rapid` property checks and `Example*` output assertions), and `./tools/testspec` to verify RFC 2119 specable sub-tests.
4. **Invariant Coverage Audit**:
   - Grep `TestProperty_` across all `*_test.go` files and cross-reference against every invariant section in `docs/invariants/invariants.md`. Every numbered invariant MUST have a matching `TestProperty_<SectionID>_<Name>` test. Missing coverage is a BLOCKING finding.
5. **Critic Math & Policy Review**:
   - Explicitly reference `docs/invariants/invariants.md` in the critic prompt.
   - Spawn a `critic` agent to math-review and policy-review all new or modified files in the completed group against `docs/architecture.md` and `AGENTS.md`. Resolve any BLOCKING findings before proceeding; document ADVISORY ones in context.

## Self-Maintenance

1. If you do something more than once, write a tool or a skill to do it for you.
2. If you expect a human will need to do something more than once, add it to the Makefile as a target.
3. Write runbooks as how-to guides.
  - A how-to guide MUST be aimed at readers assuming no prior knowledge of the topic. It MUST provide preconditions.
  - A how-to SHOULD be step by step.
  - A how-to SHOULD provide verifiable expected outcomes after each mutating step.
4. Provide a template for each documentation type you add. Consider adding SHACL or other executable schema to validate the documentation.

## Sketch

This is a sketch of the package structure. It MUST NOT be treated as gospel. It MUST be updated as we expand/refactor. It SHOULD be considered prior to writing code.

```sh
markov/
├── kernel.go        # Kernel, NewKernel, NormalizeRows, Submatrix, NearlyEqual
├── distribution.go  # Distribution validation, RestrictDistribution, ProjectDistribution
├── partition.go     # Partition, Validate, IndicatorMatrix, DistributorMatrix
├── reduce/          # Trace, IsTraceOf, Lump, IsStronglyLumpable, LumpabilityResidual
├── solve/           # Stationary (GTH/power), FundamentalMatrix, MFPT, SCCs, EntropyRate
├── infer/           # MLE/Dirichlet estimation from streams, EMA decay, order tests
└── control/         # MDPs, Value/Policy Iteration, MDP bisimulation
```


## DAG Order List

	1. Layer 1 (markov): errors.go, kernel.go, distribution.go, partition.go, plus shared rapid generators for random stochastic kernels and partitions.
	2. Structural & Ergodic Solvers (solve): classes.go \(\rightarrow \) stationary.go (GTH) \(\rightarrow \) fundamental.go \(\rightarrow \) mfpt.go \(\rightarrow \) entropy.go (needed to test stationary preservation in reduce).
	3. Model Reduction (reduce): trace.go \(\rightarrow \) lump.go \(\rightarrow \) bisim.go.
	4. Inference & Control (infer, control): estimate.go, order.go, mdp.go, dp.go, bisim.go.


## Known Unknowns Protocol
When implementing a package, read known_unknowns.list first; pick the most standard mathematical default, document the choice in doc.go, and remove the resolved item from known_unknowns.list.


