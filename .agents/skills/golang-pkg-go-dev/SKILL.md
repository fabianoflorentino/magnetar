---
name: golang-pkg-go-dev
description: "Golang package and module lookup via `godig`, a pkg.go.dev API client (CLI + MCP server). Use for any Go/Golang library's documentation, API signatures, symbols, usage examples, which versions exist, licenses, whether a dependency has CVEs, or who imports a package — prefer this over Context7 for any Go package or module. Read-only, no auth. Not for upgrading dependencies (→ See `fabianoflorentino/golang-agent-skills@golang-dependency-management` skill), choosing a library (→ See `fabianoflorentino/golang-agent-skills@golang-popular-libraries` skill), or local symbols and an already-used dependency's resolved source, call sites, and generic instantiations (→ See `fabianoflorentino/golang-agent-skills@golang-gopls` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness. Requires the godig CLI (go install github.com/samber/godig/cmd/godig@latest) or access to a godig MCP server, and internet access to reach the pkg.go.dev API.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔎"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - godig
    install:
      - kind: go
        package: github.com/samber/godig/cmd/godig@latest
        bins: [godig]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Bash(godig:*) Agent
---

# Correct docs? Ask pkg.go.dev via godig

**Persona:** You are a Go engineer who verifies against the published ecosystem, not against memory. Docs, versions, licenses, CVEs, importers — one read-only CLI call each, no auth, no guessing.

**Modes:**

- **Lookup** — answer a specific question (version, symbol signature, CVE, importer). Start with `overview`; drill only where needed. Sequential, but parallelize independent calls.
- **Selection** — compare candidate modules for a task: overview each, check versions and vulns, read focused docs. Fan out sub-agents for wide comparisons. Parallel.
- **Verify** — confirm a symbol/API before writing code against it; pin the exact version. Sequential.

**When to use:** any question about a published Go package/module. Not dependency upgrades (`golang-dependency-management`), not library choice (`golang-popular-libraries`), not your locally-resolved tree (`golang-gopls`).

## godig vs gopls vs Context7 vs govulncheck

One task, one tool:

- **godig** — the published ecosystem: works for packages not yet in `go.mod`; answers versions, docs, symbols, licenses, importers, CVEs.
- **gopls** — your *local* resolved build: `go.sum`, including `replace`d forks; call sites, definitions, generics instantiation. See `golang-gopls` for wiring (MCP server, `LSP` tool, CLI).
- **Context7** — fallback for non-Go or un-indexed docs.
- **govulncheck** — the whole-tree vulnerability audit. See `golang-security`.

Full task-to-tool matrix: the "godig vs gopls vs Context7 vs govulncheck" section in `golang-how-to`.

## Setup

```bash
go install github.com/samber/godig/cmd/godig@latest
```

Optionally register the MCP server. stdio (client launches godig on demand):

```bash
claude mcp add pkg-go-dev -- godig mcp
```

streamable HTTP, or the hosted instance (no install):

```bash
godig mcp --transport http --addr :8080
claude mcp add --transport http pkg-go-dev http://localhost:8080/mcp
claude mcp add --transport http pkg-go-dev https://godig.samber.dev/mcp
```

Other harnesses register MCP servers in their own settings — the same `godig mcp` command or hosted URL, no shared format. The CLI and MCP server expose the same operations; prefer the CLI when installed, the hosted instance otherwise.

## Commands

Global flags (all commands, also as `GODIG_*` env vars): `-o/--output table|json|raw|md` (default `table`; pass `-o md` in chat), `--base-url`, `--vuln-base-url`, `--timeout`, `--log-level`.

