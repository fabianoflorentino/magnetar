---
name: golang-gopls
description: "Golang semantic code intelligence via `gopls`, the official Go language server — go-to-definition, find references, call/implementation hierarchy, workspace symbol search, package API discovery, diagnostics, safe rename, refactors (extract/inline/fill/rewrite code actions), formatting, and generated tests. Reaches an agent via gopls's own MCP server (`go_*` tools), Claude Code's native `LSP` tool, or the `gopls` CLI. Use when navigating or refactoring Go code — jumping to a definition, finding call sites before a rename, understanding a file's or package's dependencies, running diagnostics after an edit, or extracting/inlining/renaming. Not for the published ecosystem — packages not in your `go.mod`, versions, licenses, importers — → See `fabianoflorentino/golang-agent-skills@golang-pkg-go-dev` skill (`godig`). Not for a whole-tree vulnerability audit → See `fabianoflorentino/golang-agent-skills@golang-security` skill (`govulncheck`)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness. Requires the gopls binary (go install golang.org/x/tools/gopls@latest) v0.20+ on PATH.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🛰️"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - gopls
    install:
      - kind: go
        package: golang.org/x/tools/gopls@latest
        bins: [gopls]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who reaches for semantic code intelligence instead of grep whenever the question is about the resolved build — grep finds text, `gopls` finds meaning (types, call graphs, shadowing, implementation relationships).

**Dependencies:** `gopls` — `go install golang.org/x/tools/gopls@latest` (v0.20+). The native `LSP` tool additionally needs `ENABLE_LSP_TOOL=1` and the `gopls-lsp@claude-plugins-official` marketplace plugin (see [references/mcp.md](references/mcp.md)).

**When to use:** any question about your locally resolved build — a symbol's definition, call sites, package shape, diagnostics, safe rename. `gopls` only answers questions about **your specific, locally resolved build**: the workspace plus every dependency exactly as pinned in go.sum, including `replace` directives. For anything outside that build (versions, docs, licenses, CVEs of a package you haven't added), → See `golang-pkg-go-dev` (`godig`) instead.

## Three ways to reach gopls

Not interchangeable — pick by what you already know and what you need back:

| Way | Setup | Best when | Positions |
| --- | --- | --- | --- |
| **MCP server (preferred)** | `claude mcp add gopls -- gopls mcp` | agent queries by name/path, not cursor | headless, disk-only files |
| **Native `LSP` tool** | `ENABLE_LSP_TOOL=1` + official gopls plugin | you already have a `line:col`; free auto-diagnostics after edits | `line`/`character` |
| **`gopls` CLI** | none | one-shot scripted checks; documented as experimental/debugging-only | `file:line:col` or `file:#offset` |

Preference order: **MCP → native `LSP` → CLI**. MCP tools match how an agent thinks (by name/path); the native tool adds automatic diagnostics after every edit; the CLI is the documented last resort. Wire as many as you have and let the task pick the tool.

## Capability map

Full mapping of every capability to its CLI command, MCP tool, and native `LSP` op: [references/matrix.md](references/matrix.md).

- **Navigation** — definition, implementations, call graph, references. Details: [features.md](references/features.md#navigation).
- **Code discovery** — workspace shape (`go_workspace`), fuzzy symbol search (`go_search`), a dependency's public surface (`go_package_api`).
- **Documentation** — hover type/doc/size, signature help, rendered package docs (`source.doc`, including internal packages pkg.go.dev never sees).
- **Diagnostics & safety** — compiler/analyzer errors after every edit, plus a lightweight reachability `go_vulncheck`.
- **Formatting** — gofmt-equivalent + import organization (scriptable or code action).
- **Refactoring** — safe rename (blocks a break to interface satisfaction), extract/inline, and the `refactor.rewrite.*` family (fill struct/switch, invert if, split/join lines, remove unused param, add struct tags, implement interface). Gotchas: [features.md](references/features.md#transformation).

## Read workflow (understand first)

1. `go_workspace` — layout (module/workspace/GOPATH); baseline when it hasn't run.
2. `go_search` — fuzzy-locate a type/function/variable.
3. `go_file_context` — what a file pulls in from its package; re-run if its deps change.
4. `go_package_api` — a dependency's or sibling package's public surface without reading every file.

## Edit workflow (iterate to clean diagnostics)

1. Read first (workflow above).
2. `go_symbol_references` before any definition change — judge blast radius, then read and edit every referencing file.
3. Make all planned edits, including reference sites, before moving on.
4. `go_diagnostics` on every changed file — mandatory after each modification.
5. Apply suggested quick-fixes after reviewing their diffs, then re-run diagnostics. Ignore hint/info noise unrelated to the task; a message may paraphrase its source.
6. Only when `go.mod` changed: `go_vulncheck` on the whole workspace — after diagnostics are clean.
7. `go test <changed-package-paths>` — not `./...` unless asked, to keep the loop fast.

## Gotchas

- `references` reflects only the **build config of the queried file** — a query on `foo_test.go`/`foo_windows.go` misses matches under other build tags; re-run under the relevant `GOOS` when a cross-platform match is missing.
- `call_hierarchy` only shows **static** calls — function values and interface dispatch are invisible; corroborate with `references`.
- Extract/inline are less rigorous than rename: comments can be dropped, and `DO NOT EDIT` generated files get no code actions.
- `refactor.rewrite.fillStruct` searches only the current file above the cursor and needs the package imported — run `source.organizeImports` first for a freshly typed type.

## gopls vs godig vs Context7 vs govulncheck

`gopls` reasons only about code present and resolvable in the local build:

| Tool | Answers | Local build needed? |
| --- | --- | --- |
| **gopls** | local symbols, call sites, diagnostics, safe rename | yes |
| **godig** (`golang-pkg-go-dev`) | version history, license, importers, CVEs of any package | no |
| **govulncheck** (`golang-security`) | comprehensive whole-tree CVE audit (CI gates) | yes |
| **Context7** | non-Go docs or modules not indexed on pkg.go.dev | no |

## Cross-references

- `golang-pkg-go-dev` — published-ecosystem facts via godig.
- `golang-security` — whole-tree govulncheck audits.
- `golang-refactoring` / `golang-modernize` — safe-rename and rewrite-driven workflows built on gopls.
- `golang-how-to` — the full "godig vs gopls vs Context7 vs govulncheck" task-to-tool matrix.

Set up details, every MCP tool, CLI reference, and settings: [mcp.md](references/mcp.md), [cli.md](references/cli.md), [settings.md](references/settings.md).
