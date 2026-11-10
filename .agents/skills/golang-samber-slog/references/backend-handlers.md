# Backend Handlers

Every backend adapter implements `slog.Handler` and shares the `Option{}.NewXxxHandler()` constructor pattern. Options are thin; the heavy resources (clients, writers, buffers) are built beforehand and passed in.

## Common option fields

| Field | Purpose |
| --- | --- |
| `Level` | Minimum level accepted (default `slog.LevelDebug`) |
| `AddSource` | Include source file and line |
| `ReplaceAttr` | Attribute rewrite before emission |
| `Converter` | Custom payload builder for the target format |
| `AttrFromContext` | Functions extracting attributes from `context.Context` |

## Cloud backends

**Datadog** (`slog-datadog/v2`) batches by default — records are buffered and pushed every few seconds. Skipping the flush loses the tail of the log.

```go
import slogdatadog "github.com/samber/slog-datadog/v2"

handler := slogdatadog.Option{
    Level: slog.LevelInfo,
}.NewDatadogHandler()
defer handler.(interface{ Stop(context.Context) error }).Stop(context.Background())
```

`Stop(ctx)` flushes at shutdown; `Flush(ctx)` pushes mid-lifecycle. Service, source, hostname, and tags come from the Datadog client configuration.

**Sentry** (`slog-sentry/v2`) recognizes attributes by key: any `error`, plus `request`, `dist`, `environment`, `release`, `server_name`, and `transaction`. The global `ErrorKeys = []string{"error", "err"}` marks which attributes count as errors. Use `slog.Group("tags", ...)` for Sentry tags and `slog.Group("user", ...)` for user context.

```go
import slogsentry "github.com/samber/slog-sentry/v2"

handler := slogsentry.Option{
    Level:     slog.LevelWarn,
    Hub:       sentry.CurrentHub(),
    AddSource: true,
}.NewSentryHandler()
defer sentry.Flush(2 * time.Second)
```

**Loki** (`slog-loki/v3`) sends attributes as labels by default. High-cardinality keys (request IDs, trace IDs) overflow the label set, so set `HandleRecordsWithMetadata: true` to ship them as structured metadata instead. The client must be stopped at exit.

```go
import slogloki "github.com/samber/slog-loki/v3"

lokiClient, _ := loki.New(lokiCfg)
defer lokiClient.Stop()

handler := slogloki.Option{
    Level:  slog.LevelDebug,
    Client: lokiClient,
}.NewLokiHandler()
```

**Graylog** (`slog-graylog/v2`) emits GELF over UDP:

```go
import sloggraylog "github.com/samber/slog-graylog/v2"

gelfWriter, _ := gelf.NewWriter("localhost:12201")
handler := sloggraylog.Option{
    Level:  slog.LevelDebug,
    Writer: gelfWriter,
}.NewGraylogHandler()
```

## Messaging backends

**Kafka** (`slog-kafka/v2`):

```go
import slogkafka "github.com/samber/slog-kafka/v2"

writer := &kafka.Writer{
    Addr:  kafka.TCP("localhost:9092"),
    Topic: "logs",
    Async: true, // non-blocking writes
}
handler := slogkafka.Option{
    Level:       slog.LevelDebug,
    KafkaWriter: writer,
    Timeout:     60 * time.Second,
}.NewKafkaHandler()
defer writer.Close()
```

**Fluentd** (`slog-fluentd/v2`):

```go
client, _ := fluent.New(fluent.Config{FluentHost: "localhost", FluentPort: 24224})
handler := slogfluentd.Option{
    Level:  slog.LevelDebug,
    Client: client,
    Tag:    "api",
}.NewFluentdHandler()
defer client.Close()
```

**Logstash** (`slog-logstash/v2`) writes JSON with `@timestamp`, `level`, `message`, `error`, and `extra` to a TCP connection:

```go
conn, _ := net.Dial("tcp", "localhost:9999")
handler := sloglogstash.Option{
    Level: slog.LevelDebug,
    Conn:  conn,
}.NewLogstashHandler()
defer conn.Close()
```

## Notification backends

**Slack** (`slog-slack/v2`) accepts either a webhook URL or a bot token:

```go
handler := slogslack.Option{
    Level:      slog.LevelError,
    WebhookURL: "https://hooks.slack.com/services/...",
    Channel:    "alerts",
}.NewSlackHandler()
```

**Telegram** (`slog-telegram/v2`) posts to a channel via a bot token; **Webhook** (`slog-webhook/v2`) POSTs to any endpoint with a timeout. All three sit on the error path and deliver synchronously.

## Storage backends

**Parquet** (`slog-parquet/v2`) buffers records and writes Parquet files to any Thanos `objstore.Bucket` (S3, GCS, Azure) once `maxRecords` or `maxInterval` is reached:

```go
buffer := slogparquet.NewParquetBuffer(bucket, "logs/", 10000, 5*time.Minute)
defer buffer.Flush(true) // synchronous flush of the remainder

handler := slogparquet.Option{
    Level:  slog.LevelDebug,
    Buffer: buffer,
}.NewParquetHandler()
```

## Logging bridges

`slog-zap`, `slog-zerolog`, and `slog-logrus` point `slog` at a legacy logger during incremental migration:

```go
import slogzap "github.com/samber/slog-zap/v2"

zapLogger, _ := zap.NewProduction()
handler := slogzap.Option{
    Level:  slog.LevelDebug,
    Logger: zapLogger,
}.NewZapHandler()
slog.SetDefault(slog.New(handler)) // slog.Info() now routes through Zap
```

## Graceful shutdown checklist

Handlers that buffer internally must be flushed or the tail is lost:

| Handler | Shutdown call | Without it |
| --- | --- | --- |
| `slog-datadog` | `handler.Stop(ctx)` | The buffered batch (default ~5s) is dropped |
| `slog-loki` | `lokiClient.Stop()` | Pending pushes are dropped |
| `slog-kafka` | `writer.Close()` | Queued messages never send |
| `slog-parquet` | `buffer.Flush(true)` | The partial file is not written |

Synchronous handlers (Sentry, Slack, Telegram, Webhook) need no close, though `sentry.Flush(timeout)` is still recommended.

```go
func main() {
    lokiClient, _ := loki.New(lokiCfg)
    defer lokiClient.Stop()

    slog.SetDefault(slog.New(slogloki.Option{
        Level: slog.LevelDebug, Client: lokiClient,
    }.NewLokiHandler()))

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()
    // ... start server ...
    <-ctx.Done()
    // deferred Stop() flushes here
}
```