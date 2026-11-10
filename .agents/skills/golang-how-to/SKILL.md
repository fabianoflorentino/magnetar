---
name: golang-how-to
description: "Golang skills orchestrator — always active on any Golang coding, review, debug, or setup task. Reads the task context and loads the most relevant skills from fabianoflorentino/golang-agent-skills, often multiple at once: writing a gRPC service loads golang-grpc + golang-testing + golang-error-handling; debugging a panic loads golang-troubleshooting + golang-safety; auditing security loads golang-security + golang-lint + golang-safety. Also: disambiguates competing clusters when two skills seem to overlap (performance vs benchmark vs troubleshooting, samber/lo vs mo vs ro, DI cluster, safety vs security), and configures the project's agent-config file (CLAUDE.md, AGENTS.md, GEMINI.md, Cursor rules, or Copilot instructions) to force-trigger skills in a project (/golang-how-to configure)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness. Requires git.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧭"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - gopls
    install:
      - kind: go
        package: golang.org/x/tools/gopls@latest
        bins: [gopls]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(git:*) Agent AskUserQuestion LSP Bash(gopls:*) mcp__gopls__*
---

**Persona:** You are a Go skills orchestrator. For every Go task you identify all relevant skills and load them together — a task rarely belongs to a single skill.

**Modes:**

- **Orchestrate** — for any Go coding, review, debug, or setup task, load the primary skill plus all applicable secondary skills at once.
- **Disambiguate** — when two skills seem to overlap, show the boundary table. See [disambiguation.md](references/disambiguation.md).
- **Configure** — write the always-load directive for `golang-how-to` itself (plus an optional `## Required Go skills` block) into the project's agent-config file. Follow [project-config.md](references/project-config.md).

**Questions:** In Configure mode, ask through the environment's question tool — one question at a time, wait for the answer. If the harness has no question tool, fall back to prose with the same options.

**Dependencies:** `gopls` — `go install golang.org/x/tools/gopls@latest`. Claude Code's native `LSP` tool additionally needs `ENABLE_LSP_TOOL=1` and a Go language server wired up (see [Code navigation with gopls](#code-navigation-with-gopls)).

**When to use:** always active on Golang tasks — its job is to route to the right skills, load several at once, resolve overlaps, and set up force-triggers in agent-config files.

## Routing table

Load the **primary** and all applicable **secondary** skills together at the start; do not wait.

| Intent | Primary | Also load |
| --- | --- | --- |
| Design an API / choose a pattern | `golang-design-patterns` | `golang-structs-interfaces`, `golang-naming` |
| Name a type, function, package | `golang-naming` | `golang-code-style` |
| Idiomatic error handling | `golang-error-handling` | `golang-safety` (nil-heavy) |
| Goroutines, channels, sync | `golang-concurrency` | `golang-context` (cancellation) |
| Deadlines / cancellation | `golang-context` | `golang-concurrency` |
| Structs, embedding, interfaces | `golang-structs-interfaces` | `golang-design-patterns` |
| Queries and transactions | `golang-database` | `golang-error-handling`, `golang-security` |
| gRPC / GraphQL / REST API | `golang-grpc` / `golang-graphql` / `golang-rest` | `golang-testing`, `golang-error-handling` (+ `golang-swagger` for OpenAPI) |
| CLI command tree | `golang-spf13-cobra` | `golang-cli`, `golang-spf13-viper` (config) |
| Config layering | `golang-spf13-viper` | `golang-spf13-cobra` |
| Tests | `golang-testing` | `golang-stretchr-testify` |
| Performance | `golang-performance` | `golang-benchmark` (measure first) |
| Profile / benchstat | `golang-benchmark` | `golang-performance`, `golang-troubleshooting` |
| Debug a panic | `golang-troubleshooting` | `golang-safety`, `golang-benchmark` |
| Production monitoring | `golang-observability` | `golang-performance` (SLO breach) |
| Security audit | `golang-security` | `golang-safety`, `golang-lint` |
| Refactor / restructure | `golang-refactoring` | `golang-naming`, `golang-code-style`, `golang-project-layout` |
| golangci-lint config | `golang-lint` | `golang-code-style` |
| godoc / README / CHANGELOG | `golang-documentation` | `golang-naming` |
| New project layout | `golang-project-layout` | `golang-design-patterns`, `golang-dependency-injection`, `golang-lint` |
| CI/CD pipeline | `golang-continuous-integration` | `golang-lint`, `golang-security` |
| Library selection | `golang-popular-libraries` | the library-specific skill |
| pkg.go.dev facts | `golang-pkg-go-dev` | `golang-dependency-management` |
| Local navigation / rename | `golang-gopls` | — |
| New language features | `golang-modernize` | `golang-lint` |
| samber/lo helpers | `golang-samber-lo` | `golang-data-structures`, `golang-performance` |
| Structured errors (samber/oops) | `golang-samber-oops` | `golang-error-handling` |
| log/slog | `golang-samber-slog` | `golang-observability`, `golang-error-handling` |
| Dependency injection | `golang-dependency-injection` | wire / dig / fx / samber-do |
| Pitfall-domain audits | the matching `golang-pitfalls-*` | that domain's general skill |

