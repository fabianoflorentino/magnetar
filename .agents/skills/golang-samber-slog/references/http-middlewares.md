# HTTP Middlewares

The `slog-gin`, `slog-echo`, `slog-fiber`, `slog-chi`, and `slog-http` packages share one design: a `Config` struct, `New(logger)` / `NewWithConfig(logger, cfg)` entry points, filter helpers, and per-request custom attributes.

## Config fields

| Field | Default | Purpose |
| --- | --- | --- |
| `DefaultLevel` | `slog.LevelInfo` | Level for 2xx/3xx responses |
| `ClientErrorLevel` | `slog.LevelWarn` | Level for 4xx responses |
| `ServerErrorLevel` | `slog.LevelError` | Level for 5xx responses |
| `WithUserAgent` | `false` | Include the `User-Agent` header |
| `WithRequestID` | `false` | Include the request ID |
| `WithRequestBody` | `false` | Include the request body (capped) |
| `WithResponseBody` | `false` | Include the response body (capped) |
| `WithRequestHeader` | `false` | Include request headers |
| `WithResponseHeader` | `false` | Include response headers |
| `WithSpanID` / `WithTraceID` | `false` | Include OpenTelemetry span / trace ID |
| `WithClientIP` | `false` | Include the client IP |
| `Filters` | `nil` | Request filters |

Package-level globals tune body capture and redaction before middleware creation: `RequestBodyMaxSize` / `ResponseBodyMaxSize` (64 KB each by default), `HiddenRequestHeaders` / `HiddenResponseHeaders` for redaction, and `TraceIDKey` / `SpanIDKey` for the OTel context keys.

Every middleware logs `method`, `path`, `status`, `latency`, `request-length`, and `response-length` by default.

## Gin — `slog-gin`

```go
import sloggin "github.com/samber/slog-gin"

router := gin.New()
router.Use(sloggin.NewWithConfig(logger, sloggin.Config{
    DefaultLevel:     slog.LevelInfo,
    ClientErrorLevel: slog.LevelWarn,
    ServerErrorLevel: slog.LevelError,
    WithRequestBody:  true,
    WithUserAgent:    true,
    Filters: []sloggin.Filter{
        sloggin.IgnorePath("/health", "/metrics"),
        sloggin.IgnorePathPrefix("/static"),
    },
}))

// Per-request attributes
router.GET("/api/users", func(c *gin.Context) {
    sloggin.AddCustomAttributes(c, slog.String("user_id", userID))
    c.JSON(http.StatusOK, users)
})
```

## Echo — `slog-echo`

```go
import slogecho "github.com/samber/slog-echo"

e := echo.New()
e.Use(slogecho.NewWithConfig(logger, slogecho.Config{
    DefaultLevel:     slog.LevelInfo,
    ClientErrorLevel: slog.LevelWarn,
    ServerErrorLevel: slog.LevelError,
    Filters: []slogecho.Filter{
        slogecho.IgnoreStatus(404),
        slogecho.IgnorePath("/health"),
    },
}))

e.GET("/api/users", func(c echo.Context) error {
    slogecho.AddCustomAttributes(c, slog.String("user_id", userID))
    return c.JSON(http.StatusOK, users)
})
```

## Fiber — `slog-fiber`

```go
import slogfiber "github.com/samber/slog-fiber"

app := fiber.New()
app.Use(slogfiber.NewWithConfig(logger, slogfiber.Config{
    DefaultLevel:     slog.LevelInfo,
    ClientErrorLevel: slog.LevelWarn,
    ServerErrorLevel: slog.LevelError,
    Filters: []slogfiber.Filter{
        slogfiber.IgnorePath("/health"),
        slogfiber.IgnoreStatus(404),
    },
}))

app.Get("/api/users", func(c fiber.Ctx) error {
    slogfiber.AddCustomAttributes(c, slog.String("user_id", userID))
    return c.JSON(users)
})
```

Fiber runs on `fasthttp`, not `net/http`, so its request/response types differ from the other middlewares.

## Chi — `slog-chi`

```go
import slogchi "github.com/samber/slog-chi"

router := chi.NewRouter()
router.Use(slogchi.NewWithConfig(logger, slogchi.Config{
    DefaultLevel:     slog.LevelInfo,
    ClientErrorLevel: slog.LevelWarn,
    ServerErrorLevel: slog.LevelError,
    Filters: []slogchi.Filter{
        slogchi.IgnorePath("/health", "/ready"),
        slogchi.IgnoreStatus(401, 404),
    },
}))

router.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
    slogchi.AddCustomAttributes(r, slog.String("user_id", userID))
    json.NewEncoder(w).Encode(users)
})
```

## net/http — `slog-http`

Wraps any `http.Handler`:

```go
import sloghttp "github.com/samber/slog-http"

handler := sloghttp.New(logger)(mux)
http.ListenAndServe(":8080", handler)
```

## Filters

All middlewares share the filter helpers:

```go
sloggin.IgnorePath("/health", "/metrics")      // exact path match
sloggin.IgnorePathPrefix("/static", "/assets") // path prefix
sloggin.IgnoreStatus(401, 404)                 // skip specific statuses

// Custom filter
sloggin.Accept(func(c *gin.Context) bool {
    return c.Request.Method != "OPTIONS" // skip CORS preflight
})
```

## Logger grouping

Wrap the logger with `WithGroup("http")` to namespace all middleware attributes:

```go
router.Use(sloggin.New(logger.WithGroup("http")))
// {"http": {"method": "GET", "path": "/api", "status": 200, ...}}
```