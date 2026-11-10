---
name: golang-concurrency
description: "Golang concurrency design — goroutine lifecycle and leak prevention, channels and `select`, channel ownership and direction, `sync.Mutex`/`RWMutex`/`sync.Map`/`sync.Once`/atomics, `errgroup`, `singleflight`, worker pools, and fan-out/fan-in pipelines. Use when writing or reviewing concurrent Go code, when choosing between channels and mutexes, when protecting a shared map or counter, or when a goroutine has no clear exit. Not for defensive coding unrelated to concurrency such as nil panics, slice aliasing, or numeric overflow (→ See `fabianoflorentino/golang-agent-skills@golang-safety` skill), and not for debugging a specific hung, crashing, or racing program after the fact (→ See `fabianoflorentino/golang-agent-skills@golang-troubleshooting` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "⚡"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
paths:
  - "**/*.go"
---

# Concurrency in Go

**Persona:** You are a Go concurrency engineer. Every goroutine you spawn is a liability until proven necessary — correctness and leak-freedom come before speed, and every concurrent value has exactly one owner.

**Modes:**

- **Write** — implement goroutines, channels, sync primitives, pools, pipelines. Sequential, in the order below.
- **Review** — review a diff for leaks, missing context, ownership violations, unprotected shared state. Sequential.
- **Audit** — survey concurrent code across a codebase. Fan out one sub-agent per concern and merge. Parallel.

**When to use:** any Go code that runs concurrently or protects shared state. For defensive pitfalls unrelated to concurrency (nil panics, slice aliasing) use `golang-safety`; for live debugging of hangs/races use `golang-troubleshooting`; for cancellation and deadlines use `golang-context`.

## Principles

1. **Every goroutine must have a clear exit** — context, done channel, or explicit wait. Goroutines are cheap but never free; a leaked one accumulates silently until the process dies.
2. **Share memory by communicating** — a channel is an explicit ownership handoff; a mutex makes ownership implicit and easy to get wrong.
3. **Send copies, not pointers**, on channels — a pointer on a channel recreates invisible shared memory and voids the point of the handoff.
4. **Only the sender closes a channel** — a receiver close panics the next sender; closed-from-receiver is a race by construction.
5. **Declare channel direction** (`chan<-`, `<-chan`) — the compiler enforces misuse at build time.
6. **Prefer unbuffered channels.** Buffers hide backpressure; introduce them with measured justification, not for luck.
7. **Always select on `ctx.Done()`** — a goroutine that never looks at cancellation is a leak the moment the caller gives up.
8. **Keep the remainder of the heap** — reuse `time.After` results; each call allocates a timer. In hot loops use `time.NewTimer` + `Reset`.

## Choosing the coordination primitive

| Situation | Use | Why |
| --- | --- | --- |
| Passing data between goroutines | Channel | Expresses ownership transfer |
| Coordinating lifecycle | Channel + context | Clean cancellation via select |
| Protecting a few shared fields | `sync.Mutex` / `sync.RWMutex` | Simple critical sections |
| Counters, flags | `sync/atomic` typed atomics | Lock-free; `atomic.Int64`, `atomic.Bool` (Go 1.19+) |
| Read-heavy map | `sync.Map` | Tuned for that skew; concurrent map writes still crash hard |
| Expensive one-time computation | `sync.Once` / `singleflight` | Run once / dedupe concurrent callers |

| Need | Use | Why |
| --- | --- | --- |
| Just wait (no errors) | `sync.WaitGroup` | Zero ceremony fire-and-wait. Go 1.25+: `wg.Go(func(){})` for non-panicking tasks |
| Wait + first error | `errgroup.Group` | Propagates the first failure |
| Wait + cancel siblings on error | `errgroup.WithContext` | One failure stops the set |
| Wait + bounded concurrency | `errgroup.SetLimit(n)` | Built-in pool, no hand-rolled semaphore |

Rules for these decisions:

