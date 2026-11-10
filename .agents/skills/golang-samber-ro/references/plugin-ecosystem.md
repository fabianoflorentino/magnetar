# Plugin Ecosystem

40+ plugins fold domain operators into `ro` pipelines. Each is its own Go module, so pull only what a service needs:

```bash
go get github.com/samber/ro/plugins/<category>/<name>
```

## Data manipulation

| Plugin | Import | Operators for |
| --- | --- | --- |
| Bytes | `plugins/bytes` | Byte-slice operations on streams |
| Strings | `plugins/strings` | Split, trim, join |
| Sort | `plugins/sort` | Ordering |
| Strconv | `plugins/strconv` | String/numeric conversion |
| Iter | `plugins/iter` | Go 1.23+ iterator interop |
| SIMD | `plugins/exp/simd` | Experimental SIMD numeric transforms |

## Encoding

| Plugin | Import | Operators for |
| --- | --- | --- |
| JSON | `plugins/encoding/json` | Marshal/unmarshal in-stream |
| CSV | `plugins/encoding/csv` | CSV rows in, rows out |
| Base64 | `plugins/encoding/base64` | Base64 encode/decode |
| Gob | `plugins/encoding/gob` | Go binary encoding |

```go
import rojson "github.com/samber/ro/plugins/encoding/json"

parsed := ro.Pipe1(rawBytes, rojson.Unmarshal[MyStruct]())
```

## Scheduling

`plugins/cron` emits on cron expressions; `plugins/ics` parses iCal files into event streams:

```go
import rocron "github.com/samber/ro/plugins/cron"

daily := rocron.Schedule("0 0 * * *")
```

## Network and I/O

| Plugin | Import | Operators for |
| --- | --- | --- |
| HTTP | `plugins/http` | Request operators (GET, POST, …) |
| I/O | `plugins/io` | File and stream reading/writing |
| FSNotify | `plugins/fsnotify` | File-system change events |

```go
import rofsnotify "github.com/samber/ro/plugins/fsnotify"

ro.Pipe1(
    rofsnotify.Watch("/var/log/app/"),
    ro.Filter(func(e fsnotify.Event) bool { return e.Op == fsnotify.Write }),
).Subscribe(ro.OnNext(func(e fsnotify.Event) {
    log.Println("Modified:", e.Name)
}))
```

## Observability

| Plugin | Import | Integrates |
| --- | --- | --- |
| Log | `plugins/observability/log` | stdlib `log` |
| Zap | `plugins/observability/zap` | Uber Zap |
| Logrus | `plugins/observability/logrus` | Logrus |
| Slog | `plugins/observability/slog` | `log/slog` |
| Zerolog | `plugins/observability/zerolog` | Zerolog |
| Sentry | `plugins/observability/sentry` | Sentry |
| Oops | `plugins/samber/oops` | samber/oops structured errors |

```go
import roslog "github.com/samber/ro/plugins/observability/slog"

ro.Pipe1(dataStream, roslog.Tap[Data](logger, slog.LevelInfo))
```

## Rate limiting

| Plugin | Import | Mechanism |
| --- | --- | --- |
| Native | `plugins/ratelimit/native` | Built-in token bucket |
| Ulule | `plugins/ratelimit/ulule` | ulule/limiter integration |

## Text processing

`plugins/regexp` matches and extracts on streams; `plugins/template` renders Go text/html templates.

## System integration

`plugins/proc` runs process operators; `plugins/signal` turns OS signals into an observable — the canonical shutdown signal:

```go
import rosignal "github.com/samber/ro/plugins/signal"

shutdown := rosignal.Notify(syscall.SIGTERM, syscall.SIGINT)
ro.Pipe1(workStream, ro.TakeUntil[Work, os.Signal](shutdown))
```

## Validation

`plugins/ozzo/ozzo-validation` validates stream values as they flow.

## Testing

`plugins/testify` provides assertions for observable streams.

## Utilities

`plugins/hyperloglog` estimates cardinality on streams; `plugins/samber/hot` integrates the in-memory cache; `plugins/samber/psi` emits starvation/pressure metrics.

## Plugin convention

Every plugin operator keeps the same contract:

1. Import the plugin package and use its operators inside `Pipe` chains.
2. Operators return `func(Observable[T]) Observable[R]` — standard pipeline stages.
3. No global state: each operator instance is independent.

Per-plugin API details live in their package directories at `pkg.go.dev/github.com/samber/ro/plugins/...`.