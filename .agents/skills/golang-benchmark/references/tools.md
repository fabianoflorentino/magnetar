# Diagnostic Tools Quick Reference

Validate the root cause of a slowdown *before* optimizing. Do not let tools auto-apply fixes — interpret the output and change the code manually with explanatory comments.

Details for the profile- and trace-oriented tools live in their own files:

- [`pprof.md`](./pprof.md) — CPU, heap, goroutine, mutex, block profiles.
- [`benchstat.md`](./benchstat.md) — statistical benchmark comparison.
- [`trace.md`](./trace.md) — execution tracer.
- [`compiler-analysis.md`](./compiler-analysis.md) — escape analysis, inlining, SSA, assembly.

## Runtime diagnostics

Runtime behavior is configured through `GODEBUG` environment variables — no recompile needed.

| Command | Use for |
| --- | --- |
| `GODEBUG=gctrace=1 ./app` | One line per GC cycle: frequency, pause times, heap sizes, CPU |
| `GODEBUG=gcpacertrace=1 ./app` | Why GC triggers when it does — pacer decisions, trigger ratio, heap goal |
| `GODEBUG=schedtrace=1000 ./app` | Load balance and goroutine spread across Ps, printed every 1000 ms |
| `GODEBUG=schedtrace=1000,scheddetail=1 ./app` | Schedtrace plus per-goroutine state detail |
| `go tool pprof -alloc_objects` (heap profile) | Allocation sites and object churn — the live source for allocation-site data |

See `fabianoflorentino/golang-agent-skills@golang-troubleshooting` for full GODEBUG interpretation.

### In-process APIs

| API | What it gives you | Use for |
| --- | --- | --- |
| `runtime.ReadMemStats` | Heap size, `NumGC`, pause circular buffer, cumulative `TotalAlloc` | Dashboards and heap-growth alerting |
| `debug.ReadGCStats` | GC pause percentiles, timeline, total pause | More focused than ReadMemStats |
| `runtime/metrics` (Go 1.16+) | Stable, concurrency-safe, low-overhead namespaced metrics | Production instrumentation: `/gc/cycles/total:gc-cycles`, `/gc/heap/allocs:bytes`, `/gc/pauses:seconds`, `/sched/latencies:seconds`, `/memory/classes/heap/released:bytes` |
| `debug.FreeOSMemory()` | Forced GC returning memory to the OS | One-off after large temporary allocations; not for steady state |
| `expvar` | Stdlib metrics at `/debug/vars` as JSON, via `import _ "expvar"` | Lightweight dependency-free metrics for Netdata, Telegraf, custom dashboards |

## Static analysis

| Command | Use for |
| --- | --- |
| `fieldalignment ./...` | Struct field padding waste. Do not use `-fix` — apply reorderings manually with comments |
| `unsafe.Sizeof` / `Alignof` / `Offsetof` | Struct memory layout at compile time; quantify a reorder |
| `go vet ./...` | Suspicious constructs: printf mismatches, unreachable code, misused results, shifts |
| `staticcheck ./...` | Deeper checks: empty branches (SA9003), unused values (SA4006), deprecated APIs (SA1019) |
| `go test -race ./...` | Data-race detection at runtime; doubles as a false-sharing check |

## Third-party profilers

| Tool | Adds | When |
| --- | --- | --- |
| `fgprof` (`github.com/felixge/fgprof`) | Whole-process profile of on-CPU *and* off-CPU (I/O wait) time in one capture | pprof CPU shows low CPU% while latency stays high |
| Pyroscope / Parca | Continuous profiling, aggregates pprof profiles over time, cross-deployment comparison | Production monitoring and trend analysis; setup in `fabianoflorentino/golang-agent-skills@golang-observability` |
| Linux `perf` (`perf record -g ./app && perf report`) | Hardware counters: cache misses, branch mispredictions, TLB misses; `perf_data_converter` bridges to pprof format | Microarchitecture analysis when pprof is too coarse |