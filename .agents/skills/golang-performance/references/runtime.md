# Runtime Tuning

Runtime knobs control GC frequency, memory targets, CPU visibility, and compiler optimization. The defaults suit most workloads — tune only after profiling names a problem.

## Garbage collector tuning

**Diagnose first:** `GODEBUG=gctrace=1` prints one line per cycle (watch cycles/s, CPU%, pause length); `runtime/metrics` and `debug.ReadGCStats` give pause percentiles for dashboards; in production `rate(go_gc_duration_seconds_count[5m])` tracks GC frequency.

**GOGC** — the heap-growth ratio that triggers collection. `GOGC=100` (default) collects when live heap doubles since the last pass. Lower = more frequent, shorter pauses; higher = rarer collections, more memory.

```bash
GOGC=50  ./app   # latency-sensitive: frequent, short pauses
GOGC=200 ./app   # throughput: fewer stops, more memory
GOGC=off ./app   # testing only
```

**GOMEMLIMIT** (Go 1.19+) — soft ceiling: the pacer escalates GC effort to stay under it. In a container, set it around 80-90% of the cap so the runtime collects *before* the OOM killer does, leaving headroom for goroutine stacks and non-heap memory:

```bash
GOMEMLIMIT=900MiB ./app   # under a 1GiB container limit
```

**Programmatic control** mirrors the environment variables for dynamic tuning:

```go
debug.SetGCPercent(200)                    // equivalent to GOGC=200
debug.SetMemoryLimit(450 * 1024 * 1024)    // 450 MiB soft limit
```

**Ballast pattern** — pre-1.19 teams seeded a giant `[1 << 30]byte` at startup to inflate live heap and space out GC cycles. GOMEMLIMIT delivers the same effect without wasting RAM; don't add new ballasts.

## GC diagnostics

`GODEBUG=gctrace=1` sample:

```
gc 5 @1.234s 2%: 0.012+12+0.9 ms clock, 45->92->50 MB, 200 MB goal, 8 P
```

- `5` — cycle number; `@1.234s` — elapsed since start; `2%` — CPU spent in GC.
- `45->92->50 MB` — heap before, peak during, after; `200 MB goal` — target; `8 P` — schedulers.

Watch: cycles/s too high = allocation rate too high; pauses growing = larger heap or more pointers; % CPU high = tune GOGC *or* cut allocations.

Programmatic monitoring:

```go
var m runtime.MemStats
runtime.ReadMemStats(&m)
// m.Alloc (live), m.TotalAlloc (cumulative), m.Sys (requested from OS),
// m.NumGC, m.PauseNs — feed dashboards and alerting
```

**The pacer** schedules collections from four inputs: live heap after the last pass, GOGC percentage, GOMEMLIMIT (if set), and current allocation rate. Faster allocation means earlier starts.

## Allocation rate reduction

Lowering the allocation rate beats raising GOGC — it fixes the cause, not the symptom:

- Value types over pointers where possible (values can stay on the stack).
- Pool hot allocations with `sync.Pool` (see [memory.md](./memory.md)).
- Preallocate and size-hint slices and maps.
- Typed parameters and generics over `any` to avoid boxing.

## GOMAXPROCS in containers

**Go 1.25+** detects container limits on Linux (cgroup v2 CPU quota, process affinity mask), so `GOMAXPROCS` matches the container instead of the host. On cgroup v1, validate the detected value at startup — known gaps exist (e.g. Oracle OCPUs) where a manual `GOMAXPROCS` is a fine workaround.

**Go 1.24 and earlier** need the library:

```go
import _ "go.uber.org/automaxprocs" // sizes GOMAXPROCS to the container limit
```

Manual overrides: `GOMAXPROCS=2 ./app`, or `GODEBUG=updatemaxprocs=0 ./app` to freeze dynamic updates on Go 1.25+.

Verify with `runtime.GOMAXPROCS(0)` at startup; if it reports the host's core count while the container has 2, over-scheduling explains your latency spikes.

## Profile-guided optimization (PGO)

Go 1.21+ can build using a production CPU profile for better inlining and devirtualization — typically 2-7% for near-zero effort.

1. Capture a representative profile (30-60s):

   ```bash
   curl 'http://localhost:6060/debug/pprof/profile?seconds=60' > cpu.pprof
   ```

2. Place it as `default.pgo` in the main package directory.
3. `go build` auto-detects it.

Helps most on interface-heavy code with hot call stacks; helps least on already-tuned or memory-bound code. Refresh the profile after significant changes — stale profiles mislead the compiler.

## Logging overhead in hot paths

Log formatting runs even when the level would discard the message: `logger.Debug(fmt.Sprintf(...))` formats before `Debug` is consulted. With slog the level check comes first and typed args are cheap to build:

```go
slog.LogAttrs(ctx, slog.LevelDebug, "processing item",
    slog.Int("id", item.ID)) // ~zero allocation when the level is off
```

Prefer typed attributes (`slog.Int`, `slog.String`, `slog.Bool`); `slog.Any` can still allocate.

## Panic/recover cost

`panic` unwinds the stack through every deferred function — benchmarked 10-100x more expensive than returning an error. Never use it for control flow:

```go
// don't lean on strconv panicking for invalid input
v, err := strconv.Atoi(s)
if err != nil {
    continue
}
```

Reserve panic for genuinely unrecoverable program state, and convert to errors at package boundaries.
