# CPU Optimization

CPU-bound bottlenecks show up as one function dominating the CPU profile. The patterns below attack the usual causes: inlining that isn't happening, poor cache use, and unnecessary computation.

## Diagnosing CPU hotspots

- `go tool pprof` CPU profile — a small helper dominating cumulative time usually means it isn't being inlined.
- `go build -gcflags="-m"` — read the compiler's inline decisions; `cannot inline` gives the exact reason.
- `go test -bench=.` — baseline ns/op to attach numbers to every change.

## Function inlining

The compiler inlines small functions, eliminating call overhead. Loops, many statements, and calls to non-inlineable functions defeat inlining — which matters when the function runs millions of times.

```go
// log.Printf blocks inlining
func abs(x int) int {
    if x < 0 {
        log.Printf("negative: %d", x)
        return -x
    }
    return x
}
```

Move side effects (logging, telemetry) out of hot paths, or gate them behind a check. Inspect decisions with:

```bash
go build -gcflags="-m" ./... 2>&1 | grep "can inline"
go build -gcflags="-m" ./... 2>&1 | grep "inlining call"
```

### Value receivers enable inlining

Pointer receivers add indirection that stops inlining. Value receivers let the compiler fuse fluent chains:

```go
// value receiver: fully inlined, markedly faster in fluent chains
func (c config) WithTimeout(d time.Duration) config {
    c.timeout = d
    return c
}
```

## Cache locality

CPUs fetch memory in 64-byte cache lines and prefetch sequential access. Random access wastes that.

- **Row-major loops** — Go stores 2D data row-major; column-first traversal jumps between lines and can cost 10-50x on large matrices. Iterate rows outer, columns inner.
- **Contiguous allocations** — allocate the matrix as one `[]float64` and slice rows out of it, instead of `make` per row: one allocation, sequential memory.
- **SoA over AoS** — iterating one field of an Array-of-Structs loads the whole struct (cache waste). Group fields into parallel slices (`xs`, `ys`, `zs`) when you iterate a single field; keep AoS when fields are always read together or structs are small.
- **Index over pointer** — linked structures scatter nodes across the heap. Store nodes in a contiguous array and reference them by index:

```go
type Node struct {
    value              int
    left, right        int // indices into nodes []Node
}
```

## False sharing

Two goroutines writing different fields of the same cache line make the cores invalidate each other's copies. Separate contended counters with padding:

```go
type Counters struct {
    a int64
    _ [56]byte // push b onto its own cache line
    b int64
}
```

Apply only when profiling confirms contention on concurrent counters/flags; padding everything wastes memory.

## Instruction-level parallelism

A single accumulator serializes a loop — each addition waits on the previous. Multiple accumulators let the CPU run independent chains concurrently; expect 2-4x on pure arithmetic loops:

```go
var s0, s1, s2, s3 int64
limit := len(data) - len(data)%4
for i := 0; i < limit; i += 4 {
    s0 += data[i]
    s1 += data[i+1]
    s2 += data[i+2]
    s3 += data[i+3]
}
for i := limit; i < len(data); i++ {
    s0 += data[i]
}
total := s0 + s1 + s2 + s3
```

## SIMD

Reach for SIMD only when a numeric inner loop consumes >20% of CPU and the compiler didn't already vectorize it.

- **Auto-vectorization first** — write plain loops over `[]float64`/`[]int32`. Check with `go build -gcflags="-d=ssa/prove/debug=2"`; inspect the SSA (`GOSSAFUNC=MyFunc go build`) or objdump the binary for vector instructions (e.g. `VMOVAPD`, `VADDPD`).
- **`math/bits`** — `OnesCount`, `LeadingZeros`, `RotateLeft` map to single instructions (POPCNT, CLZ, ROL).
- **Experimental `simd` packages** — Go 1.26+ ships intrinsics behind `GOEXPERIMENT=simd`: `simd/archsimd` is architecture-specific (amd64; arm64/wasm at 128-bit from Go 1.27), while Go 1.27's portable `simd` package (types like `Int8s`, `Float32s`) falls back to scalar code. Both are unstable API — never expose them publicly, and verify import paths against your toolchain.
- **Third-party libraries** — prefer SIMD-optimized hashing/compression/encoding libraries over writing your own.
- **Hand-written assembly** — `.s` files with AVX2/NEON are the last resort: high maintenance, platform-specific, hard to debug.

### Handling CPU-specific instruction sets

| Strategy | How | Cost |
| --- | --- | --- |
| Build on production hardware | compile on a machine matching the target CPU | per-CPU binaries, breaks CI portability |
| Runtime dispatch | detect features via `cpu.X86.HasAVX2` in `init()` and pick an implementation | one dispatch call, single binary (stdlib base64/sha256 style) |
| Build tags | `//go:build amd64 && !nosimd` selects per-target code | zero runtime overhead, multiple binaries to ship |

## Tight loops and the scheduler

A CPU-only loop with no calls may not reach a preemption point and can starve other goroutines. Rare since Go 1.14 added asynchronous preemption, but a fully inlined arithmetic loop is still the risk case.

- Confirm starvation before acting: `go tool pprof` goroutine profile for runnable-but-not-running goroutines, `go tool trace` for one goroutine monopolizing a P, or `/sched/latencies:seconds` from `runtime/metrics`.
- **Function calls are preemption points** — a loop body that calls a non-inlined function lets the scheduler intervene.
- **`//go:noinline`** — force a small hot function to stay a call so the call site becomes a guaranteed preemption point. It costs ~10-30 cycles per call and reduces caller ILP; apply only when profiling proves starvation.

Short bursts (<10ms) don't need this; prioritize inlining there.

## Reflection and type assertions

`reflect` in a hot path is 10-100x slower than typed code; `reflect.DeepEqual` is 50-200x slower than typed equality. Use `slices.Equal`, `maps.Equal`, `bytes.Equal`, or generics.

Prefer one type switch over repeated assertions:

```go
switch v := v.(type) {
case string:
    return v
case int:
    return strconv.Itoa(v)
}
```

## Monotonic time

Compare durations with `time.Since`/`time.Since(start)`, which use the monotonic clock — immune to wall-clock adjustments (NTP, DST) and slightly cheaper than subtracting wall times:

```go
elapsed := time.Since(appStart)
if elapsed > threshold { ... }
```
