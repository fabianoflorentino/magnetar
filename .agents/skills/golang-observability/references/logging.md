# Structured Logging with `slog`

Log management systems index, filter, and aggregate structured key/value pairs — none of that works on formatted strings:

```go
// freeform — unfilterable by user_id
log.Printf("ERROR: create user %s failed: %v", userID, err)

// structured — machine-parseable
slog.Error("user creation failed", "user_id", userID, "error", err)
```

See `golang-error-handling` for the single-handling rule that keeps log output clean.

## Handler setup

- **Production: JSON** — plain-text output breaks when a line contains embedded newlines (stack traces); collectors split it into separate records. Use `slog.NewJSONHandler` to `os.Stdout`:

```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})))
```

- **Development: text** — the readable `key=value` output of `slog.NewTextHandler` on stderr at `slog.LevelDebug`.

## Levels

| Level | Meaning | Example |
| --- | --- | --- |
| `Debug` | detailed internals | cache lookups, retry attempts |
| `Info` | routine lifecycle | order created, request completed |
| `Warn` | degraded but succeeded | rate limit approached |
| `Error` | an operation failed | payment failed |

Rule of thumb: unsure between Warn and Error? Ask "did the operation succeed?" Yes (even with degradation) = Warn; no = Error.

## Cost of logging

Every line costs CPU (formatting), I/O (pipe/disk), and money (ingestion). Volume is controlled by level:

- **Debug in production** can multiply ingestion 10-100x; the production default is `Info`, with `Debug` enabled only while chasing a specific issue.
- High-throughput services can sample verbose logs — `samber/slog-sampling` (e.g. emit 1 in 100 Debug lines) instead of dropping them entirely.

## Logging with context

Use the `*Context` variants. With an OpenTelemetry bridge configured, `trace_id` and `span_id` are injected into each record automatically:

```go
slog.ErrorContext(ctx, "query failed", "error", err)
```

## Request-scoped attributes

`slog.With` builds a child logger that adds attributes to every line. Middleware seeds it with request fields:

```go
logger := slog.With(
    "request_id", r.Header.Get("X-Request-ID"),
    "method", r.Method,
    "path", r.URL.Path,
)
```

Store the enriched logger in the request context and read it downstream so every log line from the handler chain carries the same request.

## The slog ecosystem

| Purpose | Package |
| --- | --- |
| Fan-out, routing, failover | `samber/slog-multi` |
| Sampling high-volume logs | `samber/slog-sampling` |
| Transform/format attributes | `samber/slog-formatter` |
| HTTP server middleware | `samber/slog-http` (plus `slog-gin`, `slog-echo`, `slog-fiber`, `slog-chi`) |
| Terminal color output | `lmittmann/tint` |
| Backends | `slog-datadog`, `slog-sentry`, `slog-loki`, `slog-nats`, `slog-syslog`, `slog-fluentd`, `slog-slack` |
| Legacy bridges | `slog-zap`, `slog-logrus`, `slog-zerolog` |

Full index: the [slog resources wiki](https://go.dev/wiki/Resources-for-slog).

## Migrating from zap / logrus / zerolog

`log/slog` is stdlib since Go 1.21 with a stable API and a broad ecosystem — migrate in three steps:

1. **Bridge** — route `slog` through the existing logger so output shape doesn't change while you migrate call sites:

```go
zapLogger, _ := zap.NewProduction()
slog.SetDefault(slog.New(
    slogzap.Option{Level: slog.LevelInfo, Logger: zapLogger}.NewZapHandler(),
))
```

2. **Replace call sites** — `zap.L().Info("order created", zap.String("order_id", id))`, `logrus.WithField("order_id", id).Info(...)`, and `log.Info().Str("order_id", id).Msg(...)` all become `slog.Info("order created", "order_id", id)`.
3. **Drop the bridge** — once no legacy calls remain, swap in the native JSON handler and remove the old logger dependency:

```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})))
```

## Common mistakes

```go
// log AND return — the error is reported twice up the chain (single-handling rule)
if err != nil {
    slog.Error("query failed", "error", err)
    return fmt.Errorf("query: %w", err)
}
// prefer: return the wrapped error, log once at the top level
```

```go
// never log PII — emails, tokens, account numbers are data, not identifiers
slog.Info("user logged in", "user_id", user.ID)
```
