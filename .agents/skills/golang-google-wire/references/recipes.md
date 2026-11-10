# Recipes — google/wire

End-to-end examples. Each recipe is self-contained.

## Table of Contents

- [HTTP server with Postgres and Redis](#http-server-with-postgres-and-redis)
- [Build variants: prod vs dev](#build-variants-prod-vs-dev)
- [Cleanup-heavy graph](#cleanup-heavy-graph)
- [Embedding wire in a CLI](#embedding-wire-in-a-cli)
- [Feeding external values into wire](#feeding-external-values-into-wire)

## HTTP server with Postgres and Redis

A layered service: config → DB → Redis → repo → service → transport, each layer owning its providers in a `wire.go`.

```
myapp/
├── config/       config.go, wire.go
├── infra/        db.go, cache.go, wire.go
├── repo/         user.go, wire.go
├── service/      user.go, wire.go
├── transport/    handler.go, wire.go
├── wire.go       // injector — //go:build wireinject
├── wire_gen.go   // generated — commit this
└── main.go
```

```go
// config/config.go
type Config struct{ Addr, DSN, CacheAddr string }

// config/wire.go
var ConfigSet = wire.NewSet(NewConfig)
```

```go
// infra/db.go
func NewDB(cfg *config.Config) (*sql.DB, func(), error) {
    db, err := sql.Open("postgres", cfg.DSN)
    if err != nil {
        return nil, nil, err
    }
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, nil, err
    }
    return db, func() { db.Close() }, nil
}

// infra/cache.go
func NewRedis(cfg *config.Config) (*redis.Client, func(), error) {
    c := redis.NewClient(&redis.Options{Addr: cfg.CacheAddr})
    if err := c.Ping(context.Background()).Err(); err != nil {
        return nil, nil, err
    }
    return c, func() { c.Close() }, nil
}

// infra/wire.go
var InfraSet = wire.NewSet(NewDB, NewRedis)
```

```go
// repo/user.go
type UserStore interface {
    GetUser(ctx context.Context, id int64) (*User, error)
}

type PostgresUserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *PostgresUserRepo { return &PostgresUserRepo{db: db} }

// repo/wire.go
var RepoSet = wire.NewSet(NewUserRepo, wire.Bind(new(UserStore), new(*PostgresUserRepo)))
```

```go
// service/user.go
type UserService struct {
    store repo.UserStore
    cache *redis.Client
}

// service/wire.go
var ServiceSet = wire.NewSet(NewUserService)
```

```go
// wire.go
//go:build wireinject

package main

func InitApp() (*transport.Handler, func(), error) {
    wire.Build(
        config.ConfigSet,
        infra.InfraSet,
        repo.RepoSet,
        service.ServiceSet,
        transport.NewHandler,
    )
    return nil, nil, nil
}
```

```go
// main.go
func main() {
    handler, cleanup, err := InitApp()
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    srv := &http.Server{Addr: ":8080", Handler: handler}
    log.Fatal(srv.ListenAndServe())
}
```

## Build variants: prod vs dev

Different provider sets behind the same injector name, selected by build constraints on the injector files:

```go
// wire_prod.go
//go:build wireinject && !dev

package main

func InitApp() (*App, func(), error) {
    wire.Build(ProdSet, NewApp)
    return nil, nil, nil
}

// wire_dev.go
//go:build wireinject && dev

package main

func InitApp() (*App, func(), error) {
    wire.Build(DevSet, NewApp) // DevSet swaps the real DB for in-memory SQLite
    return nil, nil, nil
}
```

Generate each variant into its own file, because both passes would otherwise overwrite the same `wire_gen.go`:

```bash
wire -tags dev -output_file_prefix=wire_gen_dev gen .
wire -output_file_prefix=wire_gen_prod gen .
```

Then give the generated files matching build constraints so only one compiles per build:

```go
// wire_gen_prod.go — top of generated file
//go:build !dev

// wire_gen_dev.go — top of generated file
//go:build dev
```

Commit both generated files; the build compiles exactly one.

## Cleanup-heavy graph

Wire's reverse-order cleanup pays off when several resources need coordinated shutdown:

```go
func NewDBPool(cfg *Config) (*pgxpool.Pool, func(), error) {
    pool, err := pgxpool.New(context.Background(), cfg.DSN)
    if err != nil {
        return nil, nil, err
    }
    return pool, func() { pool.Close() }, nil
}

func NewOTelExporter(cfg *Config) (*otlptrace.Exporter, func(), error) {
    exp, err := otlptracegrpc.New(context.Background())
    if err != nil {
        return nil, nil, err
    }
    return exp, func() { exp.Shutdown(context.Background()) }, nil
}

func NewTracerProvider(exp *otlptrace.Exporter) (*trace.TracerProvider, func(), error) {
    tp := trace.NewTracerProvider(trace.WithBatcher(exp))
    return tp, func() { tp.Shutdown(context.Background()) }, nil
}
```

Shutdown runs `TracerProvider` → `Exporter` → `DBPool`, so in-flight spans flush before the exporter closes, and the exporter closes before the pool is released.

## Embedding wire in a CLI

Wire emits a value, not an app framework — the lifecycle stays yours. Signal handling and graceful shutdown are explicit:

```go
// wire.go
//go:build wireinject

package cmd

func InitServer(cfg *Config) (*http.Server, func(), error) {
    wire.Build(InfraSet, ServiceSet, NewHTTPServer)
    return nil, nil, nil
}

// cmd/serve.go
func runServe(cfg *Config) error {
    srv, cleanup, err := InitServer(cfg)
    if err != nil {
        return err
    }
    defer cleanup()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    return srv.Shutdown(ctx)
}
```

Unlike `fx.Run()`, there is no managed run loop — a deliberate trade for short-lived tools that need precise control over the shutdown sequence.

## Feeding external values into wire

Values built outside the graph become injector parameters; wire resolves the parameter as a provided value of its type.

```go
//go:build wireinject

// cfg is resolved externally — treated as an already-provided *Config
func InitApp(cfg *Config) (*App, func(), error) {
    wire.Build(InfraSet, ServiceSet, NewApp)
    return nil, nil, nil
}

// main.go
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
app, cleanup, err := InitApp(cfg)
```

No `wire.Value` or extra set entry is needed for the parameter's type.