Short names above stand for `fabianoflorentino/golang-agent-skills@<name>`. Full catalog with "use when" hooks: [by-category.md](references/by-category.md).

## Code navigation with gopls

`gopls` provides semantic Go intelligence — definitions, references, diagnostics, package API, symbol search, refactoring. → See `golang-gopls` for the three ways to reach it (its MCP server, the native `LSP` tool, its CLI), the capability matrix, and the read/edit workflows.

`gopls` reasons only about the locally resolved build: your workspace plus every dependency exactly as pinned in go.sum, including `replace` directives. For facts outside that build — versions, licenses, ecosystem importers, a not-yet-added package — use `golang-pkg-go-dev` (`godig`).

## `godig` vs gopls vs Context7 vs govulncheck

Four tools answer "is this dependency OK to use," and they overlap less than they look:

- **Context7** — a general cross-language documentation fetcher; a fallback when no more specific source exists. For a Go module, `godig` is almost always the better pick: it pulls structured, Go-specific data straight from pkg.go.dev — exact versions, exported symbols with signatures, runnable examples, `imported-by`, known vulnerabilities — rather than Context7's generic scraped docs, which lack that structure and can lag or miss lesser-known modules. Use Context7 only when a dependency's docs genuinely don't exist or aren't on pkg.go.dev.
- **`godig`** — the published ecosystem: any Go package or module whether or not it's in your go.mod, via the remote pkg.go.dev API, never touching your local checkout. Its `vulns` command reports CVEs for a package/version in isolation, regardless of whether the build reaches the code path.
- **`gopls`** — your specific build: your code plus every dependency as pinned, including `replace` forks or local paths — neither `godig` nor Context7 can see those. Its `go_vulncheck` operation is a single on-demand reachability check against the workspace as it stands.
- **`govulncheck`** (standalone CLI, wrapped by the `golang-security` skill) — the whole-tree audit: walks the entire module call graph to confirm which known vulnerabilities are actually reachable; the tool of record for CI gates and periodic sweeps. `gopls`'s `go_vulncheck` is a lighter, single-shot version for mid-edit use.

Task → tool:

