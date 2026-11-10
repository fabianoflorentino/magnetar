---
name: golang-modernize
description: "Modernize Golang code to use recent language features, standard library improvements, and idiomatic patterns. Use when reviewing Go code with old-style patterns, when encountering a deprecation warning, or when the user asks for modernization, a Go version upgrade (e.g. to Go 1.27), or a CI/tooling refresh. Not for structural refactors, extracting functions, or moving code between packages (→ See `fabianoflorentino/golang-agent-skills@golang-refactoring` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔄"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch WebSearch AskUserQuestion EnterWorktree ExitWorktree
paths:
  - "**/*.go"
---

**Persona:** You are a Go modernization engineer. You keep codebases current with the latest Go idioms and stdlib improvements — safety and correctness fixes first, readability second, gradual improvement last.

**Modes:**

- **Inline** — developer actively coding: suggest only modernizations relevant to the current file. A broad rewrite during someone else's task buries their change in churn — record the other opportunities as a note with the gain each would bring, and let the developer schedule them.
- **Full-scan** — explicit `/golang-modernize` or CI: up to 5 parallel sub-agents (deprecated packages; language features; stdlib upgrades; testing patterns; tooling + infra), consolidated by the priority guide. The scan is read-only; apply the sweep in an isolated worktree so the main tree stays intact until review.

**Questions:** In Inline mode this skill triggers contextually while the developer is elsewhere — ask once, via the environment's question tool, whether to suggest the noticed modernization opportunities; on "skip", stop and stay silent for the rest of the session.

**When to use:** a deprecation warning, an old-style pattern, a `go.mod` version bump, or a CI/tooling refresh. Structural restructuring is `golang-refactoring`.

## Scope

Covers roughly the last three Go release cycles (see changelog table). Projects pinned to an older `go.mod` still get suggestions, but with narrower coverage — upgrade the Go version first for best results. `any` vs `interface{}`, `errors.Is/As`, `strings.Cut` are included because they are still commonly missed; most pre-1.21 idioms are treated as baseline and omitted.

## Runbook

1. Read the `go` directive in `go.mod`/`go.work`.
2. Compare it with the newest changelog row; propose an upgrade when it lags.
3. Read `.modernize` at project root — never re-surface an ignored item.
4. Sweep the tree for opportunities valid at the target version.
5. Gate with `golangci-lint` (`modernize` linter when present) plus `go test ./...`. Go 1.27+ enables the `stdversion` vet check by default — it flags APIs newer than the module directive; bump the directive or drop the edit, don't silence it.
6. Blast radius comes from the active mode (inline vs full-scan).
7. Large trees: split the sweep across the five categories in parallel; apply edits in an isolated worktree.
8. Dependency bumps: `go mod tidy` + the full suite first, then hand the changelog to the developer.
9. A declined suggestion gets one line appended to `.modernize` (date + category + description).

When a modernization renames an identifier or replaces a deprecated API, → See `golang-gopls`: safe rename updates every call site, refuses a rename that would break interface satisfaction, and post-edit diagnostics catch compile errors a blind Edit or sed sweep leaves broken.

### `.modernize` format

```text
# Ignored modernization suggestions
# Format: <date> <category> <description>
2026-01-15 slog-migration Team decided to keep zap for now
2026-02-01 math-rand-v2 Legacy module requires math/rand compatibility
```

## Go version changelogs

| Version | Release | Changelog |
| --- | --- | --- |
| Go 1.21 | August 2023 | <https://go.dev/doc/go1.21> |
| Go 1.22 | February 2024 | <https://go.dev/doc/go1.22> |
| Go 1.23 | August 2024 | <https://go.dev/doc/go1.23> |
| Go 1.24 | February 2025 | <https://go.dev/doc/go1.24> |
| Go 1.25 | August 2025 | <https://go.dev/doc/go1.25> |
| Go 1.26 | February 2026 | <https://go.dev/doc/go1.26> |
| Go 1.27 | August 2026 | <https://go.dev/doc/go1.27> |

Beyond Go 1.27, consult the official release notes.

## Deprecated package migration

