---
type: runbook
title: How to Inspect, Update, and Generate Documentation from the Surface Graph
description: Step-by-step how-to runbook for querying mathematical symbols, updating docs/surface/surface.jsonld, regenerating docs/architecture.md, and running SHACL + Go AST verification.
resource: docs/surface/surface.jsonld
tags: [runbook, how-to, surface-graph, shacl, docgen, validation]
timestamp: 2026-11-14T19:22:00Z
---

# How to Inspect, Update, and Generate Documentation from the Surface Graph

This runbook walks you step-by-step through using `markov`'s executable knowledge graph (`docs/surface/surface.jsonld`) and its CLI tool (`tools/surface`). You do **not** need prior knowledge of JSON-LD, RDF, or W3C SHACL to follow this guide.

---

## 1. When to Use This Runbook

Use this runbook whenever you want to:

1. **Look up the math behind a function** (its formal definition, preconditions, error contract, algebraic invariants, and textbook citation) directly from your terminal.
2. **Regenerate `docs/architecture.md`** from `docs/surface/surface.jsonld` so the Mermaid dependency diagram, GitHub LaTeX tables, and ASCII symbol tree stay in lockstep.
3. **Add or modify an exported function, method, type, or invariant** and verify that the JSON-LD graph, SHACL shapes, Go AST signatures, and tests all agree.

---

## 2. Preconditions

Before starting, verify that your local environment meets these preconditions:

1. **Working Directory**: You are at the repository root (`/Users/stephen.mcelhose.ext/repos/markov`).
   ```sh
   pwd
   ```
   *Expected outcome*: Prints a path ending in `/markov`.

2. **Go Toolchain (`go >= 1.22`)**:
   ```sh
   go version
   ```
   *Expected outcome*: Prints `go version go1.27...` (or `>= 1.22`).

3. **Canonical Files Present**:
   ```sh
   ls docs/surface/surface.jsonld docs/surface/surface.shacl.ttl docs/architecture.md
   ```
   *Expected outcome*: Lists all three files with exit code `0`.

---

## 3. Step-by-Step Procedure

### Step 1: Query a Mathematical Symbol in the Terminal (`inspect`)

To inspect the mathematical specification and reverse-graph relationships of any symbol (or package/file substring) without opening a 2,100-line JSON file, run the `inspect` subcommand:

```sh
go run ./tools/surface inspect MFPTDelta
```

**Verifiable Expected Outcome**:
Your terminal prints the structured specification card for `solve.MFPTDelta`:

```text
─── urn:markov:sym:solve.MFPTDelta (Function) ───
  File:        solve/mfpt.go
  Signature:   func MFPTDelta(k *markov.Kernel, from, to, target int, delta, tol float64) (float64, error)
  Definition:  $\Delta m_{\text{from},\text{target}} = m_{\text{from},\text{target}}(\tilde{P}) - m_{\text{from},\text{target}}(P),\quad \tilde{P}_{\text{from},\cdot} = (1-\delta)P_{\text{from},\cdot} + \delta e_{\text{to}}^\top,\; \delta \in [0, 1]$
  Requires:    $\text{from}, \text{to}, \text{target} \in \{0,\dots,n-1\},\; \delta \in [0, 1]$, perturbed row remains stochastic
  Errors:      Returns `ErrNotStochastic` or `ErrSingularOperator` if perturbation is invalid or target unreachable
  Sources:
    • stephen-mcelhose/soka-cli (`soka suggest` row perturbation sensitivity) (https://github.com/stephen-mcelhose/soka-cli)
  Invariants:
    ⊢ Zero-Perturbation Identity: $\delta = 0 \implies \Delta m_{\text{from},\text{target}} = 0$
    ⊢ Direct Target Shortcut Monotonicity: $\text{to} = \text{target} \land \delta > 0 \implies \Delta m_{\text{from},\text{target}} \le 0$
```

You can also query an entire file, operator, or sentinel error (for example, `go run ./tools/surface inspect reduce.Lump` or `go run ./tools/surface inspect ErrSingularOperator`), or run `go run ./tools/surface --help` to list all Cobra subcommands (`validate`, `ast`, `inspect`, `render`, `emit`).

---

### Step 2: Generate `docs/architecture.md` from `docs/surface/surface.jsonld` (`make docs-gen`)

