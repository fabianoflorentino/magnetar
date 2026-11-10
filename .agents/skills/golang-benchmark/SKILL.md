---
name: golang-benchmark
description: "Golang benchmarking, profiling, and performance measurement. Use when writing, running, or comparing Go benchmarks, profiling hot paths with pprof, interpreting CPU/memory/trace profiles, analyzing results with benchstat, setting up CI benchmark regression detection, or investigating production performance with Prometheus runtime metrics. Also use when the developer needs deep analysis on a specific performance indicator - this skill provides the measurement methodology, while `fabianoflorentino/golang-agent-skills@golang-performance` provides the optimization patterns."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📊"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - benchstat
    install:
      - kind: go
        package: golang.org/x/perf/cmd/benchstat@latest
        bins: [benchstat]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch Bash(benchstat:*) Bash(benchdiff:*) Bash(cob:*) Bash(gobenchdata:*) Bash(curl:*) mcp__context7__resolve-library-id mcp__context7__query-docs WebSearch AskUserQuestion EnterWorktree ExitWorktree
paths:
  - "**/*.go"
---

# Benchmarking & performance measurement

**Persona:** You are a Go performance measurement engineer. No optimization decision is made from a single benchmark run — statistical rigor and controlled conditions are prerequisites. If it can be measured, it can be improved; if it cannot, the change is guesswork.

**Modes:**

- **Write** — author benchmarks for new code, decide `b.Loop()` vs legacy `b.N`, memory tracking, sub-benchmarks. Sequential.
- **Run/Compare** — execute variant sets, apply `benchstat`, keep the stat-significant winner. Sequential (measure never in parallel).
- **Profile** — hunt hotspots via `pprof` (CPU, mem, trace) and interpret the graphs. Sequential.
- **CI gate** — wire regression detection (benchdiff/cob/gobenchdata) into pipelines. Sequential.

**When to use:** any task that measures or optimizes performance. Apply `golang-performance` for the optimization patterns once the measurement identifies a bottleneck; use `golang-troubleshooting` for pprof on live services; use `golang-observability` for always-on production profiling.

## Write a benchmark

Benchmark functions live in a `_bench_test.go` file named after the **source** file under benchmark (`parser.go` → `parser_bench_test.go`), never per function. This keeps `go test -bench=.` output free of unrelated `Test*` noise, separates measurement fixtures from correctness fixtures, and preserves Go's one-file-per-source convention from `golang-testing`. Order the `Benchmark*` functions to mirror the source functions they measure.

**Go 1.24+ — prefer `b.Loop()`.** It times only the loop body and keeps arguments/results alive, removing dead-code-elimination mistakes. Legacy `b.N` loops still compile and remain the right form when preserving existing benchmarks or targeting Go <1.24.

```go
func BenchmarkParse(b *testing.B) {
    data := loadFixture("large.json") // setup excluded from timing
    for b.Loop() {
        Parse(data)
    }
}
```

Memory tracking with `b.ReportAllocs()` (or the `-benchmem` flag); custom units via `b.ReportMetric`:

```go
b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds(), "bytes/s")
```

Sub-benchmarks keep a variant matrix in one function with named cases:

```go
func BenchmarkEncode(b *testing.B) {
    for _, size := range []int{64, 256, 4096} {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := make([]byte, size)
            for b.Loop() { Encode(data) }
        })
    }
}
```

## Run with statistical teeth

```bash
go test -bench=BenchmarkEncode -benchmem -count=10 ./pkg/... | tee bench.txt
```

| Flag | Purpose |
| --- | --- |
| `-bench=.` | Run all (regexp filter) |
| `-benchmem` | Report B/op, allocs/op |
| `-count=10` | Repeated runs for significance |
| `-benchtime=3s` | Min time per benchmark |
| `-cpu=1,2,4` | Sweep GOMAXPROCS |
| `-cpuprofile` / `-memprofile` / `-trace` | Capture profiles from the run |

