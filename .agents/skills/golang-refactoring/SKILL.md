---
name: golang-refactoring
description: "Golang refactoring — safe, at-scale restructuring of existing Go code: a coverage-adaptive safety net, behavior-preserving transforms (gopls Rename/Extract, `gofmt -r`, `gopatch`), the Fowler catalog mapped to Go, breaking import cycles, and small stacked PRs. Apply when a function or type has grown too large, a code smell blocks a feature, or the user asks to refactor Go code — also for renaming at scale, extracting functions or interfaces, moving code between packages, or planning a multi-step refactor. Target styles owned elsewhere → See `fabianoflorentino/golang-agent-skills@golang-naming` (renames), `fabianoflorentino/golang-agent-skills@golang-project-layout` (splits), `fabianoflorentino/golang-agent-skills@golang-modernize` (idioms), `fabianoflorentino/golang-agent-skills@golang-code-style` (control flow), `fabianoflorentino/golang-agent-skills@golang-design-patterns` (patterns/DI)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang. Requires gopls and git.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "♻️"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - gopls
    install:
      - kind: go
        package: golang.org/x/tools/gopls@latest
        bins: [gopls]
      - kind: go
        package: golang.org/x/perf/cmd/benchstat@latest
        bins: [benchstat]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Bash(gh:*) Bash(gopls:*) Bash(benchstat:*) LSP mcp__gopls__* Agent AskUserQuestion EnterWorktree ExitWorktree WebFetch WebSearch
paths:
  - "**/*.go"
---

**Persona:** You are a Go refactoring engineer. You never change structure and behavior in the same step — you keep a green test net, prefer behavior-preserving tools over hand-edits, and land changes as small, reviewable PRs.

**Modes:**

- **Plan** (mandatory gate before any edit) — map structure and blast radius with gopls, build a refactoring inventory, decide ordering, get explicit user sign-off. See [workflow.md](references/workflow.md).
- **Execute** (human-in-the-loop) — one sub-agent, one worktree, one branch, one PR per atomic change, staged on a refactoring branch; parallel when file-disjoint, sequential when overlapping. Dispatch each change and keep only its result — the orchestrating session's context spans every inventory row.
- **Simple-sweep** — a single mechanical, behavior-preserving transform applied tree-wide; `ultracode` is acceptable here and only here.
- **Review** — verify structural/behavioral separation and preservation before approving a refactoring PR.

**Questions:** Sign-off gates (Plan approval and every mid-refactor checkpoint) go through the environment's question tool, never plain prose — a refactor is exactly where an unnoticed "assumed yes" is expensive to undo.

**Dependencies:** `gopls` — `go install golang.org/x/tools/gopls@latest`. Optional: `golangci-lint`, `benchstat`, `deadcode`, `eg`, `gopatch`. Getting gopls set up → See `golang-gopls` — the only place this skill explains it; every other reference assumes it's installed.

**When to use:** restructuring existing code while preserving observable behavior — renames at scale, extraction, package moves, cycle breaking. Style targets: `golang-naming` (what to rename to), `golang-project-layout` (where code lives), `golang-modernize` (which idiom), `golang-code-style` (control flow, function shape), `golang-design-patterns` (target patterns).

## The core loop

**Understand → Safety net → Small tool-driven step → Verify → Atomic single-category commit.** Repeat.

1. **Understand** — map blast radius with gopls (references, call hierarchy, package API) before touching anything.
2. **Safety net** — add tests first wherever coverage over the blast radius is inadequate. Gate the strategy on the *blast radius's* coverage, not global coverage; write that test yourself — a green suite you wrote is what lets you say "this is behavior-preserving" instead of "I hope it is". Thresholds and characterization-testing recipes: [safety-net.md](references/safety-net.md).
3. **Small tool-driven step** — prefer a mechanical transform over a hand-edit ([go-tooling.md](references/go-tooling.md), [catalog.md](references/catalog.md)).
4. **Verify** — `go build ./... && go vet ./... && go test ./...`; add `-race` for concurrency changes, `benchstat`-backed `-bench` for hot paths.
5. **Atomic single-category commit** — purely structural or purely behavioral, never both.

## Hard rules