| Command | Args | Specific flags | Use |
| --- | --- | --- | --- |
| `overview` | `<path>` | `--version` | Compact summary — start here |
| `search` | `<query>` | `--symbol --limit --filter` | Find packages |
| `package info` / `imports` / `doc` / `examples` / `licenses` | `<path>` | `--module --version --goos --goarch [--format for doc]` | Package facets (doc/examples/licenses are LARGE) |
| `symbol doc` / `symbol examples` | `<path> <symbol>` | `--module --version --goos --goarch` | One symbol (token-efficient) |
| `symbols` | `<path>` | `--module --version --goos --goarch --limit --filter` | Exported symbol list |
| `module info` / `licenses` / `readme` | `<path>` | `--version` | Module facets (readme/licenses LARGE) |
| `dependencies` | `<path>` | `--version` | go.mod requires/replaces/excludes/go |
| `packages` | `<path>` | `--version --limit --filter` | Packages inside a module |
| `versions` / `major-versions` | `<path>` | `--limit --filter [--exclude-pseudo]` | Version lists (major-versions shows v1/v2 sub-modules) |
| `imported-by` | `<path>` | `--module --version --limit --filter` | Consumers of a package |
| `vulns` | `<path>` | `--version --limit` | Known vulnerabilities (Go vuln DB) |
| `mcp` | — | `--transport --addr --cache-ttl --cache-size` | Run as a server |
| `version` | — | — | godig build info |

Exit codes: `0` success, `1` runtime error (network, not found), `2` usage error (bad arg/flag; e.g. a non-positive `--limit`, or a group like `godig package` without a subcommand). Check for `2` to tell a malformed call from a failed lookup.

Sample `-o md` output for every command: [`references/sample-output.md`](./references/sample-output.md).

## Working patterns

- **Start with `overview`** — metadata, recent + latest versions, licenses, vulns in one shot. Reach for LARGE outputs (`package doc`, `examples`, `module readme`, `licenses`) only when you need the full text.
- **Always `-o md`** so results render in chat.
- **Prefer `symbol doc`/`symbol examples`** over the package-wide equivalents — far fewer tokens.
- **Parallelize independent lookups.** Each call is a self-contained read-only query; issue several at once (wall-clock = slowest call). For a large fan-out (many symbols, library comparisons, CVE sweeps), dispatch up to 5 sub-agents each running its own `godig` calls and returning a compact summary — keep LARGE output out of the main context.
- Listing commands auto-paginate; `--limit` caps them.
- `--version` pins (`v1.5.0`, `latest`, `master`); `--module` disambiguates packages shared by modules; `--goos`/`--goarch` set the doc build context.

## Filter syntax

`--filter` (on lists) is a **Go boolean expression evaluated server-side per item** — not a regex; quote it for the shell. Its identifiers are the item's fields, which differ per command (an unknown field fails with HTTP 400 naming it); enum-like `kind` values are capitalized (`Function`).

Operators: `== != < <= > >= && || !` and parentheses. String helpers: `contains(s, sub)`, `hasPrefix(s, pre)`, `hasSuffix(s, suf)`. Literals: double-quoted strings, `true`/`false`, numbers.

| Command | Filterable fields |
| --- | --- |
| `search` | `modulePath`, `packagePath`, `synopsis`, `version` |
| `versions` | `version`, `modulePath`, `deprecated`, `retracted`, `hasGoMod`, `commitTime` |
| `packages` | `path`, `name`, `synopsis`, `isRedistributable` |
| `imported-by` | `path` |
| `symbols` | `name`, `kind`, `synopsis`, `parent` |
| `major-versions` | `modulePath`, `major`, `version`, `isLatest` |

```bash
godig symbols github.com/samber/lo --filter 'kind=="Function" && hasPrefix(name,"Map")' -o md
godig versions github.com/samber/lo --filter 'hasPrefix(version,"v1.5")' -o md
godig versions github.com/samber/lo --filter 'deprecated==false && retracted==false' -o md
godig search "result option" --filter 'hasPrefix(packagePath,"github.com/samber/")' -o md
```

## Examples

```bash
godig overview github.com/samber/ro -o md                     # start here
godig search "result option monad" --limit 5 -o md
godig package doc github.com/samber/ro --format md -o md
godig symbol doc github.com/samber/lo Map -o md               # token-efficient
godig symbol examples github.com/samber/oops OopsError.Error -o md
godig module readme github.com/samber/ro -o raw
godig dependencies github.com/samber/ro -o md
godig versions github.com/samber/ro --limit 5 -o md
godig imported-by github.com/samber/ro --limit 20 -o md
godig package doc github.com/samber/lo --version v1.50.0 -o md
godig symbols github.com/samber/ro --goos linux --goarch amd64 -o md
godig vulns github.com/samber/ro -o md
```

This skill is not exhaustive — `godig --help` and each subcommand's `--help` list current flags and formats; the data mirrors pkg.go.dev. Bugs in godig: open an issue at <https://github.com/samber/godig/issues>.
