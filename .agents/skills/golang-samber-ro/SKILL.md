---
name: golang-samber-ro
description: "Reactive streams and event-driven programming in Golang using samber/ro — ReactiveX implementation with 150+ type-safe operators, cold/hot observables, 5 subject types (Publish, Behavior, Replay, Async, Unicast), declarative pipelines via Pipe, 40+ plugins (HTTP, cron, fsnotify, JSON, logging), automatic backpressure, error propagation, and Go context integration. Apply when using or adopting samber/ro, when the codebase imports github.com/samber/ro, or when building asynchronous event-driven pipelines, real-time data processing, streams, or reactive architectures in Go. Not for finite slice transforms (→ See `fabianoflorentino/golang-agent-skills@golang-samber-lo` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "👁"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who reaches for reactive streams when data arrives asynchronously or without end. You build pipelines with operators instead of raw goroutine/channel plumbing, and you know when a slice plus `lo` is the simpler answer.

**Modes:**

- **Build** — expressing an event pipeline as an observable graph.
- **Review** — auditing pipelines for unhandled errors, unbounded streams, and hot/cold confusion.
- **Debug** — tracing a missed event or a leaked subscription.

**When to use:** any task centered on samber/ro. For finite collection transforms use `samber/lo`; for bounded goroutine fan-out the stdlib `errgroup` may be all you need. Load `golang-concurrency` and `golang-observability` alongside when lifecycle and monitoring matter.

## Streams vs slices

| Need | Tool |
| --- | --- |
| Transform a slice once | `samber/lo` — eager, synchronous |
| Bounded fan-out with error handling | `errgroup` |
| Infinite event streams (websocket, ticks, fsnotify) | `samber/ro` |
| Combine several async sources with timing | `samber/ro` (combine/zip operators) |
| One source, many consumers | `samber/ro` hot observables / subjects |

## The four building blocks

1. **Observable** — emits values over time; cold by default (each subscriber triggers its own execution).
2. **Observer** — consumes the stream through `onNext`, `onError`, `onComplete`.
3. **Operator** — transforms an observable into another, composed by `Pipe`.
4. **Subscription** — the wiring between them; `Wait` blocks, `Unsubscribe` cancels.

```go
odds := ro.Pipe2(
    ro.FromChannel(rawCh),
    ro.Filter(func(n int) bool { return n%2 != 0 }),
    ro.Map(func(n int) string { return fmt.Sprintf("odd-%d", n) }),
)
odds.Subscribe(ro.NewObserver(
    func(s string) { out <- s },
    func(err error) { log.Printf("stream: %v", err) },
    func() { close(out) },
))
```

## Typed vs untyped pipelines

Use the typed `Pipe2` … `Pipe25` family for compile-time chain checking. The bare `Pipe` takes `any` and drops type safety — reach for it only when composing operators dynamically.

## Cold and hot sources

Cold (the default) re-runs per subscriber: safe, deterministic, but wasteful when the source is expensive. Hot sources share one execution among all subscribers — the natural shape for websockets, DB polls, and any single event source with many consumers.

| Conversion | Behavior |
| --- | --- |
| `Share()` | cold → hot, reference-counted teardown |
| `ShareReplay(n)` | hot + replays last `n` values to late joiners |
| `Connectable()` | hot, but waits for an explicit `Connect()` |
| Subjects | natively hot; you push with `Send`/`Error`/`Complete` |

| Subject | Replay to late subscribers |
| --- | --- |
| `PublishSubject` | none |
| `BehaviorSubject` | last value |
| `ReplaySubject` | last `N` values |
| `AsyncSubject` | last value, and only after completion |
| `UnicastSubject` | only the single subscriber |

Subject details and hot-source patterns are in [subjects guide](references/subjects-guide.md).

## Operators at a glance

| Family | Key operators |
| --- | --- |
| Creation | `Just`, `FromSlice`, `FromChannel`, `Range`, `Interval`, `Defer`, `Future` |
| Transform | `Map`, `MapErr`, `FlatMap`, `Scan`, `Reduce`, `GroupBy` |
| Filter | `Filter`, `Take`, `Skip`, `Distinct`, `First`, `Last`, `Find` |
| Combine | `Merge`, `Concat`, `Zip2…Zip6`, `CombineLatest2…5`, `Race` |
| Error | `Catch`, `OnErrorReturn`, `Retry`, `RetryWithConfig` |
| Timing | `Delay`, `Timeout`, `ThrottleTime`, `SampleTime`, `BufferWithTime` |
| Side effect | `Tap`, `TapOnNext`, `TapOnError`, `TapOnComplete` |
| Terminal | `Collect`, `ToSlice`, `ToChannel`, `ToMap` |

The full catalog lives in [operators guide](references/operators-guide.md).

## Common mistakes

| Mistake | Why it fails | Fix |
| --- | --- | --- |
| Subscribing without an error callback | errors vanish silently | `NewObserver(onNext, onError, onComplete)` |
| Bare `Pipe` | type mismatches surface at runtime | typed `Pipe2`…`Pipe25` |
| No unsubscribe on infinite streams | goroutine leak for life | `TakeUntil`, context cancellation, explicit `Unsubscribe` |
| `Share()` when cold suffices | lifecycle complexity for no consumers | hot only when many need the same source |
| Streams for slice work | goroutine + subscription overhead for a sync op | `samber/lo` |
| Not wiring context | streams ignore shutdown signals | `ContextWithTimeout` / `ThrowOnContextCancel` |

## Plugins

30–40+ plugins fold domain operators into pipelines — encoding (JSON, CSV), network (`plugins/http`, `plugins/fsnotify`), scheduling (`plugins/cron`), observability (slog, zap, zerolog), rate limiting, and string/data helpers. Browse [plugin ecosystem](references/plugin-ecosystem.md) and real-world recipes in [patterns](references/patterns.md).

## Best practices

1. Always pass all three callbacks; silent stream errors are production time-bombs.
2. Favor typed `Pipe` families; reserve `Pipe` for dynamic chains.
3. Bound every infinite stream — `Take(n)`, `TakeUntil(signal)`, `Timeout(d)`, or context.
4. Use `Collect` for finite streams you want as `[]T`.
5. `lo` for data you already hold; `ro` for data that arrives over time.
6. Report upstream bugs at [samber/ro issues](https://github.com/samber/ro/issues).

## Cross-references

- `golang-samber-lo` — eager transforms of finite slices.
- `golang-samber-mo` — monadic values that thread through pipeline operators.
- `golang-samber-hot` — in-memory caching, also shipped as an `ro` plugin.
- `golang-concurrency` — goroutine/channel patterns when streams are overkill.
- `golang-observability` — tracing and metrics for reactive pipelines.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.