`docs/surface/surface.jsonld` is the single source of truth for the surface graph. Whenever `surface.jsonld` is updated, regenerate `docs/architecture.md` (which renders the Mermaid dependency diagram, per-package LaTeX reference tables, cross-package relational indices, and the ASCII tree):

```sh
make docs-gen
```

**Verifiable Expected Outcome**:
The command runs `go run ./tools/surface render` and prints:

```text
PASS  rendered docs/architecture.md from docs/surface/surface.jsonld
PASS  surface graph valid (5 packages, 22 files, 74 symbols, 80 invariants, 56 sources)
PASS  persona docs valid (1 persona(s) verified against persona.shacl.ttl)
PASS  runbook docs valid (1 runbook(s) verified against runbook.shacl.ttl)
```

---

### Step 3: Validate SHACL Shapes and Documentation Sync (`make surface`)

Run the read-only surface validator to confirm that:
1. `docs/surface/surface.jsonld` satisfies every SHACL constraint in `docs/surface/surface.shacl.ttl`,
2. `docs/architecture.md` is 100% in sync with `docs/surface/surface.jsonld`, and
3. `docs/personas/*.md` and `docs/runbooks/*.md` satisfy their SHACL schemas.

```sh
make surface
```

**Verifiable Expected Outcome**:
Exits with code `0` and prints `PASS` lines for `surface graph valid`, `persona docs valid`, and `runbook docs valid`.

---

### Step 4: Verify Live Go Code Against the Surface Graph (`make surface-ast`)

Run the AST and policy verifier to confirm that every Go file on disk matches the exact function/method/type signatures in `docs/surface/surface.jsonld` and obeys `AGENTS.md` documentation and example rules:

```sh
make surface-ast
```

**Verifiable Expected Outcome**:
Exits with code `0` and prints:

```text
PASS  AST & policy verified across 5 implemented package(s)
```

---

## 4. Verification

To run the complete end-to-end project verification suite (formatting, `go vet`, SHACL surface validation, Go AST verification, unit/property/example tests, and the RFC 2119 specification report):

```sh
make check && make spec
```

**Verifiable Expected Outcome**:
- All 7 packages pass `go test`.
- `make spec` ends with `PASS  7 package(s)  202 specable` (or more as new specs are added) and exit code `0`.

---

## 5. Troubleshooting & Rollback

| Symptom / Error Message | Cause | Fix / Rollback |
| :--- | :--- | :--- |
| `FAIL  docs/architecture.md is out of sync with docs/surface/surface.jsonld` | Someone edited `docs/surface/surface.jsonld` (or `docs/architecture.md` directly) without running `make docs-gen`. | Run `make docs-gen` (`go run ./tools/surface render`) to regenerate `docs/architecture.md` from `docs/surface/surface.jsonld`. (If you intentionally edited the ASCII tree in `docs/architecture.md` instead, run `make surface-emit` (`go run ./tools/surface emit`) to import it into `surface.jsonld`, then `make docs-gen`.) |
| `[FunctionShape] urn:markov:sym:... MUST have a mathematical definition (def) containing LaTeX $...$` | A new function node in `surface.jsonld` is missing `"def"` or its formula is not wrapped in `$...$`. | Wrap the mathematical definition in `$...$` inside `docs/surface/surface.jsonld` and re-run `make docs-gen`. |
| `[pkg/file.go] symbol X signature mismatch` | The Go signature in `.go` code differs from `"signature"` in `docs/surface/surface.jsonld`. | Update either the Go function signature or `"signature"` in `docs/surface/surface.jsonld` so they match, then run `make docs-gen && make surface-ast`. |
| **Rollback to last known good state** | An edit left `surface.jsonld` or `architecture.md` malformed. | Run `git checkout -- docs/surface/surface.jsonld docs/architecture.md` and verify with `make surface`. |

---

## 6. Runbook Template

When adding a new `type: runbook` guide under `docs/runbooks/`, use the following template (enforced by `docs/runbooks/runbook.shacl.ttl`):

```markdown
---
type: runbook
title: <How to ...>
description: <One-sentence summary of what the reader will accomplish>
resource: <Primary file or command covered>
tags: [runbook, how-to, <tag>]
timestamp: <ISO-8601 UTC>
---

# <How to ...>

## 1. When to Use This Runbook
## 2. Preconditions
## 3. Step-by-Step Procedure
## 4. Verification
## 5. Troubleshooting & Rollback
## 6. Runbook Template
```