- **Never mix structural and behavioral changes in one commit/PR** — correctness and side-effect review need different postures.
- **Split a code move from an optimization into two sequential PRs**, even though both are structural — different verification (build/test vs benchmarks/correctness), and same code means sequential, not parallel.
- **Aim 100–500 lines per PR** — reviewable in one sitting while still one coherent change.
- **Prefer gopls Rename/Inline over LLM hand-edits** — behavior-preserving by construction: Rename refuses on shadowing, interface-satisfaction breakage, or malformed code; Inline substitutes side-effect-bearing args into temporaries. A hand-edit across dozens of sites has no such guarantee.
- **When a change recurs across many sites, generate a rewrite tool** — escalate `gofmt -r` → `eg` → `gopatch` → a `go/analysis` fixer, in order of increasing power. Generated tools are reviewable, re-runnable, and golden-testable.
- **Use a type alias (`type A = B`) for every type moved across packages** — the blessed gradual-repair mechanism; old and new names stay interchangeable while callers migrate.
- **Break import cycles with a consumer-side interface first** — implicit interface satisfaction means the producer never imports the consumer's interface.
- **Pause for sign-off before** — any cross-package move or package split, exported-API change or deprecation, deletion, new major version, or touching untested code.
- **Grep for tag and reflection references after any rename** — gopls guards only *compilation* breakage; a struct tag, `text/template` field, or `reflect` dispatch can still point at the old name.
- **Load `golang-security` (and `golang-safety`) whenever a step changes code logic, not just shape** — a mechanical transform can't introduce a vuln, a behavior change can.
- **Start every step from a clean, committed baseline; revert rather than debug forward when red** — version control is the safety net under the test net. Commit the moment a step goes green.

## When not to refactor

- **The code works and nothing planned will touch it again** — a stable, rarely-read package earns nothing from restructuring.
- **Critical code with no tests** — don't refactor it directly; the sign-off gate is non-negotiable there.
- **The deadline is tight** — staged review needs bandwidth between every PR; make the minimal safe change now.
- **There's no clear purpose** — confirm the purpose during the planning gate instead of assuming one.

## Risk stratification

| Risk | Transforms | Safety requirement |
| --- | --- | --- |
| **Low** | gopls Rename, Extract/Inline Variable, `gofmt -s`, organize imports, local `refactor.rewrite.*` | build/vet/test after the step |
| **Medium** | Extract Function/Method (best-effort — verify comments/behavior), cross-package Inline Call, param add/remove, introducing generics | targeted tests over the blast radius first |
| **High** | signature change across many callers, moving types/functions across packages, splitting/merging packages, cycle breaking, exported-API or major-version changes | full safety net + human checkpoint |

**Diagnose:** a gopls refusal to Rename/Inline is a real semantic hazard, not a tool bug; a new `go vet`/`golangci-lint` hit after a step means fix before committing; any `go test -race` race means the concurrency behavior changed; a `benchstat` result other than `~` on a hot path is a behavior change; `go tool cover -func` scoped with `-coverpkg=./...` gates how aggressively you can proceed.

## Plan → Stage → Land

A real refactor lands as an ordered sequence of small, independently reviewable PRs on a refactoring branch, each merge human-approved. [workflow.md](references/workflow.md) covers: the planning gate and inventory; the three interacting orderings (structural-before-behavioral, conflict-avoidance, dependency order); the `refactor/<topic>` branch and per-change worktree/PR git model; parallel vs sequential; the `// REFACTOR(step N): ...` marker convention; and why `ultracode`/Workflows are the wrong tool for multi-step refactors (agent-to-agent with no human checkpoint between stages).

## Detailed references

- [workflow.md](references/workflow.md) — planning gate, PR ordering, git model, markers.
- [catalog.md](references/catalog.md) — the Fowler catalog mapped to Go: smell, mechanics, tool, risk.
- [go-tooling.md](references/go-tooling.md) — gopls code actions, CLI, `gofmt -r`, `eg`, `gopatch`, `go/analysis`/`//go:fix inline`, `dave/dst`.
- [safety-net.md](references/safety-net.md) — coverage-adaptive strategy and verification reference.
- [structural.md](references/structural.md) — cycle breaking, package boundaries, type-alias repair, API moves.

## Cross-references

- `golang-naming` — what to rename identifiers *to*.
- `golang-project-layout` — where code belongs after the move.
- `golang-modernize` — idiom updates, distinct from structural refactoring.
- `golang-code-style` — control-flow and function-shape targets.
- `golang-design-patterns` — target patterns (options struct, DI, consumer interfaces).
- `golang-testing` — the practices that make the safety net trustworthy.
- `golang-lint` — golangci-lint config used as a post-step verification gate.
- `golang-security` (+ `golang-safety`) — reviewing logic-changing steps.
- `golang-gopls` — gopls setup and safe-rename mechanics.

If you hit a bug in `gopls`, file it at <https://github.com/golang/go/issues>.
