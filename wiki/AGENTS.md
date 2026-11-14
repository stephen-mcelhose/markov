# Wiki Schema

This wiki is maintained by an LLM following the llm-wiki pattern
(https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f).

## Domain

Markov chain mathematics, Go library design, and practitioner usage patterns
relevant to the `github.com/stephen-mcelhose/markov` library. Covers:

- Mathematical foundations: lumpability, censoring, stationary theory, MFPTs, MDPs
- Algorithmic choices and their numerical properties (GTH, Paige–Tarjan, etc.)
- Practitioner contexts: how domain users (ML engineers, researchers, data engineers) reach for these tools
- API design decisions and their rationale
- Known unknowns and open questions per subpackage

## Conventions

- **Page slugs**: `kebab-case.md`. Subdirectories by topic group: `math/`, `practice/`, `decisions/`, `api/`
- **Frontmatter**: OKF — `type`, `title`, `description`, `timestamp` (ISO-8601 UTC); optional `resource`, `tags`
- **Cross-references**: standard relative markdown links `[Title](relative/path.md)`. Never `[[wikilinks]]`
- **Sources section**: every page ends with `## Sources`

## Raw Imports

Large source ingests start as a URL in `raw/imports/urls.md` (one URL per line).
An agent reads each URL, synthesizes a wiki page, and propagates to related pages.
Raw files are immutable — never written by the LLM.

## Operations

- `ingest <url-or-file>` — synthesize a page, propagate to related pages, update index
- `ingest raw/imports/urls.md` — batch ingest all URLs in the import list
- `query <question>` — synthesize answer from wiki pages
- `lint` — audit orphans, contradictions, stale claims, missing links
