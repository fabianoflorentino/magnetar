# Recipes — uber-go/fx

End-to-end examples beyond the `SKILL.md` basics. Each recipe is self-contained and shows a real wiring problem.

## Table of Contents

- [HTTP service with database, metrics, and graceful shutdown](#http-service-with-database-metrics-and-graceful-shutdown)
- [Background worker with graceful drain](#background-worker-with-graceful-drain)
- [Two implementations of one interface](#two-implementations-of-one-interface)
- [Supply for config and secrets](#supply-for-config-and-secrets)
- [Module-scoped decorator](#module-scoped-decorator)
- [Optional dependency for tracing](#optional-dependency-for-tracing)
- [Manual lifecycle for embedding fx in a CLI](#manual-lifecycle-for-embedding-fx-in-a-cli)
- [Event logger that filters noise](#event-logger-that-filters-noise)

## HTTP service with database, metrics, and graceful shutdown

Modules by concern, a route group, lifecycle hooks owning the server, and a real event logger:

```go
func main() {
    fx.New(
        fx.Provide(NewConfig, NewLogger, NewDatabase, NewMetricsRegistry),

        DatabaseModule,
        HTTPModule,
        MetricsModule,

        fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
            return &fxevent.ZapLogger{Logger: log}
        }),

        fx.StartTimeout(30 * time.Second),
        fx.StopTimeout(30 * time.Second),
    ).Run()
}

var DatabaseModule = fx.Module("database",
    fx.Provide(NewUserRepository, NewPostRepository),
    fx.Decorate(func(log *zap.Logger) *zap.Logger {
        return log.Named("db")
    }),
)

var HTTPModule = fx.Module("http",
    fx.Provide(
        NewRouter,
        NewHTTPServer,
        AsRoute(NewUserHandler),
        AsRoute(NewPostHandler),
        AsRoute(NewHealthHandler),
    ),
    fx.Invoke(func(*http.Server) {}), // force the server to be built
)

var MetricsModule = fx.Module("metrics",
    fx.Provide(NewPrometheusHandler),
    fx.Invoke(RegisterMetrics),
)

// Register a handler with the "routes" group.
func AsRoute(ctor any) any {
    return fx.Annotate(
        ctor,
        fx.As(new(Route)),
        fx.ResultTags(`group:"routes"`),
    )
}

type Route interface {
    Pattern() string
    http.Handler
}

type RouterParams struct {
    fx.In
    Routes []Route `group:"routes"`
}

func NewRouter(p RouterParams) *http.ServeMux {
    mux := http.NewServeMux()
    for _, r := range p.Routes {
        mux.Handle(r.Pattern(), r)
    }
    return mux
}

func NewHTTPServer(lc fx.Lifecycle, log *zap.Logger, mux *http.ServeMux, cfg *Config) *http.Server {
    srv := &http.Server{
        Addr:         cfg.Addr,
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            ln, err := net.Listen("tcp", srv.Addr)
            if err != nil {
                return fmt.Errorf("listen %s: %w", srv.Addr, err)
            }
            go func() {
                if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
                    log.Error("server error", zap.Error(err))
                }
            }()
            log.Info("listening", zap.String("addr", srv.Addr))
            return nil
        },
        OnStop: func(ctx context.Context) error {
            log.Info("shutting down")
            return srv.Shutdown(ctx)
        },
    })
    return srv
}
```

`OnStart` binds the socket and spawns the serve loop in a goroutine; the hook itself returns immediately. Shutdown runs through `srv.Shutdown(ctx)` and respects the stop timeout.

## Background worker with graceful drain

The worker loop drains in-flight work until the queue empties, then signals the stop hook:

```go
type Worker struct {
    log   *zap.Logger
    queue chan Job
    done  chan struct{}
}

func NewWorker(lc fx.Lifecycle, log *zap.Logger) *Worker {
    w := &Worker{
        log:   log,
        queue: make(chan Job, 100),
        done:  make(chan struct{}),
    }

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            go w.run()
            return nil
        },
        OnStop: func(ctx context.Context) error {
            close(w.queue) // no more jobs accepted
            select {
            case <-w.done:
                w.log.Info("worker drained cleanly")
                return nil
            case <-ctx.Done():
                w.log.Warn("worker stop timeout")
                return ctx.Err()
            }
        },
    })

    return w
}

func (w *Worker) run() {
    defer close(w.done)
    for job := range w.queue {
        job.Do(w.log)
    }
}
```

The stop hook races drain completion against the stop context — under a 30-second `fx.StopTimeout`, the worker gets exactly 30 seconds to finish; beyond that fx reports the timeout and the process exits.

## Two implementations of one interface

Named annotations keep two `Cache` implementations apart:

```go
fx.Provide(
    fx.Annotate(
        NewRedisCache,
        fx.As(new(Cache)),
        fx.ResultTags(`name:"redis"`),
    ),
    fx.Annotate(
        NewMemcachedCache,
        fx.As(new(Cache)),
        fx.ResultTags(`name:"memcached"`),
    ),
)

type ServiceParams struct {
    fx.In
    Primary  Cache `name:"redis"`
    Fallback Cache `name:"memcached"`
}
```

## Supply for config and secrets

Values built before fx runs become first-class graph members via `fx.Supply`:

```go
func main() {
    cfg := mustLoadConfig() // parsed flags + env, before fx
    secret := os.Getenv("API_KEY")

    fx.New(
        fx.Supply(cfg), // *Config available everywhere
        fx.Supply(fx.Annotate(secret, fx.ResultTags(`name:"apikey"`))),

        fx.Provide(NewLogger, NewAPIClient),
        fx.Invoke(run),
    ).Run()
}

func NewAPIClient(cfg *Config, p struct {
    fx.In
    APIKey string `name:"apikey"`
}) *APIClient {
    return &APIClient{baseURL: cfg.APIBaseURL, key: p.APIKey}
}
```

This is shorter — and clearer — than `fx.Provide(func() *Config { return cfg })`.

## Module-scoped decorator

Each module names the shared logger only for itself; no value is mutated in the parent scope:

```go
var WorkerModule = fx.Module("worker",
    fx.Provide(NewWorker, NewJobQueue),
    fx.Decorate(func(log *zap.Logger) *zap.Logger {
        return log.Named("worker")
    }),
)

var APIModule = fx.Module("api",
    fx.Provide(NewServer, NewRouter),
    fx.Decorate(func(log *zap.Logger) *zap.Logger {
        return log.Named("api")
    }),
)
```

The two modules see distinct loggers — one module's tags never leak into the other.

## Optional dependency for tracing

```go
type ServerParams struct {
    fx.In

    Logger *zap.Logger
    Tracer trace.Tracer `optional:"true"`
}

func NewServer(p ServerParams) *Server {
    s := &Server{log: p.Logger}
    if p.Tracer == nil {
        s.tracer = trace.NewNoopTracerProvider().Tracer("noop")
    } else {
        s.tracer = p.Tracer
    }
    return s
}
```

Optional makes sense here because the absence has a first-class fallback. A core service hidden behind `optional` is a nil-pointer panic dressed up as a feature.

## Manual lifecycle for embedding fx in a CLI

When fx is one component of a larger program, drive Start/Stop by hand:

```go
func runFxApp(parent context.Context) error {
    app := fx.New(
        fx.Provide(NewConfig, NewLogger, NewWorker),
        fx.Invoke(func(*Worker) {}),
    )
    if err := app.Err(); err != nil {
        return fmt.Errorf("wire: %w", err)
    }

    startCtx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    if err := app.Start(startCtx); err != nil {
        return fmt.Errorf("start: %w", err)
    }

    select {
    case <-parent.Done():
    case <-app.Done(): // SIGINT/SIGTERM
    }

    stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return app.Stop(stopCtx)
}
```

`app.Err()` validates wiring without starting — handy for a `--check`-style flag.

## Event logger that filters noise

Wrap the zap bridge and drop the per-provide chatter from production logs:

```go
type ProductionLogger struct {
    inner *fxevent.ZapLogger
}

func (l *ProductionLogger) LogEvent(e fxevent.Event) {
    switch e.(type) {
    case *fxevent.Provided, *fxevent.Supplied, *fxevent.Decorated:
        return // not useful in production
    default:
        l.inner.LogEvent(e)
    }
}

fx.New(
    fx.Provide(NewZapLogger),
    fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
        return &ProductionLogger{inner: &fxevent.ZapLogger{Logger: log}}
    }),
)
```

What remains after the filter is the lifecycle (start/stop) events and errors — the part worth auditing.
