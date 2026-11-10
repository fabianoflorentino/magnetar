---
name: golang-pitfalls-concurrency-foundations
description: "Golang concurrency foundations — concurrency vs parallelism, thinking concurrency is always faster, channels vs mutexes, not understanding race problems (data race vs race condition), not understanding workload types (CPU vs I/O bound, GOMAXPROCS), and misunderstanding Go contexts. Distilled from mistakes #55-60 of 100 Go Mistakes and How to Avoid Them. Apply when reasoning about basic Golang concurrency, goroutines, and context usage."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔄"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Concurrency Foundations

Source material: mistakes #55-60 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when reasoning about basic Go concurrency.

## 55. Mixing up concurrency and parallelism (#55)

- **Concurrency is about structure** (breaking a problem into independently executible parts); **parallelism is about execution** (running parts at the same time).
- Concurrency provides the structure that *enables* parallelism.

## 56. Thinking concurrency is always faster (#56)

- Concurrency is not automatically faster. Creating goroutines and scheduling has costs.
- Parallel merge sort on 4 cores, naively spawning goroutines for each half, was ~an order of magnitude *slower* than sequential for 10k elements.
- The fix: only parallelize workloads above a threshold (e.g., a `max` slice size, benchmarked for your machine).
- Validate assumptions with benchmarks and profiling; start sequential when unsure.

## 57. Being puzzled about channels vs. mutexes (#57)

- **Parallel** goroutines need *synchronization* (exclusive access to a shared resource) → **mutexes**.
- **Concurrent** goroutines need *coordination/orchestration* (signaling, aggregation, ownership transfer) → **channels**.
- Don't force channels everywhere because Go "promotes sharing memory by communication." Both are complementary tools.

## 58. Not understanding race problems (#58)

- **Data race**: two or more goroutines access the same memory location simultaneously and at least one writes — undefined behavior.
- **Race condition**: behavior depends on the sequence/timing of uncontrolled events. An application can be data-race-free yet non-deterministic.
- Prevent data races with `sync/atomic`, mutexes, or channels; use `go test -race`.

## 59. Not understanding workload type impacts (#59)

- **CPU-bound** workloads: optimal goroutine count ≈ `runtime.GOMAXPROCS` (default = number of CPU cores).
- **I/O-bound** workloads: the count depends on the external system, not the cores.
- Identify the workload type before sizing goroutine pools.

## 60. Misunderstanding Go contexts (#60)

- `context.Context` carries a **deadline**, a **cancellation signal**, and a **key-value list** across API boundaries.
- Deadlines: from a duration (`context.WithTimeout`) or a time (`context.WithDeadline`).
- Cancellation: `Done()` returns a receive-only channel that is **closed** on cancel/deadline — closure is broadcast to all consumers (the only channel action received by all goroutines).
- Functions the user waits on should take a context so callers can abort them.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-concurrency` for the full goroutine, channel, and sync-primitive patterns
- → See `fabianoflorentino/golang-agent-skills@golang-context` for idiomatic context.Context creation, cancellation, and propagation
- → See `fabianoflorentino/golang-agent-skills@golang-testing` for race detection and goroutine-leak testing
- → See `fabianoflorentino/golang-agent-skills@golang-benchmark` for validating decisions with measurements