- `sync.Mutex`: keep critical sections short; never hold a lock across I/O.
- `sync.RWMutex`: many readers, few writers; never upgrade an `RLock` to `Lock` (deadlock by design).
- `sync.Pool`: reuse temporary objects, call `Reset()` before `Put`, reduces GC pressure — but nothing here is safe across goroutines by itself.
- `sync.Once` (Go 1.21+): `OnceFunc`, `OnceValue`, `OnceValues` when you need the cached value instead of a closure.
- `singleflight`: cache stampede prevention; the shared result is one real compute per wave, not N.
- `wg.Add` must happen **before** `go func()` is scheduled — `Add` inside the goroutine can let `Wait` return early.

## The goroutine spawn checklist

Before writing `go`, answer:

- [ ] How does it exit? Context cancellation, channel close, or explicit signal
- [ ] Can it be stopped? A `context.Context` or done channel is in scope
- [ ] Can we wait for it? WaitGroup or errgroup, and someone calls `Wait`
- [ ] Who owns the channels? Creator/sender owns and closes
- [ ] Should this be synchronous instead? Concurrency without a measured need is complexity in advance

If any box is unchecked, do not spawn.

## Pipelines and worker pools

Wire streams with its default idioms: generator stages, fan-out to bounded workers, fan-in with `run` or dedicated aggregation. Go 1.23+ iterators (`range` over funcs) change stage plumbing — see the pipelines reference for the current shapes and `samber/ro` for functional pipeline helpers.

## Parallel concurrency audits

For a whole-tree audit, fan out by concern and merge:

1. Goroutine spawns (`go func`, `go method`) — verify each has a shutdown mechanism
2. Mutable globals and unsynchronized shared state
3. Channels — ownership, direction, closure, buffer size
4. `time.After` in loops, missing `ctx.Done()`, unbounded spawning
5. Mutex, `sync.Map`, atomics, and thread-safety documentation

## Mistakes at a glance

| Mistake | Fix |
| --- | --- |
| Fire-and-forget goroutine | Stop mechanism (context/done) |
| Receiver closes channel | Only the sender closes |
| `time.After` in hot loop | Reused `time.NewTimer` + `Reset` |
| No `ctx.Done()` in select | Select on context for cancellation |
| Unbounded spawning | `errgroup.SetLimit(n)` or semaphore |
| Pointer on a channel | Send copies / immutable values |
| `wg.Add` inside goroutine | `Add` before `go` |
| No `-race` in CI | `go test -race ./...` |
| Lock across I/O | Keep critical sections short |

## Goroutine leak detection

Track leaks in tests with `go.uber.org/goleak` (`goleak.VerifyTestMain(m)` or `defer goleak.VerifyNone(t)`). In production, the goroutine leak profile in `runtime/pprof` (GA since Go 1.27 — in 1.26 behind `GOEXPERIMENT=goroutineleakprofile`) reports goroutines that never complete:

```bash
go tool pprof http://localhost:6060/debug/pprof/goroutineleak
```

Keep the cheap signals: `runtime.NumGoroutine()` in health checks, `/debug/pprof/goroutine?debug=2` for stacks, `-race` in CI.

## References

- [`references/channels-and-select.md`](./references/channels-and-select.md) — channel idioms and `select` patterns.
- [`references/sync-primitives.md`](./references/sync-primitives.md) — deep examples and anti-patterns for the sync toolkit.
- [`references/pipelines.md`](./references/pipelines.md) — fan-out/fan-in, bounded workers, Go 1.23 iterators, `samber/ro`.
- [Go Concurrency Patterns: Pipelines](https://go.dev/blog/pipelines) — governor patterns for stream processing.
- [Effective Go: Concurrency](https://go.dev/doc/effective_go#concurrency) — the canonical guidance.

## Cross-references

- `golang-context` — cancellation propagation and timeouts that drive these leaks away.
- `golang-safety` — concurrent map access is a crash; defensive rules apply around any shared state.
- `golang-performance` — false sharing, cache-line padding, `sync.Pool` hot-path tuning.
- `golang-troubleshooting` — goroutine dumps and deadlock diagnosis.
- `golang-design-patterns` — graceful shutdown shapes that bound every goroutine.
- `golang-continuous-integration` — `-race` and AI review gating in CI.
