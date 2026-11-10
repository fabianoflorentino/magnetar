# pprof Reference

`go tool pprof` answers where CPU time goes, where memory is spent or churned, and where goroutines contend. This reference covers generating, navigating, and interpreting profiles. Exposing pprof endpoints on a running service (`net/http/pprof`, authn/authz, security) is covered by `fabianoflorentino/golang-agent-skills@golang-troubleshooting`.

## Profile types

Pick the profile that matches the symptom; the wrong one wastes the session.

| Profile | Flag / endpoint | Use when | Notes |
| --- | --- | --- | --- |
| CPU | `-cpuprofile` or `/debug/pprof/profile?seconds=30` | High CPU, slow functions | 100 Hz sampling of on-CPU functions; misses off-CPU time (I/O, sleep) |
| Heap — `alloc_objects` | `-memprofile`, then `pprof -alloc_objects` | GC pressure from allocation frequency | Counts allocation events, not bytes |
| Heap — `alloc_space` | `pprof -alloc_space` | Biggest allocation sites by volume | Total bytes allocated |
| Heap — `inuse_space` | `pprof -inuse_space` | Memory growing over time, leaks | Live heap snapshot |
| Heap — `inuse_objects` | `pprof -inuse_objects` | Many-small-object leaks | Count of live objects |
| Goroutine | `/debug/pprof/goroutine` | Blocked I/O, leaks, pool exhaustion | All goroutine stacks |
| Mutex | `/debug/pprof/mutex` | Lock contention | Cumulative wait on mutexes; must be enabled first |
| Block | `/debug/pprof/block` | Blocked on channels, timers, select | Cumulative time in sync waits; must be enabled first |
| Threadcreate | `/debug/pprof/threadcreate` | Excessive OS-thread creation | Thread-creation stacks (cgo, blocking syscalls pin threads) |

`alloc_objects` vs `alloc_space`: frequency and churn drive GC work, byte volume drives peak memory. Start with `alloc_objects` — churn is the common allocation bottleneck. `alloc_space` is cumulative since start and includes already-freed objects; `inuse_space` is the current live set — use it for leaks.

### Enabling mutex and block profiles

Both are off by default because they cost overhead:

```go
import "runtime"

runtime.SetMutexProfileFraction(5) // 1-in-5 contention events; 0 disables
runtime.SetBlockProfileRate(1)     // all blocking events; set a nanosecond
                                   // rate to sample; 0 disables
```

`SetBlockProfileRate(1)` records everything; larger values sample roughly one event per rate nanoseconds blocked (e.g. `1000000` ≈ 1 ms) — a production-friendly setting. Set both back to 0 after the investigation.

## Generating profiles

### From benchmarks — no server required

```bash
go test -bench=BenchmarkParse -cpuprofile=cpu.prof ./pkg/parser
go test -bench=BenchmarkParse -memprofile=mem.prof ./pkg/parser

# Both at once: CPU profiling adds ~5% overhead that skews memory numbers
go test -bench=BenchmarkParse -cpuprofile=cpu.prof -memprofile=mem.prof ./pkg/parser
```

### From a running service

Requires `import _ "net/http/pprof"`:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
go tool pprof -alloc_objects http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/goroutine
go tool pprof http://localhost:6060/debug/pprof/mutex
go tool pprof http://localhost:6060/debug/pprof/block
```

### Programmatic capture

```go
import "runtime/pprof"

f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()

h, _ := os.Create("heap.prof")
pprof.WriteHeapProfile(h)

pprof.Lookup("goroutine").WriteTo(f2, 0)
```

## Interactive commands

`go tool pprof cpu.prof` drops into a prompt. Everything below also works as a one-shot flag (`go tool pprof -top cpu.prof`); inside the prompt, drop the flag prefix.

### `top` — where to start

Ranks functions by self (flat) cost:

```
(pprof) top
Showing nodes accounting for 4.2s, 84% of 5s total
      flat  flat%   sum%        cum   cum%
     1.50s 30.00% 30.00%      2.80s 56.00%  encoding/json.Marshal
     0.80s 16.00% 46.00%      0.80s 16.00%  runtime.mallocgc
     0.60s 12.00% 58.00%      0.60s 12.00%  runtime.memmove
