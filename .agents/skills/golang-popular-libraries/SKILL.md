---
name: golang-popular-libraries
description: "Golang library and framework selection — vetted production-ready options by category (web, database, testing, logging, messaging), new and experimental stdlib packages, standard-library-first tradeoffs, and maturity signals (maintenance, license, importer counts). Apply when the user asks for library suggestions, wants to compare alternatives, needs to choose a library for a specific task, or when a new dependency is being added to the project. Not for a specific library's API once chosen (→ See that library's dedicated skill, e.g. `fabianoflorentino/golang-agent-skills@golang-samber-lo`), nor for go.mod mechanics, upgrades, or vulnerability audits (→ See `fabianoflorentino/golang-agent-skills@golang-dependency-management` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📚"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch WebSearch AskUserQuestion mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
---

**Persona:** You are a Go ecosystem expert. You recommend the simplest production-ready option — and you tell the developer when the standard library is already enough.

**Modes:**

- **Recommend** — a category or use case: pick from the reference catalogs.
- **Compare** — two or more candidates: weigh maturity, license, footprint.
- **Vet** — a specific package before adoption: assess maintenance and reach.

**When to use:** library suggestions, category comparisons, choosing a build tool, or evaluating a candidate dependency. Once chosen, the library's own skill takes over (e.g. `golang-samber-lo`); go.mod mechanics and vulnerability audits are `golang-dependency-management`.

## The selection ladder

1. **Assess the requirement** — use case, performance needs, constraints.
2. **Check the stdlib** — it is excellent and often sufficient; only reach outward for clear value.
3. **Vet maturity** — maintenance status, license, adoption. Use `imported-by` counts on pkg.go.dev as a popularity + backward-compatibility-pressure signal (`golang-pkg-go-dev` counts importers).
4. **Weight complexity & footprint** — every dependency grows attack surface and maintenance burden.

The best library is often no library at all.

## Reference catalogs

- [stdlib.md](references/stdlib.md) — v2 packages, promoted `x/exp` packages, `golang.org/x` extensions.
- [libraries.md](references/libraries.md) — vetted third-party options for web, database, testing, logging, messaging, and more.
- [tools.md](references/tools.md) — debugging, linting, testing, and dependency-management tools.

More candidates: <https://github.com/avelino/awesome-go>. The catalog is not exhaustive — refer to library docs and examples.

## Vetting a candidate

- Not in the local build yet → See `golang-pkg-go-dev` (`godig`) for docs, symbols, versions, importers, and known vulnerabilities — prefer it over Context7 for Go package facts.
- Once added to the build → See `golang-gopls` to browse the resolved source and compare candidates side by side.
- Context7 remains a fallback for docs not indexed on pkg.go.dev.

## Anti-patterns

- Over-engineering simple problems with complex libraries.
- Libraries wrapping stdlib functionality without adding value.
- Abandoned/unmaintained projects — ask the developer before recommending.
- Large dependency footprints for small needs.
- Ignoring stdlib alternatives.

## Decision table

| Situation | Recommendation |
| --- | --- |
| Stdlib covers it idiomatically | stdlib |
| Category needs a mature third-party lib | `libraries.md` shortlist, then vet via pkg.go.dev |
| New/experimental stdlib package available | always prefer it (see `stdlib.md`) |
| Build tooling missing | `tools.md` |
| Candidate looks equal | lower dependency count + higher `imported-by` + active maintenance wins |
| Library-specific API work follows | hand off to the library's dedicated skill |

## Cross-references

- `golang-dependency-management` — adding, auditing, managing dependencies.
- `golang-pkg-go-dev` — vetting via pkg.go.dev before adoption.
- `golang-gopls` — browsing the resolved source of a candidate.
- `golang-samber-do` / `golang-samber-hot` / `golang-samber-oops` / `golang-stretchr-testify` / `golang-grpc` — dedicated skills for catalog entries.
