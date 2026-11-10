---
name: golang-samber-slog
description: "Structured logging extensions for Golang using samber/slog-**** packages — multi-handler pipelines (slog-multi), log sampling (slog-sampling), attribute formatting (slog-formatter), HTTP middleware (slog-fiber, slog-gin, slog-chi, slog-echo), and backend routing (slog-datadog, slog-sentry, slog-loki, slog-syslog, slog-logstash, slog-graylog...). Apply when using or adopting slog, or when the codebase already imports any github.com/samber/slog-* package."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🪵"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go logging architect. You assemble pipelines where records flow through the right stages — sampling sheds noise before it costs CPU, formatters scrub PII before it leaves the process, and routing parks errors in Sentry while info drifts to Loki.

**Modes:**

- **Build** — composing a logging pipeline for a service.
- **Review** — auditing stage order, missing flushes, and silent-dropped records.
- **Optimize** — cutting per-record cost on a hot path.

**When to use:** any task centered on samber/slog-\* packages. The fundamentals of `slog` itself (levels, context, default handler) live in `golang-observability`.

## The pipeline shape

Records always flow through the same order. Sampling must come first — it discards records before formatters spend CPU on them. Formatting comes before routing so every sink receives cleaned attributes.

```text
record → [sampling] → [format/scrub via Pipe] → [route] → [sinks]
```

Reversing the order wastes work on records that are dropped anyway.

## Composition: slog-multi

| Pattern | Behavior | Cost |
| --- | --- | --- |
| `Fanout(h...)` | every record to every handler, in order | sum of latencies |
| `Router().Add(h, pred)` | to all matching handlers | sum of matching |
| `Router().Add(...).FirstMatch()` | to the first match | one handler |
| `Failover(h...)` | first handler that accepts | primary latency |
| `Pool(h...)` | one handler picks each record | one handler |
| `Pipe(mws...).Handler(sink)` | middleware chain feeding a sink | middleware + sink |

Route by level, message, or attribute with predicates like `LevelIs`, `MessageContains`, `AttrValueIs`:

```go
logger := slog.New(
    slogmulti.Router().
        Add(sentryHandler, slogmulti.LevelIs(slog.LevelError)).
        Add(slog.NewJSONHandler(os.Stdout, nil)).
        Handler(), // catch-all route
)
```

A router without a catch-all silently swallows unmatched records. Full examples in [pipeline patterns](references/pipeline-patterns.md).

## Throughput control: slog-sampling

| Strategy option | Behavior | Best for |
| --- | --- | --- |
| Uniform | drop a fixed fraction of all records | dev/staging |
| Threshold | first N per tick, then a reduced rate | production — keeps early visibility |
| Absolute | hard cap per tick | cost control |
| Custom | rate decided per record | level/time-aware rules |

Sampling must sit outermost, wrapped as middleware:

```go
logger := slog.New(
    slogmulti.Pipe(
        slogsampling.ThresholdSamplingOption{
            Tick: 5 * time.Second, Threshold: 10, Rate: 0.1,
        }.NewMiddleware(),
    ).Handler(inner),
)
```

Matchers (`MatchByLevel`, `MatchByMessage`, `MatchByLevelAndMessage` default, `MatchBySource`, `MatchByAttribute`) group lookalike records for dedup. Strategies and tuning: [sampling strategies](references/sampling-strategies.md).

## Attribute transforms: slog-formatter

Apply formatters as `Pipe` middleware so every downstream handler sees scrubbed attributes; `PIIFormatter`, `ErrorFormatter`, `IPAddressFormatter`, plus generic `FormatByType[T]` / `FormatByKey` and `FlattenFormatterMiddleware`:

```go
logger := slog.New(
    slogmulti.Pipe(slogformatter.NewFormatterMiddleware(
        slogformatter.PIIFormatter("user"),
        slogformatter.ErrorFormatter("error"),
        slogformatter.IPAddressFormatter("client"),
    )).Handler(slog.NewJSONHandler(os.Stdout, nil)),
)
```

## HTTP middleware

`router.Use(sloggin.New(logger))` and friends (`slog-gin`, `slog-echo`, `slog-fiber`, `slog-chi`, `slog-http`) share a `Config` with default/error levels, request/response body capture, and filters:

```go
router.Use(sloggin.NewWithConfig(logger, sloggin.Config{
    DefaultLevel:     slog.LevelInfo,
    ClientErrorLevel: slog.LevelWarn,
    ServerErrorLevel: slog.LevelError,
    WithRequestBody:  true,
    Filters:          []sloggin.Filter{sloggin.IgnorePath("/health", "/metrics")},
}))
```

Per-framework setup: [http middlewares](references/http-middlewares.md).

## Backend sinks

Sinks follow `Option{}.NewXxxHandler()`: cloud (`slog-datadog`, `slog-sentry`, `slog-loki`, `slog-graylog`), messaging (`slog-kafka`, `slog-fluentd`, `slog-logstash`, `slog-nats`), notifications (`slog-slack`, `slog-telegram`, `slog-webhook`), storage (`slog-parquet`), and bridges (`slog-zap`, `slog-zerolog`, `slog-logrus`).

**Batch sinks must be flushed.** datadog, loki, kafka, and parquet buffer internally; skip the shutdown call and you lose the tail of the log. Wire `Stop(ctx)`/`Close()` into process teardown. Config and flush patterns: [backend handlers](references/backend-handlers.md).

## Common mistakes

| Mistake | Failure | Fix |
| --- | --- | --- |
| Sampling after formatting | CPU spent on dropped records | sampling outermost |
| `Fanout` to many sync handlers | caller latency sums them all | `Pool()` for concurrent dispatch |
| No flush on batch sinks | buffered records vanish at exit | `Stop(ctx)` / `Close()` in teardown |
| Router without catch-all | unmatched records dropped | add a predicate-less handler |
| `AttrFromContext` without middleware | context has no attributes to read | install the HTTP middleware first |
| Empty `Pipe()` | no-op with per-record overhead | omit when no middleware is needed |

## Performance notes

- `Fanout` latency = sum of its handlers; five 10 ms handlers make every log call 50 ms. `Pool` reduces to the slowest single handler.
- Each `Pipe` middleware carries per-record call overhead — keep chains short.
- Many formatters compound sequentially; for hot-path values implement `slog.LogValuer` instead.
- Benchmark the finished pipeline with `go test -bench` before shipping it.

## Best practices

1. Order stages: sample → format → route.
2. Put cross-cutting concerns (trace-ID injection, PII scrubbing) in middleware, not in per-site logic.
3. Test pipelines with `slogmulti.NewHandleInlineHandler` and assert per stage without real sinks.
4. Feed request-scoped attributes into handlers via `AttrFromContext` set by the HTTP middleware.
5. Prefer `Router` over `Fanout` when sinks need different record subsets.
6. Report upstream bugs per package, e.g. [slog-multi/issues](https://github.com/samber/slog-multi/issues).

## Cross-references

- `golang-observability` — slog fundamentals, levels, context, default handler setup.
- `golang-error-handling` — the log-or-return rule.
- `golang-security` — PII handling in logs.
- `golang-samber-oops` — structured error context feeding these pipelines.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.
