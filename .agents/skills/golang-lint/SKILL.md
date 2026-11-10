---
name: golang-lint
description: "Linting best practices and golangci-lint configuration for Golang projects — running linters, configuring .golangci.yml, suppressing warnings with nolint directives, interpreting lint output, and selecting linters. Use when configuring golangci-lint, asking about lint warnings or nolint suppressions, setting up code quality tooling, or choosing linters. Also use when the user mentions golangci-lint, go vet, staticcheck, or revive. Not for wiring a lint step into a GitHub Actions pipeline (→ See `fabianoflorentino/golang-agent-skills@golang-continuous-integration` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧹"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - golangci-lint
    install:
      - kind: brew
        formula: golangci-lint
        bins: [golangci-lint]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
  - ".golangci.yml"
---

# Linting Go code

**Persona:** You are a Go code quality engineer. Linting is a first-class part of development, not a post-hoc cleanup: every pattern the tool can catch is a review hour returned to the maintainers.

**Modes:**

- **Setup** — create or tune `.golangci.yml`, choose linters, wire the pipeline. Sequential.
- **Coding** — write new code while a background agent runs `golangci-lint run --fix` on changed files; surface its output when done. Parallel.
- **Interpret/Fix** — read lint output, suppress the rare false positive, clean legacy code. Parallel sub-agents per linter category for a legacy sweep.

**When to use:** any Golang project that wants a `.golangci.yml`, any lint warning to interpret or suppress, any linter to choose. CI wiring lives in `golang-continuous-integration`; style decisions beyond tooling in `golang-code-style`; SAST (gosec, govulncheck) in `golang-security`.

## The mental model

`golangci-lint` aggregates 100+ linters under one binary and one config. The file `.golangci.yml` is the single source of truth for which linters run and how they are wired — agreeing on it is the whole point. See the recommended config shipped with this skill ([`assets/.golangci.yml`](./assets/.golangci.yml)) as a production baseline.

## Quick reference

```bash
golangci-lint run ./...               # configured set
golangci-lint run --fix ./...         # auto-fix what it can
golangci-lint fmt ./...               # format (v2+)
golangci-lint run --enable-only govet ./...   # one linter
golangci-lint linters                 # available set
golangci-lint run --verbose ./...     # per-linter timing
```

## Output format

```
path/to/file.go:42:10: message describing the issue (linter-name)
```

The trailing `(linter-name)` is the key. Look that linter up in [`references/linter-reference.md`](./references/linter-reference.md) (what it checks, when it fires, categories) before deciding to fix, suppress, or disable it.

## Suppress only with justification

Fix the root cause first. A suppression is an accepted debt entry and must say so:

```go
// Fine: named linter + reason
//nolint:errcheck // fire-and-forget logging; error not actionable
_ = logger.Sync()
```

Three rules, enforced by the `nolintlint` linter itself:

1. Name the linter — `//nolint:errcheck`, never a bare `//nolint`.
2. Give a reason on the same comment.
3. Never suppress security linters (`gosec`, `bodyclose`, `sqlclosecheck`) without a very strong argument.

## Choosing and configuring linters

Pick by intent, then prove the choice by running:

| Concern | Linters to consider |
| --- | --- |
| Correctness | `govet`, `staticcheck`, `errcheck`, `nilerr`, `ineffassign` |
| Style | `gofumpt`, `revive`, `misspell`, `predeclared` |
| Complexity | `gocritic` |
| Concurrency | `paralleltest`, `thelper` |
| Performance | `prealloc`, moved by cost-governed reviews |
| Security | `gosec`, `bodyclose`, `sqlclosecheck` (see `golang-security`) |

Disabled-by-default linters must be deliberately enabled in `.golangci.yml`; document why in a comment next to each.

## Common issues

| Problem | Solution |
| --- | --- |
| `deadline exceeded` | Raise `run.timeout`; v2 defaults to no timeout |
| Legacy flood | `issues.new-from-rev: HEAD~1` — lint only new code, then slowly widen |
| Linter not found | Version too old — `golangci-lint linters` to confirm |
| Linters conflict | Disable the weaker one with a reason in the config |
| v1 config errors | `golangci-lint migrate` converts the format |
| Slow on large repos | Tune `run.concurrency`, exclude paths |

## Workflow

Run `golangci-lint run ./...` after every significant change; `--fix` what it hands back; format before commit (`golangci-lint fmt ./...`). Makefile targets keep it one word:

```makefile
lint:     golangci-lint run ./...
lint-fix: golangci-lint run --fix ./...
fmt:      golangci-lint fmt ./...
```

## Parallel legacy cleanup

Adopting linting on a legacy tree? Fan out one sub-agent per category so the categories fix concurrently: (1) auto-fix, (2) security linters, (3) error handling (`errcheck`, `wrapcheck`, `nilerr`), (4) style/formatting (`gofumpt`, `goimports`, `revive`), (5) code quality (`gocritic`, `unused`, `ineffassign`).

## Cross-references

- `golang-continuous-integration` — lint CI step (`golangci-lint-action`) and AI review gating.
- `golang-code-style` — the style rules linters enforce.
- `golang-security` — SAST beyond linting (gosec, govulncheck).
- `golang-testing` — test-focused linters (`thelper`, `paralleltest`, `testifylint`).
- `golang-naming` — conventions that `revive`/`predeclared`/`errname` check.
