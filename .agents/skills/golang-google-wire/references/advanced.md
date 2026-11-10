# Advanced — google/wire

Detail topics referenced from `SKILL.md`. Each section is self-contained.

## Table of Contents

- [Cleanup chains](#cleanup-chains)
- [Multiple injectors in one package](#multiple-injectors-in-one-package)
- [Nesting provider sets](#nesting-provider-sets)
- [The `wire:"-"` exclusion tag](#the-wire--exclusion-tag)
- [Codegen errors](#codegen-errors)
- [Codegen flags](#codegen-flags)
- [`panic(wire.Build(...))` as injector body](#panicwirebuild-as-injector-body)
- [External values as injector arguments](#external-values-as-injector-arguments)
- [Quick reference](#quick-reference)

## Cleanup chains

A provider that returns `(T, func(), error)` hands wire a cleanup. The generated injector runs cleanups in **reverse construction order**: the newest dependant is torn down before the dependency that built it.

```go
func NewDB(cfg *Config) (*sql.DB, func(), error) {
    db, err := sql.Open("postgres", string(cfg.DSN))
    if err != nil {
        return nil, nil, err
    }
    return db, func() { db.Close() }, nil
}

func NewCache(cfg *Config) (*redis.Client, func(), error) {
    c := redis.NewClient(&redis.Options{Addr: cfg.CacheAddr})
    return c, func() { c.Close() }, nil
}
```

Wire emits a chain that also releases the resources already built when construction stops halfway, and otherwise wires every cleanup behind the final value in reverse build order:

```go
func InitApp() (*App, func(), error) {
    cfg := NewConfig()
    db, dbCleanup, err := NewDB(cfg)
    if err != nil {
        return nil, nil, err
    }
    cache, cacheCleanup, err := NewCache(cfg)
    if err != nil {
        dbCleanup() // partial graph: only the built components are cleaned up
        return nil, nil, err
    }
    app := NewApp(db, cache)
    return app, func() {
        cacheCleanup() // last built, released first
        dbCleanup()
    }, nil
}
```

A successful construction always returns a non-nil cleanup; a mid-way failure returns nil. Guard the caller before deferring:

```go
app, cleanup, err := InitApp()
if err != nil {
    log.Fatal(err)
}
if cleanup != nil {
    defer cleanup()
}
```

## Multiple injectors in one package

A package may hold any number of injectors, each in its own `//go:build wireinject` file. Every generated body lands in the single `wire_gen.go`.

```go
//go:build wireinject

package main

func InitProdApp() (*App, func(), error) {
    wire.Build(ProdSet, NewApp)
    return nil, nil, nil
}

func InitDevApp() (*App, func(), error) {
    wire.Build(DevSet, NewApp)
    return nil, nil, nil
}
```

Pick the variant at runtime with a flag, or at build time by tagging the injector files `//go:build prod` and `//go:build !prod`.

## Nesting provider sets

Sets embed other sets, which lets the hierarchy mirror package boundaries:

```go
// pkg/config/wire.go
var ConfigSet = wire.NewSet(NewConfig)

// pkg/infra/wire.go
var InfraSet = wire.NewSet(config.ConfigSet, NewDB, NewCache)

// pkg/service/wire.go
var ServiceSet = wire.NewSet(NewUserService, wire.Bind(new(UserStore), new(*UserRepo)))

// injector
wire.Build(infra.InfraSet, service.ServiceSet, NewApp)
```

A shared set is a compatibility contract. Safe changes for a release: replace a provider with one taking equal or fewer inputs, or add an output type no set has provided before. Breaking changes — a new required provider input, a removed output type, or a type the injector already builds — fail every downstream injector and need coordination.

## The `wire:"-"` exclusion tag

`wire.Struct` fills the named fields of a struct (or every field with `"*"`). Tagging a field `wire:"-"` opts it out of the `"*"` sweep. Unexported fields are always skipped, with or without the tag.

```go
type Server struct {
    Logger  *zap.Logger
    DB      *sql.DB
    mu      sync.Mutex    `wire:"-"`
    Timeout time.Duration `wire:"-"`
}

wire.Struct(new(Server), "*") // injects Logger and DB; skips mu and Timeout
```

## Codegen errors

| Error | Cause | Fix |
| --- | --- | --- |
| `no provider found for TYPE` | no set in `wire.Build` supplies it | add the missing provider or set |
| `multiple bindings for TYPE` | two providers return the same type | use named types or drop one |
| `argument N has no provider for TYPE` | interface requested without a binding | `wire.Bind(new(Iface), new(*Impl))` |
| `cycle detected` | A → B → A dependency loop | break it with an interface or a factory |
| `wire.Build used outside of injector function` | called from ordinary code | only call it inside a tagged injector body |
| duplicate symbol / redeclared | injector file missing its build tag | `//go:build wireinject` first line |

## Codegen flags

```bash
# Output file name prefix (default: wire_gen)
wire -output_file_prefix=init gen ./cmd/server

# Honor build tags while resolving providers
wire -tags=integration gen ./...

# Prepend a header file (e.g. license boilerplate) to generated output
wire -header_file=hack/boilerplate.go.txt gen ./...
```

## `panic(wire.Build(...))` as injector body

The injector body may consist of a single `panic(wire.Build(...))` instead of zero-value returns — handy when the return types are awkward to initialize. Wire recognizes the pattern and replaces the body during codegen; the panic never executes:

```go
func InitApp(ctx context.Context) (*App, func(), error) {
    panic(wire.Build(AppSet))
}
```

## External values as injector arguments

Values that exist before wire runs — parsed flags, already-loaded config, a test double — become injector parameters. Wire treats every parameter as a pre-built value of its type; no set entry is needed.

```go
//go:build wireinject

func InitApp(cfg *Config) (*App, func(), error) {
    wire.Build(InfraSet, ServiceSet, NewApp)
    return nil, nil, nil
}

// main.go
cfg := parseFlags()
app, cleanup, err := InitApp(cfg)
```

## Quick reference

| Symbol | Purpose |
| --- | --- |
| `wire.NewSet(providers...)` | bundle providers into a reusable set |
| `wire.Build(sets...)` | declare the injector body (codegen replaces it) |
| `wire.Bind(new(I), new(*C))` | bind an interface to a concrete type |
| `wire.Struct(new(T), "F", ...)` | inject named struct fields from the graph |
| `wire.Struct(new(T), "*")` | inject every non-`wire:"-"` field |
| `wire.Value(expr)` | supply a constant expression |
| `wire.InterfaceValue(new(I), v)` | supply a value cast to an interface |
| `wire.FieldsOf(new(T), "F", ...)` | promote struct fields as graph nodes |
| `//go:build wireinject` | keep the injector stub out of the binary |
| `wire_gen.go` | generated output — commit, never edit |
| `wire ./...` | regenerate every injector in the module |
| `wire check ./...` | validate the graph without regenerating |
