# Execution Trace Reference

`go tool trace` shows what pprof cannot: scheduling delays, GC stop-the-world phases, goroutine state transitions, and *why* a goroutine is not running. pprof samples what is on-CPU; the tracer records every state transition at nanosecond precision.

Reach for it when pprof shows low CPU% but latency is high, when GC pauses are suspected of causing tail-latency spikes, or when you need the wall-clock timeline of concurrent work.

## Generating traces

```bash
# From a benchmark
go test -bench=BenchmarkParse -trace=trace.out ./pkg/parser
go tool trace trace.out

# From tests
go test -trace=trace.out ./pkg/parser

# From a running service (requires net/http/pprof import)
curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
go tool trace trace.out
```

Captures generate data at multiple MB/s — keep them 5–10 seconds long. Longer traces are slow to parse, heavy on memory, and sluggish in the browser.

```go
import "runtime/trace"

f, _ := os.Create("trace.out")
trace.Start(f)
defer trace.Stop()
```

## Command reference

```bash
# Open the viewer (default) or bind a specific address
go tool trace trace.out
go tool trace -http=:8080 trace.out
go tool trace -http=0.0.0.0:8080 trace.out

# Extract a pprof profile from the trace: net, sync, syscall, sched
go tool trace -pprof=net trace.out > net.prof
go tool trace -pprof=sync trace.out > sync.prof
go tool trace -pprof=syscall trace.out > syscall.prof
go tool trace -pprof=sched trace.out > sched.prof
```

| `-pprof` type | Blocking cause captured |
| --- | --- |
| `net` | Network I/O waits |
| `sync` | Mutex, channel, waitgroup waits |
| `syscall` | Blocking system calls |
| `sched` | Time between runnable and running (scheduling latency) |

Extracted profiles feed straight into pprof's tooling — `go tool pprof -top -cum sync.prof`, `-list=processOrder`, `-svg`.

Worked end-to-end:

```bash
go test -bench=BenchmarkParse -trace=trace.out ./pkg/parser
go tool trace trace.out                              # visual timeline
go tool trace -pprof=sync trace.out > sync.prof      # worst sync blockers
go tool pprof -top -cum sync.prof

curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=5
go tool trace -pprof=sched trace.out > sched.prof    # scheduling latency
go tool pprof -svg sched.prof > sched.svg

go test -trace=trace.out -run=TestSlowIntegration ./pkg/api
go tool trace -pprof=net trace.out > net.prof        # network wait sites
go tool pprof -top net.prof
```

