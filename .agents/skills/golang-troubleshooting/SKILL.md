---
name: golang-troubleshooting
description: "Troubleshoot Golang programs systematically - find and fix the root cause. Use when encountering bugs, crashes, deadlocks, races, or unexpected behavior in Go code. Covers debugging methodology, common Go pitfalls, test-driven debugging, pprof setup and capture, Delve, race detection, GODEBUG tracing, and production debugging. Start here for any 'something is wrong' situation. Not for interpreting profiles or benchmarking (→ See `fabianoflorentino/golang-agent-skills@golang-benchmark` skill), applying optimization patterns (→ See `fabianoflorentino/golang-agent-skills@golang-performance` skill), or designing new code (→ See `fabianoflorentino/golang-agent-skills@golang-safety` skill for defensive coding, `fabianoflorentino/golang-agent-skills@golang-concurrency` skill for concurrency design)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔍"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - dlv
    install:
      - kind: go
        package: github.com/go-delve/delve/cmd/dlv@latest
        bins: [dlv]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Bash(dlv:*) Agent WebFetch WebSearch AskUserQuestion
paths:
  - "**/*.go"
---

# Troubleshooting Go programs

**Persona:** You are a Go systems debugger. Evidence drives, not intuition: instrument, reproduce, trace the root cause — and never propose a fix you cannot explain.

**Modes:**

- **Single-issue debug** (default) — the Golden Rules, sequentially: read the error, reproduce, one hypothesis at a time. No sub-agents for a single known symptom.
- **Codebase bug hunt** — an explicit broad sweep. Fan out one sub-agent per bug category (nil/interface, resources, error handling, races, context/slice/map) and merge. Parallel.

**When to use:** literally any "something is wrong" in Go. Not for profile interpretation/benchmarking (`golang-benchmark`), not for optimization patterns (`golang-performance`), not for defensive design (`golang-safety`, `golang-concurrency`).

## The discipline

**No fixes without root-cause investigation — especially under time pressure.** Symptom fixes cascade into longer outages.

1. Classify the symptom with the decision tree below and jump to that section.
2. Follow the Golden Rules: reproduce before you fix, one hypothesis at a time.
3. Work the methodology step by step; do not skip.
4. Watch your own reasoning for red flags (below).
5. Escalate tools incrementally — `fmt.Println` and test isolation before pprof/Delve/GODEBUG.
6. Never ship a fix you cannot explain; say so and keep digging.

## Decision tree

```
Build won't compile       → go build ./... 2>&1; go vet ./...        → compilation.md
Wrong output / logic bug  → failing test → error handling, nil, off-by-one → common-go-bugs.md, testing-debug.md
Random crashes / panics   → GOTRACEBACK=all ./app; go test -race ./... → common-go-bugs.md, diagnostic-tools.md
Sometimes works, sometimes fails → go test -race ./...                → concurrency-debug.md, testing-debug.md
Program hangs / frozen    → curl localhost:6060/debug/pprof/goroutine?debug=2 → concurrency-debug.md, pprof.md
High CPU                  → pprof CPU profile                          → performance-debug.md, pprof.md
Memory growing            → pprof heap profile                        → performance-debug.md, concurrency-debug.md
Slow / p99 spikes         → CPU + mutex + block profiles              → performance-debug.md, diagnostic-tools.md
Simple, reproducible bug  → test + debug logging                      → testing-debug.md
```

Most Go bugs reduce to: missing error checks, nil dereferences, forgotten context cancel, unclosed resources, races, and swallowed errors.

## Golden rules

1. **Read the error fully first.** File:line takes you straight there; type mismatch means signature/interface questions; "undefined" means imports/exports/build tags; "cannot use X as Y" means concrete vs interface.
2. **Reproduce before fixing.** A failing test that captures the bug, made deterministic, minimized to the smallest example; `git bisect` to find the breaking commit.
3. **If you didn't measure it, you're guessing.** pprof over intuition, the race detector over reasoning, benchmarks over assumptions.
4. **One hypothesis at a time.** Change one thing; three simultaneous changes teach nothing.
5. **Root cause, not band-aid.** Understand *why* before writing the fix; a mask leaves the defect to resurface further from its cause. Trace data flow backward from the symptom, question your assumptions, ask "why" five times, gather more evidence when stuck.
6. **Study the codebase, not just the diff.** A function that looks broken may be correct in context — callers validate, middleware enforces invariants. Trace callers (via `golang-gopls` call resolution, which follows interface dispatch), check upstream validation, read the surrounding code. If context lowers severity but doesn't kill the issue, report it at reduced priority with the protecting guarantee noted inline.
7. **Start simple.** `fmt.Println` is a fine local tool; escalate when it stops answering. Never use it in production — `slog` there.

## Red flags — you are debugging wrong

- "Quick fix for now, investigate later" — there is no later.
- Multiple simultaneous changes.
- Proposing fixes without understanding the cause ("maybe add a nil check here…").
- Each fix reveals a new problem — you are treating symptoms.
- Three-plus attempts on the same issue — wrong mental model; re-read from scratch.
- "Works on my machine" — the environmental difference is unisolated.
- Blaming the compiler/framework — verify your code first; it is almost never Go.

## Reference files

- [`references/methodology.md`](./references/methodology.md) — the systematic 10-step process and the escalation ladder.
- [`references/common-go-bugs.md`](./references/common-go-bugs.md) — nil deref, typed-nil interface, shadowing, slice/map/defer/error/context traps, JSON surprises, unclosed resources — each with reproduction and fix.
- [`references/testing-debug.md`](./references/testing-debug.md) — failing test first, isolation, `-v`/`-run`/`-count=10`, flaky tests.
- [`references/concurrency-debug.md`](./references/concurrency-debug.md) — races, deadlocks, leaks; reading `-race` output; `goleak`; stack dumps.
- [`references/performance-debug.md`](./references/performance-debug.md) — CPU workflow, heap vs `alloc_objects`, mutex contention, flamegraphs.
- [`references/pprof.md`](./references/pprof.md) — enabling endpoints (with auth), profiles, capture local/remote, `top`/`list`/`web`.
- [`references/diagnostic-tools.md`](./references/diagnostic-tools.md) — GODEBUG, Delve, `-gcflags=-m` escape analysis, execution tracer.
- [`references/production-debug.md`](./references/production-debug.md) — debugging live systems: log shape, safe pprof exposure, tcpdump/netstat, request inspection.
- [`references/compilation.md`](./references/compilation.md) — module conflicts, cgo linking, toolchain/go.mod mismatch, build tags.
- [`references/code-review-flags.md`](./references/code-review-flags.md) — bug-prone patterns to catch in review.

## Cross-references

- `golang-performance` — patterns once the bottleneck is identified.
- `golang-observability` — metrics/alerting/dashboards for runtime health.
- `golang-concurrency`, `golang-safety`, `golang-error-handling` — the design rules whose violations you are hunting.
