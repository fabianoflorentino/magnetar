---
name: golang-pitfalls-optimizations
description: "Golang performance optimization — not understanding CPU caches and cache lines, false sharing in concurrent code, instruction-level parallelism, data alignment, stack vs heap allocation, how to reduce allocations, not relying on inlining, not using pprof and the execution tracer, not understanding how the GC works, and running Go in Docker/Kubernetes. Distilled from mistakes #91-100 of 100 Go Mistakes and How to Avoid Them. Apply when optimizing, profiling, or reviewing Golang code for performance."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🚀"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Optimizations

Source material: mistakes #91-100 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when optimizing Go code. Always validate optimizations with benchmarks and profiling before/after.

## 91. Not understanding CPU caches (#91)

- L1 cache is roughly 50-100x faster than main memory — data layout and access patterns drive CPU-bound performance.
- **Cache line**: CPUs fetch a block (~64 bytes), not a word at a time. Enforce **spatial locality** so each fetched cache line is fully used.
- **Slice of structs vs. struct of slices**: the layout matters — iterate over data laid out contiguously to maximize cache-line utilization.
- **Predictability**: unit/constant strides are predictable for the CPU (prefetching); non-unit strides (linked lists) are not.
- **Cache placement policy**: caches are partitioned; a critical stride can map everything to few sets, using only a tiny portion of the cache.

## 92. Writing concurrent code that leads to false sharing (#92)

- L1/L2 caches are **per core**. When a cache line is shared across cores and at least one goroutine writes, the whole line is invalidated — even if the goroutines write *different* fields (logically independent). This is **false sharing**, and it hurts performance.
- Example: two goroutines incrementing adjacent `result.sumA` and `result.sumB` in the same struct — the two fields land in the same cache line 7/8 of the time.
- Fixes:
  - **Padding** between fields: `sumA int64; _ [56]byte; sumB int64` (keep them in separate 64-byte lines).
  - **Communication**: restructure so each goroutine owns its own struct/result (or communicate via channels). ~40% faster in the book's benchmark.
- "Sharing memory is an illusion" — cache lines, not variables, are the coherence unit.

## 93. Not taking into account instruction-level parallelism (#93)

- ILP lets a CPU execute independent instructions in parallel within a single core.
- Identify and remove **data hazards** (instructions depending on the same value) so the CPU can execute more instructions concurrently. Reorganize operations to expose parallelism.

## 94. Not being aware of data alignment (#94)

- In Go, basic types are aligned to their own size. Fields are padded to alignment, so field order affects struct size.
- Order struct fields **by size descending** to avoid padding holes → more compact structs, less memory, better spatial locality.

```go
// Better: b int (8), i int16 (2), s bool (1) — no gaps below alignment.
type T struct {
    b int
    i int16
    s bool
}
```

## 95. Not understanding stack vs. heap (#95)

- **Stack** allocations are nearly free; **heap** allocations are slower and rely on the GC to reclaim.
- Understand when values escape to the heap (e.g., returning pointers, interfaces, closures capturing variables) and design to favor stack allocation where possible.

## 96. Not knowing how to reduce allocations (#96)

- Reduce allocations via:
  - **API design** that prevents sharing up (avoid forcing heap allocation through interfaces/pointers).
  - **Compiler optimizations** (inlining, escape analysis) — measure where allocations actually happen.
  - **`sync.Pool`** to reuse expensive, transient objects (careful with size/capacity-dependent state).

## 97. Not relying on inlining (#97)

- Inlining replaces a call with the function body, eliminating call overhead; small hot functions are inlined by default.
- Use **fast-path inlining**: put the common case in a small inlinable function and keep the rare, heavy path in a non-inlined function, reducing the amortized cost of calls.

## 98. Not using Go diagnostics tooling (#98)

- **`pprof`** profiling (enable with `net/http/pprof` at `/debug/pprof/...`, or flags like `-cpuprofile`):
  - `cpu` — where the app spends time (sample-based, 10 ms rate).
  - `heap` (`?gc=1` to force GC; `-diff_base` to compare) — current heap usage and leak tracking.
  - `allocs` — past allocations since start.
  - `goroutine` (`?debug=2` full stack dump) — goroutine counts and leaks.
  - `block` (enable with `runtime.SetBlockProfileRate`) — where goroutines block on sync primitives.
  - `mutex` (enable with `runtime.SetMutexProfileFraction`) — mutex contention.
  - View with `go tool pprof -http=:8080 <file>`.
- Enable only one profiler at a time; CPU/heap together give erroneous observations.
- **Execution tracer** (`-trace trace.out`), viewed with `go tool trace`: captures runtime events, goroutine execution, GC phases; detects poorly parallelized execution. Granularity is per goroutine unless you use `runtime/trace` tasks/regions.
- Read the signs: excessive `runtime.mallocgc` → many small heap allocations; heavy channel/mutex time → contention; heavy `syscall.Read/Write` → I/O buffering work.

## 99. Not understanding how the GC works (#99)

- The Go GC is concurrent/mark-and-sweep; you can tune it via `GOGC` (whether / aggressive) and `GOMEMLIMIT` (soft memory limit, Go 1.19+).
- Tuning the GC helps handle sudden load increases and reduce GC-induced latency. Understand the trade-off between memory footprint and pause/CPU overhead.

## 100. Impacts of running Go in Docker/Kubernetes (#100)

- Not relevant anymore from Go 1.25 — `GOMAXPROCS` is now container-aware automatically (sees cgroup CPU limits). On older Go versions, set `GOMAXPROCS` explicitly (e.g., via `automaxprocs`) when running in CPU-limited containers.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-performance` for the full optimization methodology and hot-path patterns
- → See `fabianoflorentino/golang-agent-skills@golang-benchmark` for measurement, pprof interpretation, and benchstat regression detection
- → See `fabianoflorentino/golang-agent-skills@golang-troubleshooting` for root-causing performance issues
- → See `fabianoflorentino/golang-agent-skills@golang-concurrency` for parallelization design that avoids false sharing
