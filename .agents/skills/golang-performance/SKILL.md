---
name: golang-performance
description: "Golang performance optimization patterns and methodology - if X bottleneck, then apply Y. Covers allocation reduction, CPU efficiency, memory layout, GC tuning, pooling, caching, and hot-path optimization. Use when profiling or benchmarks have identified a bottleneck and you need the right optimization pattern to fix it. Also use when performing performance code review to suggest improvements or benchmarks that could help identify quick performance gains. Not for measurement methodology (→ See `fabianoflorentino/golang-agent-skills@golang-benchmark` skill) or debugging workflow (→ See `fabianoflorentino/golang-agent-skills@golang-troubleshooting` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🏎"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - benchstat
    install:
      - kind: go
        package: golang.org/x/perf/cmd/benchstat@latest
        bins: [benchstat]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch Bash(benchstat:*) Bash(fieldalignment:*) Bash(staticcheck:*) Bash(curl:*) Bash(fgprof:*) Bash(perf:*) WebSearch AskUserQuestion EnterWorktree ExitWorktree
paths:
  - "**/*.go"
---

# Performance optimization in Go

**Persona:** You are a Go performance engineer. Nothing is optimized without profiling first: measure, form one hypothesis, change one thing, re-measure. Intuition about bottlenecks is wrong roughly 80% of the time.

**Modes:**

- **Review (architecture)** — broad scan for structural anti-patterns: missing pools, unbounded goroutines, wrong structures. Fan out one sub-agent per concern — allocations/layout, I/O/concurrency, algorithmic complexity/caching — and merge. Parallel.
- **Review (hot path)** — focused analysis of one function or tight loop. Sequential; one sub-agent suffices.
- **Optimize** — a bottleneck is already identified by profiling. Follow the iterative cycle strictly, one change at a time. Sequential.

**When to use:** after a benchmark/profile has named a bottleneck and you need the *right fix*. Not for how to measure (`golang-benchmark`), not for the debugging workflow (`golang-troubleshooting`).

## Philosophy

1. **Profile before optimizing.** If pprof does not show a hotspot, "slow" is a guess.
2. **Allocations pay the biggest dividend.** Go's GC is fast but never free; cutting allocations per request usually beats micro-tuning the CPU.
3. **Document every optimization** — why the pattern is faster, and the benchmark numbers. Future readers must not "correct" a deliberate choice back into slowness.

## Rule out external bottlenecks first

If 90% of latency is a slow DB query or an upstream API, reducing allocations changes nothing. Diagnose where time actually sits:

1. `fgprof` — on-CPU and off-CPU (I/O wait) time; off-CPU dominance means the process is waiting, not working.
2. `go tool pprof` goroutine profile — many goroutines parked in `net.(*conn).Read` or `database/sql` = external wait.
3. Distributed tracing (OpenTelemetry) — the span breakdown names the slow upstream.

When external, fix the component: query tuning, caching, pools, circuit breakers (`golang-database`, the caching reference).

## The iterative cycle

1. **Define the metric** — latency, throughput, memory, CPU? Without a target, changes are random.
2. **Write an atomic benchmark** — one function per benchmark (`golang-benchmark`).
3. **Measure baseline** — `go test -bench=BenchmarkMyFunc -benchmem -count=6 ./pkg/... | tee /tmp/report-1.txt`.
4. **Diagnose** — use the "Diagnose" lines in the deep dives to pick the right tool.
5. **Improve** — ONE optimization at a time, with an explanatory comment.
6. **Compare** — `benchstat /tmp/report-1.txt /tmp/report-2.txt` — act only on significant differences.
7. **Commit** — benchstat table in the body, `perf(scope):` type, so reviewers and future readers see the exact gain.
8. **Repeat** — bump the report number, take the next hotspot.

Competing hypotheses for the same bottleneck: implement each in its own worktree via a separate sub-agent, then compare through `golang-benchmark` — and remember its serial-measurement caveat: implementations can be built in parallel, benchmarks never should be (shared CPU contaminates results). Keep the `/tmp/report-*.txt` trail.

## Decision tree: where is time spent?

| Bottleneck | pprof signal | Action |
| --- | --- | --- |
| Allocations | `alloc_objects` hot in heap profile | [memory.md](./references/memory.md) |
| CPU-bound hot loop | one function dominates CPU | [cpu.md](./references/cpu.md) |
| GC pauses / OOM | high GC%, container limits | [runtime.md](./references/runtime.md) |
| I/O latency | goroutines blocked on I/O | [io-networking.md](./references/io-networking.md) |
| Repeated work | same compute/fetch over and over | [caching.md](./references/caching.md) |
| Wrong algorithm | O(n²) where O(n) exists | caching.md → algorithmic complexity |
| Lock contention | mutex/block profile hot | `golang-concurrency` |
| Slow queries | DB time dominates traces | `golang-database` |

## Common mistakes

| Mistake | Fix |
| --- | --- |
| Optimizing without profiling | pprof first — intuition ~80% wrong |
| Default `http.Client` | `MaxIdleConnsPerHost` defaults to 2 — set it to match concurrency |
| Logging in hot loops | Log calls block inlining and allocate even when the level is off — `slog.LogAttrs` |
| `panic`/`recover` as control flow | allocates a stack trace; return errors |
| `unsafe` without proof | only when profiling shows >10% in a verified hot path |
| No GC bounds in containers | `GOMEMLIMIT` at 80–90% of container memory avoids OOM kills |
| `reflect.DeepEqual` in production | 50–200x slower than typed; `slices.Equal`, `maps.Equal`, `bytes.Equal` |

## Deep dives

- [`references/memory.md`](./references/memory.md) — allocation patterns, backing-array leaks, `sync.Pool`, struct alignment.
- [`references/cpu.md`](./references/cpu.md) — inlining, cache locality, false sharing, ILP, reflection avoidance.
- [`references/io-networking.md`](./references/io-networking.md) — HTTP transport config, streaming, JSON perf, cgo, batching.
- [`references/runtime.md`](./references/runtime.md) — `GOGC`, `GOMEMLIMIT`, GC diagnostics, `GOMAXPROCS`, PGO.
- [`references/caching.md`](./references/caching.md) — algorithmic complexity, compiled patterns, `singleflight`, work avoidance.
- [`references/observability.md`](./references/observability.md) — production metrics and continuous profiling to keep the gains.

## CI regression detection

Gate on benchmark drift before it ships — `benchdiff` and `cob` wiring in `golang-benchmark`.

## Cross-references

- `golang-benchmark` — methodology, `benchstat`, `b.Loop()`.
- `golang-troubleshooting` — pprof workflow, escape-analysis diagnostics.
- `golang-data-structures` — preallocation, `strings.Builder`.
- `golang-concurrency` — worker pools, `sync.Pool`, lock contention.
- `golang-safety` — `defer` in loops, backing-array aliasing.
- `golang-database` — pool tuning, batching.
- `golang-observability` — continuous profiling in production.