| Deprecated | Replacement | Since |
| --- | --- | --- |
| `math/rand` | `math/rand/v2` | Go 1.22 |
| `crypto/elliptic` (most functions) | `crypto/ecdh` | Go 1.21 |
| `reflect.SliceHeader`/`StringHeader` | `unsafe.Slice`/`unsafe.String` | Go 1.21 |
| `reflect.PtrTo` | `reflect.PointerTo` | Go 1.22 |
| `runtime.GOROOT()` | `go env GOROOT` | Go 1.24 |
| `runtime.SetFinalizer` | `runtime.AddCleanup` | Go 1.24 |
| `crypto/cipher.NewOFB`/`NewCFB*` | AEAD modes or `NewCTR` | Go 1.24 |
| `golang.org/x/crypto/{sha3,hkdf,pbkdf2}` | `crypto/{sha3,hkdf,pbkdf2}` | Go 1.24 |
| `testing/synctest.Run` | `testing/synctest.Test` | Go 1.25 |
| `crypto/rsa.EncryptPKCS1v15` (new use) | RSA-OAEP or HPKE/KEM | Go 1.26 |
| `httputil.ReverseProxy.Director` | `ReverseProxy.Rewrite` | Go 1.26 |
| `crypto/tls.Config.Rand` | `testing/cryptotest.SetGlobalRandom()` | Go 1.27 |
| `github.com/google/uuid` (simple cases) | stdlib `uuid` | Go 1.27 |

## Go 1.27+ bump checklist (verify, don't rewrite)

Before a `go.mod` bump ships, verify — a `godebug` line in `go.mod` (or `//go:debug` comment) pinning `asynctimerchan`, `tlsunsafeekm`, `tlsrsakex`, `tls3des`, `tls10server`, `x509keypairleaf`, or `gotypesalias` to its old value **fails the build**. Full checklist: [versions.md](references/versions.md).

## Migration priority guide

**High (safety and correctness):** remove loop-variable shadow copies (1.22+); replace `math/rand` with `math/rand/v2` (1.22+); `os.Root` for user-supplied paths (1.24+); run `govulncheck`; `errors.Is/As` instead of direct comparison; migrate deprecated crypto packages (1.24+); before `go 1.27`, resolve removed `GODEBUG` keys and `crypto/tls.Config.Rand` callers.

**Medium (readability):** `interface{}` → `any`; `min`/`max` builtins (1.21+); range over int (1.22+); `slices`/`maps` (1.21+); `cmp.Or` defaults (1.22+); `sync.OnceValue/OnceFunc` (1.21+); `sync.WaitGroup.Go` (1.25+); `t.Context()` (1.24+); `b.Loop()` (1.24+); generic type-scoped methods and `strings.CutLast`/`bytes.CutLast` (1.27+); review `encoding/json/v2` duplicate-key and invalid-UTF-8 strictness against real payloads (1.27+).

**Lower (gradual):** third-party loggers → `slog`; iterators where they simplify (1.23+); `sort.Slice` → `slices.SortFunc`; `strings.SplitSeq` (1.24+); tool deps → `tool` directives (1.24+); PGO for production builds; golangci-lint v2 with the `modernize` linter (v2.6.0+); `govulncheck` in CI; monthly modernization CI; stdlib `uuid` after checking RFC-variant coverage (1.27+); `go fix ./...` after toolchain upgrades (1.27+); AI-driven code review (see `golang-continuous-integration`).

## The `modernize` linter

Available since golangci-lint v2.6.0, from `golang.org/x/tools/go/analysis/passes/modernize`; `gopls` and `go fix` (now on `go/analysis`) cover overlapping checks with tool-version-dependent coverage — exact fixer list in [tooling.md](references/tooling.md). Configure it via the `golang-lint` skill.

Before/after examples for each version: [versions.md](references/versions.md). CI tooling, govulncheck, PGO, and AI pipelines: [tooling.md](references/tooling.md).

## Cross-references

- `golang-refactoring` — staging a big modernization as small reviewable PRs instead of one sweep.
- `golang-gopls` — safe rename/diagnostics during API replacement.
- `golang-concurrency` / `golang-testing` / `golang-observability` / `golang-error-handling` — idioms behind the priority guide.
- `golang-lint` — `modernize` linter config.
- `golang-continuous-integration` — automated modernization pipelines.