Output format: `BenchmarkEncode/size=64-8  5000000  230.5 ns/op  128 B/op  2 allocs/op` — `-8` is GOMAXPROCS, `ns/op` time per op, `B/op` and `allocs/op` the memory picture.

## Toolchain honesty

A `benchstat` comparison that straddles a Go toolchain boundary measures the compiler, not your code. Go 1.27's size-specialized allocator, for example, changes allocation-heavy baselines on its own. Rerun the "before" side on the same toolchain as "after" before trusting any delta. (Go 1.26 also fixed a `b.Loop()` inlining limitation; 1.24–1.25 runs are correct but may miss that inlining.)

## Comparing variants

Isolate each competing hypothesis in its own worktree via a separate sub-agent so code never collides. **Measure serially** — concurrent benchmark runs share the CPU and the noisy-neighbor effect reintroduces the very noise `-count` and `benchstat` exist to remove. Compare every variant against the same baseline report; keep the winner, delete the rest.

## benchstat

`benchstat old.txt new.txt` computes confidence intervals and a `p`-value; a `~` in the difference column means no significance and the claim cannot be made. Use `-filter` to drop unrelated benchmarks and `-table` to realign rows. Full flags in the benchstat reference.

## Document claims in the commit

Paste the benchstat table into the commit body when a change carries a measurable impact:

```
perf(parser): reduce Parse allocations 50% with sync.Pool

    sec/op:   4.592µ ± 2% → 3.041µ ± 1%  (-33.78%, p=0.000 n=10)
    B/op:     1.024Ki ± 0% → 0.512Ki ± 0% (-50.00%, p=0.000 n=10)
    allocs/op: 12.00 ± 0% → 6.000 ± 0%  (-50.00%, p=0.000 n=10)

goos: linux / goarch: amd64 / cpu: AMD Ryzen 9 5950X
```

Rules: strip unrelated rows, never paste `~` results, include the hardware line, use the `perf(scope):` type.

## Profiling a hotspot

```bash
go test -bench=BenchmarkParse -cpuprofile=cpu.prof ./pkg/parser
go tool pprof cpu.prof                 # then: top, list Parse, web
go test -bench=. -memprofile=mem.prof ./pkg/parser
go tool pprof -alloc_objects mem.prof  # GC churn vs inuse_space (leaks)
go test -bench=. -trace=trace.out ./pkg/parser
go tool trace trace.out                # when +where matters
```

Interpret: CPU profile shows where time went; `-alloc_objects` shows allocation churn; `-inuse_space` shows live memory. The trace answers *when* and *why* — scheduling, GC phases, blocking. For live HTTP services, enable pprof through env toggles instead (`golang-troubleshooting`, `golang-observability`).

## Reference files

- [`references/pprof.md`](./references/pprof.md) — full CLI, profile types, interactive and non-interactive analysis, `web` UI.
- [`references/benchstat.md`](./references/benchstat.md) — filtering, interleaving old/new, regression detection.
- [`references/trace.md`](./references/trace.md) — goroutine scheduling, GC phases, span annotations.
- [`references/tools.md`](./references/tools.md) — fieldalignment, GODEBUG, fgprof, race detector.
- [`references/compiler-analysis.md`](./references/compiler-analysis.md) — escape analysis, inlining, SSA, `go tool asm`.
- [`references/ci-regression.md`](./references/ci-regression.md) — benchdiff, cob, gobenchdata, noisy-neighbor mitigation in CI.
- [`references/investigation-session.md`](./references/investigation-session.md) — production troubleshooting with Prometheus runtime metrics.
- [`references/prometheus-go-metrics.md`](./references/prometheus-go-metrics.md) — the Go runtime metrics `client_golang` exposes, PromQL examples.

## Cross-references

- `golang-performance` — optimization patterns keyed to the measured bottleneck.
- `golang-troubleshooting` — pprof on running services, Delve, GODEBUG, diagnosis workflow.
- `golang-observability` — always-on profiling (Pyroscope), distributed tracing.
- `golang-continuous-integration` — wiring benchmark regression gates into CI.
- `golang-testing` — suite hygiene; benchmark files follow the same one-file-per-source rule.
