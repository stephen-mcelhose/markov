# Wiki Schema

This wiki is maintained by an LLM using the [llm-wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) skill.

## Domain

Mathematical foundations and implementation guidance for the `github.com/stephen-mcelhose/markov` Go library — a finite-state Markov chain solver, model-reduction, inference, and control library.

The wiki answers: *what formula goes here, which theorem backs it, and what numerical pitfalls must the implementation avoid?*

## Conventions

- **Page slugs**: kebab-case filenames in `docs/wiki/`. Subdirectories by topic if needed.
- **Frontmatter**: OKF — `type`, `title`, `description`, `timestamp`; optional `resource`, `tags`.
- **Cross-references**: standard relative markdown links — `[Title](page.md)`.  Never `[[wikilinks]]`.
- **Math**: GitHub-flavored LaTeX — `$...$` inline, `$$...$$` display blocks.
- **Go doc comments**: use Unicode and indented preformatted blocks; link to these wiki pages for LaTeX derivations.
- **Sources section**: every page ends with `## Sources` listing raw inputs.

## Operations

Run these via the `llm-wiki` skill:

- `ingest <source>` — read a new source, write a summary page, propagate to related pages
- `query <question>` — synthesize an answer from wiki pages, optionally write back
- `lint` — audit for orphans, contradictions, stale claims, missing links

## Page Map

| Page                            | Covers                                                        |
| ------------------------------- | ------------------------------------------------------------- |
| [references.md](references.md) | Master citation map: package → canonical textbook/paper       |
| [core.md](core.md)             | `markov/`: Kernel, Distribution, Partition                    |
| [reduce.md](reduce.md)         | `reduce/`: Trace (Schur), Lump (K–S), Bisimulation           |
| [solve.md](solve.md)           | `solve/`: Stationary (GTH), Z, MFPT, SCCs, Entropy           |
| [infer.md](infer.md)           | `infer/`: MLE, Dirichlet prior, EMA decay, Order test         |
| [control.md](control.md)       | `control/`: MDP, Value/Policy Iteration, MDP Bisimulation     |
| [../personas/dreyfus-stage-2.md](../personas/dreyfus-stage-2.md) | Dreyfus Stage 2 (Advanced Beginner) `type: persona`, situational maxims & examples roadmap |
