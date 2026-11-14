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


### Fuzzing

Fuzzing, or mathematical mutation testing, should be used to test the robustness of the library. Fuzzing should assert all invariants of the library.

## Documentation

1. Each package MUST have a `doc.go` file.
2. Documentation MUST be written for all exported identifiers.
3. Markdown documentation (`.md`) MUST use GitHub-flavored LaTeX (`$...$` inline and `$$...$$` display blocks) for mathematical notation.
4. Go doc comments SHOULD use readable Unicode and indented preformatted blocks (`P_A = a + b(I - c)⁻¹d`) so `go doc` and `pkg.go.dev` render cleanly without raw LaTeX backslashes, linking to Markdown derivations where deeper LaTeX is warranted.
5. Documentation MUST otherwise follow standard Go doc comment conventions.

## Examples

1. Go examples SHOULD be written for all public functions and methods.
2. Examples SHOULD be runnable (`// Output:`) and MUST follow standard Go `testing` example naming conventions (`ExampleFunc`, `ExampleType_Method`).
3. Examples MAY be deferred until a second-phase documentation pass.


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