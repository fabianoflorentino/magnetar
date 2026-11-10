# Recipes — uber-go/dig

End-to-end examples beyond the `SKILL.md` basics. Each recipe is self-contained and models a real wiring problem.

## Table of Contents

- [HTTP server with a route group](#http-server-with-a-route-group)
- [Read-write and read-only databases](#read-write-and-read-only-databases)
- [Publishing an interface with `dig.As`](#publishing-an-interface-with-digas)
- [Request-scoped dependencies](#request-scoped-dependencies)
- [Graceful degradation with an optional dependency](#graceful-degradation-with-an-optional-dependency)
- [Cross-cutting behavior with `Decorate`](#cross-cutting-behavior-with-decorate)
- [DryRun validation in tests](#dryrun-validation-in-tests)
- [Visualizing a failed graph](#visualizing-a-failed-graph)

## HTTP server with a route group

Handlers contribute one entry each to a `routes` group; the server consumes every member:

```go
type Route struct {
    Pattern string
    Handler http.Handler
}

type RouteResult struct {
    dig.Out
    Route Route `group:"routes"`
}

func NewHealthRoute() RouteResult {
    return RouteResult{Route: Route{
        Pattern: "/health",
        Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
            w.WriteHeader(http.StatusOK)
        }),
    }}
}

func NewUserRoute(repo *UserRepo) RouteResult {
    return RouteResult{Route: Route{
        Pattern: "/users",
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            users, err := repo.List(r.Context())
            if err != nil {
                http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
                return
            }
            fmt.Fprintf(w, "%d users", len(users))
        }),
    }}
}

type ServerParams struct {
    dig.In
    Routes []Route `group:"routes"`
}

func NewServer(p ServerParams) *http.Server {
    mux := http.NewServeMux()
    for _, r := range p.Routes {
        mux.Handle(r.Pattern, r.Handler)
    }
    return &http.Server{Addr: ":8080", Handler: mux}
}

func main() {
    c := dig.New()

    must(c.Provide(NewDB))
    must(c.Provide(NewUserRepo))
    must(c.Provide(NewHealthRoute)) // joins the group
    must(c.Provide(NewUserRoute))   // joins the group
    must(c.Provide(NewServer))

    err := c.Invoke(func(srv *http.Server) error {
        log.Println("listening on", srv.Addr)
        return srv.ListenAndServe()
    })
    if err != nil {
        log.Fatal(err)
    }
}

func must(err error) {
    if err != nil {
        panic(err)
    }
}
```

Group members arrive in no defined order; if mount order matters, build one ordered slice in a single constructor.

## Read-write and read-only databases

Two connections to the same type need names:

```go
type DBResult struct {
    dig.Out
    Primary  *sql.DB `name:"primary"`
    ReadOnly *sql.DB `name:"readonly"`
}

func NewDatabases(cfg *Config) (DBResult, error) {
    rw, err := sql.Open("postgres", cfg.PrimaryDSN)
    if err != nil {
        return DBResult{}, fmt.Errorf("primary: %w", err)
    }
    ro, err := sql.Open("postgres", cfg.ReadOnlyDSN)
    if err != nil {
        rw.Close()
        return DBResult{}, fmt.Errorf("readonly: %w", err)
    }
    return DBResult{Primary: rw, ReadOnly: ro}, nil
}

type RepoParams struct {
    dig.In
    Writer *sql.DB `name:"primary"`
    Reader *sql.DB `name:"readonly"`
}

func NewUserRepo(p RepoParams) *UserRepo {
    return &UserRepo{w: p.Writer, r: p.Reader}
}
```

Closing the already-open connection on the second failure keeps the partial graph clean.

## Publishing an interface with `dig.As`

Consumers ask for `Cache`; the concrete type and its internals stay private:

```go
type Cache interface {
    Get(key string) (string, bool)
    Set(key, value string)
}

type RedisCache struct {
    client  *redis.Client
    metrics *Metrics // internal — consumers should never see this
}

func NewRedisCache(client *redis.Client, m *Metrics) *RedisCache {
    return &RedisCache{client: client, metrics: m}
}

func (c *RedisCache) Get(key string) (string, bool) { /* ... */ }
func (c *RedisCache) Set(key, value string)         { /* ... */ }

func main() {
    c := dig.New()
    must(c.Provide(NewRedisClient))
    must(c.Provide(NewMetrics))
    must(c.Provide(NewRedisCache, dig.As(new(Cache))))
    must(c.Invoke(func(cache Cache) {
        cache.Set("hello", "world")
    }))
}
```

## Request-scoped dependencies

A child scope inherits the shared providers and adds per-request ones:

```go
root := dig.New()
must(root.Provide(NewLogger))
must(root.Provide(NewDB))
must(root.Provide(NewHandler)) // shared; the scope inherits it

func handle(w http.ResponseWriter, req *http.Request) {
    scope := root.Scope("request")

    must(scope.Provide(func() *http.Request { return req }))
    must(scope.Provide(func() RequestID { return RequestID(req.Header.Get("X-Request-ID")) }))
    must(scope.Decorate(func(l *zap.Logger) *zap.Logger {
        return l.With(zap.String("request_id", req.Header.Get("X-Request-ID")))
    }))

    err := scope.Invoke(func(h *Handler) error {
        return h.Serve(w, req)
    })
    if err != nil {
        http.Error(w, err.Error(), 500)
    }
}
```

The decorator applies only inside this request scope — every in-flight request keeps its own logger.

## Graceful degradation with an optional dependency

```go
type WorkerParams struct {
    dig.In

    DB     *sql.DB
    Tracer trace.Tracer `optional:"true"` // app still boots without OTel
}

func NewWorker(p WorkerParams) *Worker {
    w := &Worker{db: p.DB}
    if p.Tracer != nil {
        w.tracer = p.Tracer
    } else {
        w.tracer = trace.NewNoopTracerProvider().Tracer("noop")
    }
    return w
}
```

Mark a dependency optional only when its absence is a supported state — hiding a missing DB behind `optional` just defers the failure to a nil-pointer panic at first use.

## Cross-cutting behavior with `Decorate`

One decorator wraps a value for every consumer in scope:

```go
// Wrap the *sql.DB with a metrics-recording proxy everywhere.
must(c.Decorate(func(db *sql.DB, m *Metrics) *sql.DB {
    return wrapWithMetrics(db, m)
}))

// Tag the logger with service and environment.
must(c.Decorate(func(log *zap.Logger, cfg *Config) *zap.Logger {
    return log.With(
        zap.String("service", cfg.ServiceName),
        zap.String("env", cfg.Env),
    )
}))
```

Decorators are scope-local: one on the root applies everywhere, one on a child scope only to that subtree.

## DryRun validation in tests

```go
func TestWiringIsValid(t *testing.T) {
    c := dig.New(dig.DryRun(true))

    must := func(err error) { require.NoError(t, err) }
    must(c.Provide(NewConfig))
    must(c.Provide(NewLogger))
    must(c.Provide(NewDB))
    must(c.Provide(NewServer))

    // Validates types without executing any constructor.
    require.NoError(t, c.Invoke(func(*http.Server) {}))
}
```

This catches "no provider for *X" in CI instead of on a production startup.

## Visualizing a failed graph

```go
err := c.Invoke(run)
if err != nil {
    f, _ := os.Create("graph.dot")
    defer f.Close()
    _ = dig.Visualize(c, f, dig.VisualizeError(err))
    log.Fatalf("wiring failed (graph in graph.dot): %v", err)
}
// Render: dot -Tpng graph.dot -o graph.png
```

`VisualizeError` highlights the missing edges — far faster than tracing the wrapped error chain by hand.
