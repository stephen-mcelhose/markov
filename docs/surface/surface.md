# Surface Knowledge Graph & SHACL Schema

This directory contains the machine-walkable JSON-LD surface graph for `markov` and its executable SHACL validation schema:

* `context.jsonld` — JSON-LD 1.1 `@context` mapping the top-level package keys (`layer1`, `reduce`, `solve`, `infer`, `control`), file keys (`kernel`, `distribution`, `partition`, `trace`, `lump`, `bisim`, etc.), and symbol predicates (`def`, `req`, `err`, `src`, `invariants`, `delegatesTo`, `references`) to the `https://github.com/stephen-mcelhose/markov/vocab#` RDF vocabulary.
* `surface.shacl.ttl` — W3C SHACL (`sh:NodeShape` / `sh:PropertyShape`) constraints enforcing IRI naming patterns, cardinality (`sh:minCount`, `sh:maxCount`), LaTeX math delimiters on `def` and `expr`, valid URLs on `src`, and referential class integrity across all graph nodes.
* `surface.jsonld` — The canonical JSON-LD knowledge graph generated from `docs/architecture.md`.

## Node Template

When adding a new exported symbol to `docs/architecture.md` and `docs/surface/surface.jsonld`, adhere to the following predicate template:

### Markdown Tree Template (`docs/architecture.md`)

```text
├── <file>.go
│   ├── func <Name>(<params>) (<returns>)
│   │   ├── def: $<formal mathematical definition>$
│   │   ├── req: $<domain preconditions>$
│   │   ├── err: Returns `<SentinelError>` when <boundary condition>
│   │   ├── src: [<Author (Year)>, *<Title>*, <Section>](<https://open-access-url>)
│   │   └── ⊢ <Invariant Name>: $<algebraic or probabilistic invariant>$
```

### JSON-LD Node Template (`docs/surface/surface.jsonld`)

```json
{
  "@id": "urn:markov:sym:<pkg>.<Name>",
  "@type": "Function",
  "name": "<Name>",
  "signature": "func <Name>(<params>) (<returns>)",
  "def": "$<formal mathematical definition>$",
  "req": "$<domain preconditions>$",
  "err": "Returns `<SentinelError>` when <boundary condition>",
  "references": [
    "urn:markov:sym:markov.Kernel"
  ],
  "src": [
    {
      "@type": "Source",
      "citation": "<Author (Year)>, *<Title>*, <Section>",
      "url": "https://open-access-url"
    }
  ],
  "invariants": [
    {
      "@id": "urn:markov:inv:<pkg>.<Name>.<Invariant_Slug>",
      "@type": "Invariant",
      "name": "<Invariant Name>",
      "expr": "$<algebraic or probabilistic invariant>$"
    }
  ]
}
```

## Verification

* Validate the surface graph against `context.jsonld`, `surface.shacl.ttl`, Go signature syntax, and `docs/architecture.md`:
  ```sh
  make surface
  ```
* Regenerate `docs/surface/surface.jsonld` after editing `docs/architecture.md`:
  ```sh
  make surface-emit
  ```