The viewer serves `/` (index), `/trace` (interactive timeline), `/goroutines` (aggregated goroutine stats by creation stack — counts, execution, scheduling wait, blocking — click through to individual goroutines), and `/goroutine/<id>` (a single goroutine's lifecycle).

## Reading the timeline

The Chrome-style viewer: `W`/`S` zoom the time axis, `A`/`D` pan, click an event for its detail, `Shift+click` selects a time range, `M` marks it, `/` searches, `?` lists shortcuts.

| Lane color | State | Meaning |
| --- | --- | --- |
| Green | Running | Executing on a P — normal work |
| Blue | Syscall | Pinned to an OS thread |
| Yellow/orange | Runnable | Ready but waiting for a P — CPU saturation |
| Red/pink | Waiting | Blocked on I/O, channel, mutex, sleep, select |
| Blue bands | GC assist | Goroutine drafted to help mark/sweep |
| Red bands (all Ps) | STW | Stop-the-world pause |
| Light blue bands | Concurrent mark | GC mark phase |
| Purple | Region | User annotation via `trace.WithRegion` |

Gaps in a P lane mean idle — no runnable goroutines or all blocked. Idle gaps alongside a backlog of runnable goroutines indicate scheduling contention.

### GC phases

- **Mark assist** — the runtime taxes heavy allocators by forcing their goroutines to help scan; shows as gaps in application execution.
- **STW** — short all-goroutine stops (mark setup/termination) that spike latency as vertical bands.
- **Sweep** — concurrent reclamation of unreachable objects; typically cheap unless the heap is huge.

Frequent cycles with long mark assist mean allocation rate is the problem; long STW phases suggest too many pointers to scan; cycles clustering after one operation name its heavy allocator.

### Scheduling latency

Yellow (runnable) gaps before green (running) segments, many runnable goroutines at once, and uneven P utilization all point to scheduling problems: competing goroutines, noisy neighbors or throttling, or threads pinned by cgo/long syscalls.

### Blocking

Long red/pink stretches mean a blocked goroutine — click the event for the cause (channel receive, mutex, network read). Many goroutines on the same channel or mutex is a serialization bottleneck; network waits are external latency, captured as a `-pprof=net` profile.

### Lifecycle

Goroutines created in an unbounded loop, or created and never finishing, are leaks. Very short-lived goroutines created in bulk are overhead — consider batching or a worker pool.

## Custom annotations

`runtime/trace` adds application context that makes runtime events correlate with business operations. Annotations are near-free while tracing is off — they check a flag and return.

```go
ctx, task := trace.NewTask(ctx, "processOrder")   // a logical operation,
defer task.End()                                  // spanning goroutines

trace.WithRegion(ctx, "validateAddress", func() { // a named phase
    validateAddress(order.Address)
})

trace.Log(ctx, "orderID", order.ID)               // a point-in-time marker
```

Rules of thumb: wrap every request handler in a task; add regions to the phases you want wall-clock numbers for; add logs at decision points when chasing intermittent latency.

## Flight recorder (Go 1.25+)

The flight recorder fixes the timing problem for long-running services: by the time a timeout or failed health check surfaces, it is too late to start a trace. It keeps a circular buffer of trace data in memory and snapshots it on demand — an airplane black box.

```go
fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
    MinAge:   10 * time.Second, // keep at least 10s of data
    MaxBytes: 5 << 20,          // cap at 5 MiB
})
fr.Start()
```

Sizing: `MinAge` roughly 2× the problem window (10 s beats a 5 s timeout). Busy services emit ~1–10 MB/s, so start `MaxBytes` around 1–5 MiB; `MaxBytes` wins over `MinAge` when the buffer fills.

Snapshot on the failure, once — `sync.Once` prevents overwrites:

```go
var once sync.Once
func capture(fr *trace.FlightRecorder) {
    once.Do(func() {
        f, _ := os.Create("snapshot.trace")
        defer f.Close()
        if _, err := fr.WriteTo(f); err != nil {
            return
        }
        fr.Stop()
    })
}
```

Trigger patterns: a request slower than a threshold, a failed health check, or a dedicated `/debug/flightrecorder` endpoint that streams `fr.WriteTo(w)`. Analyze a snapshot exactly like any trace: `go tool trace snapshot.trace`.

Constraints: one flight recorder at a time (may change in future Go); it can run concurrently with `trace.Start`; `WriteTo` is single-goroutine; `Stop` blocks until an in-flight `WriteTo` finishes.

| Scenario | Choice |
| --- | --- |
| Known slow operation | `go test -trace` or `trace.Start`/`Stop` around it |
| Intermittent latency spikes | Flight recorder — retroactive capture |
| Post-mortem after timeout/crash | Flight recorder |
| Continuous monitoring | `fabianoflorentino/golang-agent-skills@golang-observability` (Pyroscope) |

## Practical limits

| Dimension | Guidance |
| --- | --- |
| CPU overhead | ~1–2% while capturing; negligible when idle |
| Data volume | MB/s; a 10 s busy trace can be 50–100 MB |
| Capture length | 5–10 s typical |
| Viewer memory | The whole trace loads into RAM; large traces want 1 GB+ |
| Browser | Struggle above ~100 MB — keep captures short |
| Production | Fine for short captures on one instance; never continuous |

## Trace vs pprof

| Question | Tool |
| --- | --- |
| Where does CPU time go? | pprof CPU profile |
| High latency, low CPU? | `go tool trace` — waiting states |
| Where do allocations happen? | pprof heap profile |
| Why are GC pauses long? | `go tool trace` — STW, mark assist timeline |
| Lock contention? | pprof mutex/block to quantify; trace for the timeline |
| Goroutine leak? | pprof for the stack; trace for creation/lifecycle |
| Which goroutines compete for CPU? | `go tool trace` |
| Wall-clock breakdown of one request? | `go tool trace` with annotations |

When in doubt, start with pprof — lower overhead, simpler output. Switch to trace when pprof cannot explain the latency or the timeline itself matters.