```

| Column | Meaning |
| --- | --- |
| `flat` / `flat%` | Time in the function itself, excluding callees |
| `sum%` | Running total of `flat%` down the list |
| `cum` / `cum%` | The function plus everything it calls |

Variants: `top 5` limits the rows; `top -cum 10` ranks by cumulative; `top -flat 20` by flat.

### `top -cum` — follow the responsibility chain

When `top` shows `runtime.mallocgc`, `runtime.memmove`, or `runtime.scanobject` on top, those are symptoms. Cumulative ranking names the application functions that pull them in:

```
(pprof) top -cum
     0.40s  8.00%  8.00%      3.80s 76.00%  myapp/pkg/handler.HandleRequest
     0.10s  2.00% 10.00%      2.80s 56.00%  myapp/pkg/handler.serializeResponse
     1.50s 30.00% 40.00%      2.80s 56.00%  encoding/json.Marshal
```

The path `HandleRequest → serializeResponse → json.Marshal` is the hot path; the optimization target is `serializeResponse`, not the runtime.

### `list` — annotated source

Per-line flat (left) and cumulative (right) cost; a regex matches several functions (`list Parse.*`, `list \.Handle`):

```
(pprof) list serializeResponse
Total: 5s
ROUTINE ======================== myapp/pkg/handler.serializeResponse
     0.10s      2.80s (flat, cum) 56.00% of Total
         .          .     38:func serializeResponse(w http.ResponseWriter, data any) {
         .      0.20s     39:    w.Header().Set("Content-Type", "application/json")
     0.10s      2.60s     40:    buf, err := json.Marshal(data)
```

### `peek` — one-hop call neighborhood

Callers above the divider, callees below. Answers whether a hot function is over-called upstream or slow downstream:

```
(pprof) peek json.Marshal
----------------------------------------------+-------------
                                               |  flat  flat%  sum%    cum  cum%
 myapp/pkg/handler.serializeResponse 2.60s     |
 myapp/pkg/api.buildResponse         0.20s     |  1.50s 30.00% 30.00%  2.80s 56.00%  encoding/json.Marshal
----------------------------------------------+-------------
 reflect.Value.MapRange              0.40s     |
 encoding/json.(*encodeState).marshal 0.30s    |
 runtime.mallocgc                     0.80s    |
```

### `tree` — full call tree

```
(pprof) tree
     0.40s  8.00%  8.00%      3.80s 76.00%  myapp/pkg/handler.HandleRequest
              0.10s  myapp/pkg/handler.serializeResponse
                     1.50s  encoding/json.Marshal
                            0.80s  runtime.mallocgc
```

### `traces` — raw stacks

Every sampled stack, unfiltered — useful for spotting call paths you did not expect.

### `web` / `svg` — call graph

Requires graphviz. Thicker edges carry more time; larger and redder nodes are hotter. `svg` writes a file for later inspection where text fails to reveal the shape.

### `disasm` — instruction-level cost

Generated assembly with per-instruction samples — verifies SIMD, surviving bounds checks, and inlined calls at the instruction level.

### `weblist` — browser `list`

The annotated source in a browser with per-line heat shading; falls back to `list` without a browser.

### `tags` and `tagroot`/`tagleaf` — profile labels

Profiles carry labels added via `pprof.Do`; `tags` shows their distribution, `tagroot` and `tagleaf` regroup the tree by them:

```go
labels := pprof.Labels("request_type", "api", "endpoint", "/users")
pprof.Do(ctx, labels, func(ctx context.Context) { handleRequest(ctx) })
```

```
(pprof) tags
request_type: api (85%), batch (15%)
```

```
(pprof) tagroot request_type   # roots the tree on the tag
(pprof) top                    # now per request_type
(pprof) tagleaf endpoint       # leaf-group by another tag
```

### Grouping, sorting, and display

```
(pprof) granularity=functions    # default
(pprof) granularity=filefunctions
(pprof) granularity=files
(pprof) granularity=lines        # reveals hot lines inside one function
(pprof) granularity=addresses
(pprof) sort=flat                # default for top
(pprof) sort=cum
(pprof) source handler           # annotated source for every match
(pprof) unit=ms
(pprof) unit=seconds
(pprof) unit=MB
```

### Filtering

Stateful filters persist until cleared.

| Command | Effect |
| --- | --- |
| `focus=regexp` | Keep only call paths touching a match; everything else dropped |
| `ignore=regexp` | Remove matches; their cost is attributed to callers |
| `show=regexp` | Display matches only; accounting unchanged |
| `hide=regexp` | Hide matches from display; accounting unchanged |
| `tagfocus=key=value` | Keep only samples carrying that tag |
| `tagignore=key=value` | Drop samples carrying that tag |
| `reset` | Clear all filters |

`show_from=regexp` trims every frame above the first match — good for cutting framework routing noise; `noinlines` attributes inlined functions to the first out-of-line caller.

### Multi-metric heap profiles

Heap profiles carry all four views; switch without reloading:

```
(pprof) sample_index=alloc_objects   # then top for counts
(pprof) sample_index=inuse_space     # then top for live memory
```

### Comparison and export

```
(pprof) normalize       # scale the base profile to the main total (with -base)
(pprof) callgrind       # export for KCachegrind/QCachegrind
(pprof) proto > filtered.pb.gz   # save the filtered profile
(pprof) help [command]
```

## Non-interactive equivalents

Every interactive command has a one-shot form; the profile path is omitted inside the prompt.

**Reporting**

```bash
go tool pprof -top cpu.prof
go tool pprof -cum -top -nodecount=20 cpu.prof
go tool pprof -list=json.Marshal cpu.prof
go tool pprof -peek=serializeResponse cpu.prof
go tool pprof -tree cpu.prof
go tool pprof -traces cpu.prof
go tool pprof -disasm=Parse cpu.prof
go tool pprof -source='handler\..*' cpu.prof
go tool pprof -text cpu.prof
```

**Graphs and exports**

```bash
go tool pprof -svg cpu.prof > cpu.svg
go tool pprof -svg -focus=handler cpu.prof > handler.svg
go tool pprof -pdf cpu.prof > cpu.pdf
go tool pprof -png cpu.prof > cpu.png
go tool pprof -gif cpu.prof > cpu.gif
go tool pprof -dot cpu.prof > cpu.dot
go tool pprof -callgrind cpu.prof > cpu.callgrind
go tool pprof -proto -focus=handler cpu.prof > handler-only.pb.gz
go tool pprof -weblist=serializeResponse cpu.prof
```

**Filtering and grouping**

```bash
go tool pprof -focus=myapp/pkg/handler -top cpu.prof
go tool pprof -ignore=runtime -top cpu.prof
go tool pprof -show=handler -top cpu.prof
go tool pprof -hide=testing -svg cpu.prof > clean.svg
go tool pprof -show_from=handler.Handle -top cpu.prof
go tool pprof -noinlines -top cpu.prof
go tool pprof -cum -top -nodecount=10 -focus=handler -ignore=runtime cpu.prof
```

**Tag filters** (labels via `pprof.Do`)

```bash
go tool pprof -tags cpu.prof
go tool pprof -tagfocus=endpoint=/users -top cpu.prof
go tool pprof -tagignore=request_type=batch -top cpu.prof
go tool pprof -tagroot=request_type -top cpu.prof
go tool pprof -tagleaf=endpoint -top cpu.prof
go tool pprof -tagshow=endpoint -svg cpu.prof > tagged.svg
go tool pprof -taghide=thread_id -svg cpu.prof > clean.svg
```

**Granularity and display**

```bash
go tool pprof -granularity=lines -top cpu.prof
go tool pprof -granularity=filefunctions -top cpu.prof
go tool pprof -granularity=files -top cpu.prof
go tool pprof -granularity=addresses -top cpu.prof
go tool pprof -unit=ms -top cpu.prof
go tool pprof -edgefraction=0.01 -nodefraction=0.005 -svg cpu.prof > clean.svg
go tool pprof -trim=false -svg cpu.prof > full.svg
```

**Heap profiles**

```bash
go tool pprof -top -alloc_objects mem.prof    # churn, GC pressure
go tool pprof -top -alloc_space mem.prof      # peak volume
go tool pprof -top -inuse_space mem.prof      # live memory, leaks
go tool pprof -top -inuse_objects mem.prof    # many small live objects
go tool pprof -alloc_objects -list=Parse mem.prof
go tool pprof -alloc_objects -svg mem.prof > allocs.svg
go tool pprof -top -base heap-baseline.prof heap-after.prof
go tool pprof -normalize -top -base heap-baseline.prof heap-after.prof
go tool pprof -base heap-baseline.prof -svg heap-after.prof > leak.svg
go tool pprof -base heap-baseline.prof -list=handleRequest heap-after.prof
```

**Profiles from a running service**

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
go tool pprof -svg "http://localhost:6060/debug/pprof/profile?seconds=10" > cpu.svg
go tool pprof -timeout=60 "http://localhost:6060/debug/pprof/profile?seconds=30"
go tool pprof -top -alloc_objects http://localhost:6060/debug/pprof/heap
go tool pprof -top http://localhost:6060/debug/pprof/goroutine
go tool pprof -top http://localhost:6060/debug/pprof/mutex
go tool pprof -top http://localhost:6060/debug/pprof/block

# Raw dumps without pprof
curl -o heap.prof http://localhost:6060/debug/pprof/heap
curl "http://localhost:6060/debug/pprof/goroutine?debug=0"   # text-ish stacks
curl "http://localhost:6060/debug/pprof/goroutine?debug=1"   # + creation sites/labels
curl "http://localhost:6060/debug/pprof/heap?debug=1"

# TLS
go tool pprof -tls_cert=client.crt -tls_key=client.key -tls_ca=ca.crt https://myservice:6060/debug/pprof/profile?seconds=30
go tool pprof https+insecure://myservice:6060/debug/pprof/profile?seconds=30
```

**Diffs**

```bash
go tool pprof -base cpu-before.prof cpu-after.prof     # subtract base; deltas
go tool pprof -diff_base=cpu-before.prof cpu-after.prof # percentages vs base
go tool pprof -normalize -base heap-before.prof heap-after.prof
go tool pprof -top -base cpu-before.prof cpu-after.prof
go tool pprof -svg -base cpu-before.prof cpu-after.prof > diff.svg
```

**Web UI**

```bash
go tool pprof -http=:8080 cpu.prof
go tool pprof -http=:9090 mem.prof
go tool pprof -http=:8080 -alloc_objects mem.prof
go tool pprof -http=:8080 -focus=handler cpu.prof
go tool pprof -http=:8080 -base heap-baseline.prof heap-after.prof
go tool pprof -http=:8080 -no_browser cpu.prof
```

**Symbolization**

```bash
go tool pprof -symbolize=none cpu.prof               # raw addresses
go tool pprof -symbolize=local cpu.prof              # local binaries only
go tool pprof -symbolize=remote "http://localhost:6060/debug/pprof/profile?seconds=10"
go tool pprof -symbolize=demangle=none cpu.prof      # mangled cgo names
go tool pprof -symbolize=demangle=full cpu.prof
```

**Environment variables**

| Variable | Purpose |
| --- | --- |
| `PPROF_BINARY_PATH` | Where to find binaries for symbolization; default `$HOME/pprof/binaries` — set it when profiling remote hosts |
| `PPROF_TOOLS` | Directory with binutils helpers (`addr2line`, `nm`, `objdump`) when not in `$PATH` |

## Web UI

`-http=:8080` opens an interactive viewer with a **flamegraph** (width ∝ cost, click to zoom), a weighted **call graph**, a sortable **top** view, browsable **source** annotations, **disassembly**, and **peek**. Prefer the CLI for quick diagnosis; use the UI when exploring an unfamiliar graph, diffing visually, or presenting findings.

## Comparing profiles

**Leak hunting** — take two heap snapshots and diff what grew:

```bash
curl http://localhost:6060/debug/pprof/heap > heap-baseline.prof
# ...let the leak accumulate (minutes to hours)...
curl http://localhost:6060/debug/pprof/heap > heap-after.prof
go tool pprof -base heap-baseline.prof heap-after.prof
# top / list / peek now show deltas
```

**Code-version comparison** — capture on each side and load both in the UI:

```bash
go test -bench=BenchmarkParse -cpuprofile=cpu-before.prof ./pkg/parser
go test -bench=BenchmarkParse -cpuprofile=cpu-after.prof ./pkg/parser
go tool pprof -http=:8080 cpu-before.prof
go tool pprof -http=:8081 cpu-after.prof
```

For statistical comparison of benchmark *numbers* — not profiles — use [`benchstat.md`](./benchstat.md).

## Common pattern vocabulary

- **Flat high + cum high** — the function itself is the bottleneck; optimize its own code.
- **Flat low + cum high** — a coordinator delegating to slow callees; drill in with `list`/`peek`.
- **`alloc_objects` high, `inuse_space` low** — short-lived allocation churn (hot-loop `fmt.Errorf` with `%v`, interface boxing, string/byte conversions, ungrown slices). Patterns in `fabianoflorentino/golang-agent-skills@golang-performance`.
- **`inuse_space` growing** — a leak. Diff two snapshots with `-base`; usual suspects: unbounded caches, maps whose deleted buckets never shrink, goroutine leaks holding references.
- **Mutex/block profile hot** — contention, not CPU. Shrink critical sections, shard locks, or go lock-free (`sync/atomic`, `sync.Map` for reads). See `fabianoflorentino/golang-agent-skills@golang-concurrency`.
- **Many goroutines stuck on one channel/mutex** — serialization bottleneck; worker pools, sharded work, or buffered channels.
- **`runtime.mallocgc` dominating CPU** — allocation rate, not compute. Switch to the `alloc_objects` heap view to name the sites, then apply `golang-performance` patterns.
- **`runtime.memmove` dominating** — large copies from slice growth, `copy()`, or string/byte conversion. Preallocate to final capacity, reuse buffers, stay in `[]byte`.
- **`runtime.scanobject` dominating** — GC pointer scanning. Lower pointer density: values over pointers in slices/maps, flatter structures, `[N]byte` over `string` in hot structs.

## Which profile for which symptom

| Symptom | Profile | Command |
| --- | --- | --- |
| High CPU, slow function | CPU | `-cpuprofile` / `pprof/profile` |
| Too many allocations | Heap `alloc_objects` | `-memprofile`, then `pprof -alloc_objects` |
| Peak memory / RSS | Heap `alloc_space` | `pprof -alloc_space` |
| Memory growing | Heap `inuse_space` | compare snapshots with `-base` |
| Lock contention | Mutex | `pprof/mutex`; enable `SetMutexProfileFraction` |
| Blocked on sync | Block | `pprof/block`; enable `SetBlockProfileRate` |
| Goroutine pile-up / leak | Goroutine | `pprof/goroutine` |
| High latency, low CPU | Goroutine + Block + Trace | scheduling and I/O waits — see [`trace.md`](./trace.md) |
| Thread churn | Threadcreate | `pprof/threadcreate` |