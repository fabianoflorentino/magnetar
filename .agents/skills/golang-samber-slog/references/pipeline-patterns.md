# Pipeline Patterns

Worked examples of every `slog-multi` composition. The record always flows sample → format → route → sink; the shapes below slot into that flow.

## Fanout — broadcast

Every record reaches every handler, in order, so latency sums across them:

```go
import slogmulti "github.com/samber/slog-multi"

logger := slog.New(
    slogmulti.Fanout(
        slog.NewJSONHandler(os.Stdout, nil),
        slog.NewTextHandler(logFile, nil),
        slogsentry.Option{Level: slog.LevelError}.NewSentryHandler(),
    ),
)
```

Fit: destinations needing every record (audit, compliance). For distinct record needs use Router; for latency-sensitive fan-out use Pool.

## Router — predicate routing

Records go to **all** handlers whose predicate matches. A constant hazard: records matching no predicate are silently dropped, so always append a catch-all.

```go
logger := slog.New(
    slogmulti.Router().
        Add(sentryHandler, slogmulti.LevelIs(slog.LevelError)).
        Add(slackHandler, slogmulti.LevelIs(slog.LevelWarn)).
        Add(lokiHandler, slogmulti.LevelIs(slog.LevelInfo, slog.LevelDebug)).
        Add(slog.NewJSONHandler(os.Stdout, nil)). // catch-all
        Handler(),
)
```

Built-in predicates:

```go
slogmulti.LevelIs(slog.LevelError)             // exact levels
slogmulti.LevelIsNot(slog.LevelDebug)          // level exclusion
slogmulti.MessageIs("payment processed")       // exact message
slogmulti.MessageIsNot("healthcheck")          // message exclusion
slogmulti.MessageContains("timeout")           // partial message
slogmulti.MessageNotContains("debug")          // partial exclusion
slogmulti.AttrValueIs("module", "billing")     // attribute value
slogmulti.AttrKindIs(slog.KindString)          // attribute kind
```

Custom predicates are plain `func(ctx, record) bool`:

```go
func matchRegion(region string) func(context.Context, slog.Record) bool {
    return func(ctx context.Context, r slog.Record) bool {
        match := false
        r.Attrs(func(attr slog.Attr) bool {
            if attr.Key == "region" && attr.Value.String() == region {
                match = true
                return false
            }
            return true
        })
        return match
    }
}

logger := slog.New(
    slogmulti.Router().
        Add(slackUS, matchRegion("us")).
        Add(slackEU, matchRegion("eu")).
        Handler(),
)
```

The router above has no catch-all, so records with no region are dropped — acceptable only when silently discarding unmatched records is the intended behavior.

## FirstMatch — short-circuit routing

Like Router but stops at the first matched handler, so every record reaches exactly one destination. Order encodes priority — put the most specific handlers first, a predicate-less one last:

```go
logger := slog.New(
    slogmulti.Router().
        Add(queryHandler, matchQueryLogs).     // priority 1
        Add(requestHandler, matchRequestLogs). // priority 2
        Add(defaultHandler).                   // fallback
        FirstMatch().
        Handler(),
)
```

## Failover — sequential fallback

Tries handlers in order and sticks with the first that returns a nil error — the shape for flaky network sinks:

```go
logger := slog.New(
    slogmulti.Failover()(
        slogloki.Option{Level: slog.LevelDebug, Client: lokiClient}.NewLokiHandler(),
        slog.NewJSONHandler(localFile, nil),
        slog.NewTextHandler(os.Stderr, nil),
    ),
)
```

## Pool — load-balanced dispatch

Randomly hands each record to one of several equivalent handlers, bounding latency to a single handler instead of the Fanout sum:

```go
logger := slog.New(
    slogmulti.Pool()(
        lokiHandler1, // shard 1
        lokiHandler2, // shard 2
        lokiHandler3, // shard 3
    ),
)
```

## Pipe — middleware chain

Middleware intercepts, transforms, or enriches records before the sink sees them:

```go
logger := slog.New(
    slogmulti.
        Pipe(samplingMiddleware).         // drop noise first
        Pipe(piiScrubbingMiddleware).     // mask PII
        Pipe(traceInjectionMiddleware).   // add trace_id
        Pipe(slogmulti.RecoverHandlerError( // catch handler panics
            func(ctx context.Context, record slog.Record, err error) {
                log.Println("handler error:", err)
            },
        )).
        Handler(slog.NewJSONHandler(os.Stdout, nil)),
)
```

## Inline handlers and middleware

Quick closures stand in for full structs — handy for tests and simple consumers:

```go
handler := slogmulti.NewHandleInlineHandler(
    func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
        fmt.Printf("LOG: %s %s\n", record.Level, record.Message)
        return nil
    },
)

middleware := slogmulti.NewHandleInlineMiddleware(
    func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
        record.AddAttrs(slog.String("service", "my-api"))
        return next(ctx, record)
    },
)
```

## AttrFromContext

HTTP middlewares stash request-scoped attributes in the context; backend handlers pull them out with `AttrFromContext`. The extraction only works when the context actually holds the values — install the HTTP middleware first or the functions return nil:

```go
handler := slogsentry.Option{
    Level: slog.LevelError,
    AttrFromContext: []func(ctx context.Context) []slog.Attr{
        func(ctx context.Context) []slog.Attr {
            if traceID := ctx.Value("trace_id"); traceID != nil {
                return []slog.Attr{slog.String("trace_id", traceID.(string))}
            }
            return nil
        },
    },
}.NewSentryHandler()
```

## Full production pipeline

Canonical order: sample → scrub → route → sink, with errors bypassing sampling via FirstMatch:

```go
import (
    slogmulti "github.com/samber/slog-multi"
    slogsampling "github.com/samber/slog-sampling"
    slogformatter "github.com/samber/slog-formatter"
    slogsentry "github.com/samber/slog-sentry/v2"
    slogloki "github.com/samber/slog-loki/v3"
)

sampling := slogsampling.ThresholdSamplingOption{
    Tick: 5 * time.Second, Threshold: 20, Rate: 0.1,
}.NewMiddleware()

pii := slogformatter.NewFormatterMiddleware(
    slogformatter.PIIFormatter("user"),
    slogformatter.IPAddressFormatter("client_ip"),
)

recovery := slogmulti.RecoverHandlerError(func(ctx context.Context, r slog.Record, err error) {
    log.Printf("slog handler error: %v", err)
})

sentryHandler := slogsentry.Option{Level: slog.LevelError}.NewSentryHandler()
lokiHandler := slogloki.Option{Level: slog.LevelDebug, Client: lokiClient}.NewLokiHandler()
defer lokiClient.Stop()

logger := slog.New(
    slogmulti.
        Pipe(pii).
        Pipe(recovery).
        Handler(
            slogmulti.Router().
                Add(sentryHandler, slogmulti.LevelIs(slog.LevelError)).
                Add(slogmulti.Pipe(sampling).Handler(lokiHandler)).
                FirstMatch().
                Handler(),
        ),
)
slog.SetDefault(logger)
```