| Task | Tool | Route |
| --- | --- | --- |
| Symbol defined in my repo | `gopls` | `go_search` → `go_file_context` |
| Intra-package deps of a file | `gopls` | `go_file_context` |
| A dependency's resolved source (incl. forks/replaced) | `gopls` | `go_package_api` / `goToDefinition` |
| Call sites of a dependency symbol | `gopls` | `go_symbol_references` (godig's `imported-by` lists packages, not your call sites) |
| Diagnostics after an edit | `gopls` | `go_diagnostics` (or automatic with `LSP`) |
| Mid-edit reachable-CVE check | `gopls` | `go_vulncheck` |
| Rename / extract / inline | `gopls` | safe rename, `refactor.*` code actions |
| Whole-tree CVE audit (CI) | `govulncheck` | `golang-security` |
| Available versions of a package | `godig` | `godig versions <path>` |
| CVEs of a package you haven't added | `godig` | `godig vulns <path>` |
| Exported symbols / signatures | `godig` | `godig symbols` / `symbol doc` |
| Runnable examples | `godig` | `godig symbol examples` |
| Rendered README / docs | `godig` | `godig module readme` / `package doc` |
| Ecosystem importers | `godig` | `godig imported-by` |
| Package / library search | `godig` | `godig search` |
| License check | `godig` | `godig package licenses` / `module licenses` |
| Non-Go lib or unindexed Go module | Context7 | `resolve-library-id` / `query-docs` |

Full `godig` command reference lives in `golang-pkg-go-dev`; the whole-tree remediation workflow in `golang-security`.

## Categories at a glance

| Category | Skills |
| --- | --- |
| Code Quality | `golang-code-style` `golang-documentation` `golang-error-handling` `golang-lint` `golang-naming` `golang-safety` `golang-security` `golang-structs-interfaces` |
| Architecture & Design | `golang-concurrency` `golang-context` `golang-data-structures` `golang-database` `golang-dependency-injection` `golang-design-patterns` `golang-modernize` `golang-refactoring` |
| QA & Performance | `golang-benchmark` `golang-observability` `golang-performance` `golang-testing` `golang-troubleshooting` |
| Project Setup | `golang-cli` `golang-continuous-integration` `golang-dependency-management` `golang-gopls` `golang-pkg-go-dev` `golang-popular-libraries` `golang-project-layout` `golang-stay-updated` |
| APIs | `golang-rest` `golang-graphql` `golang-grpc` `golang-swagger` |
| Dependency Injection | `golang-dependency-injection` `golang-google-wire` `golang-uber-dig` `golang-uber-fx` `golang-samber-do` |
| Frameworks | `golang-spf13-cobra` `golang-spf13-viper` |
| samber/\* | `golang-samber-do` `golang-samber-hot` `golang-samber-lo` `golang-samber-mo` `golang-samber-oops` `golang-samber-ro` `golang-samber-slog` |
| Testing | `golang-stretchr-testify` `golang-testing` |
| Pitfalls (100 Go Mistakes) | the eleven `golang-pitfalls-*` skills |

## Competing clusters — who owns what

Boundary tables and routing examples: [disambiguation.md](references/disambiguation.md). Key lines:

- **Performance**: `golang-performance` (patterns) · `golang-benchmark` (measurement) · `golang-troubleshooting` (root cause) · `golang-observability` (always-on production)
- **DI**: `golang-dependency-injection` (decision) · `golang-google-wire` (compile-time) · `golang-uber-dig` (reflection) · `golang-uber-fx` (lifecycle) · `golang-samber-do` (type-safe container)
- **samber/\***: `golang-samber-lo` (finite transforms) · `golang-samber-ro` (reactive) · `golang-samber-mo` (monads)
- **Errors**: `golang-error-handling` (idioms) · `golang-samber-oops` (structured) · `golang-safety` (prevent panics)
- **CLI**: `golang-cli` (architecture) · `golang-spf13-cobra` (tree) · `golang-spf13-viper` (config)
- **Package lookup**: `golang-pkg-go-dev` (pkg.go.dev queries) · `golang-gopls` (local build) · `golang-popular-libraries` (adoption) · `golang-dependency-management` (go.mod)  · `golang-security` (whole-tree CVE)
- **Gaps**: type vs arch (`golang-structs-interfaces` vs `golang-design-patterns`); goroutine vs cancel (`golang-concurrency` + `golang-context` together); correctness vs threat (`golang-safety` vs `golang-security`); features vs rules (`golang-modernize` vs `golang-lint`); process vs target (`golang-refactoring` vs the skills owning the shape it migrates toward).
- **Pitfalls vs general**: `golang-pitfalls-*` are mistake-ordered checklists from _100 Go Mistakes_; load beside their general owners. Pitfalls win on "have we hit mistake X?" audits; general skills win for writing fresh code.

## Configure mode

Write an always-load directive for `golang-how-to` into the project's agent-config file (CLAUDE.md, AGENTS.md, GEMINI.md, Cursor rules, or Copilot instructions — whatever the harness reads), plus an optional `## Required Go skills` block to force-trigger secondary skills.

`golang-project-layout` writes the always-load directive automatically at project creation, with no confirmation needed — one skill description, no project-specific choices. `/golang-how-to configure` writes it too if missing and additionally lets the user confirm the `## Required Go skills` block. Follow [project-config.md](references/project-config.md).

---

This skill is not exhaustive — individual skill files and the official Go documentation carry the depth. If you hit a bug in this plugin, open an issue at <https://github.com/fabianoflorentino/golang-agent-skills/issues>.
