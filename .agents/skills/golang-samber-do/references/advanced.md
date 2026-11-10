# samber/do — Advanced Wiring

Registration and resolution primitives are covered in the owning SKILL. This page covers scoped subtrees, explicit aliasing, struct injection, lifecycle hooks, and container introspection.

## Scopes

`injector.Scope(name)` creates a child injector. A scope resolves everything registered in its ancestors, owns its own registrations, and stays invisible to sibling scopes:

```go
root := do.New()
do.Provide(root, NewConfig)

api := root.Scope("api")
do.Provide(api, func(i do.Injector) (UserService, error) {
    cfg := do.MustInvoke[Config](i) // resolved from root
    return NewUserService(cfg), nil
})

request := root.Scope("request")
do.Provide(request, NewRequestContext)
```

Organize scopes by lifetime and visibility: stateless services live in the root, per-request services in a scope created per request. Keep sibling scopes free of cross-references.

## Explicit aliasing

Implicit interface resolution (`do.InvokeAs`/`do.MustInvokeAs`) covers most cases. When legacy code must also resolve a concrete type through an interface, register the alias explicitly:

```go
do.Provide(injector, func(i do.Injector) (*PostgreSQLDatabase, error) {
    return NewPostgreSQLDatabase(), nil
})
do.MustAs[*PostgreSQLDatabase, Database](injector)

primary := do.MustInvoke[*PostgreSQLDatabase](injector)
iface := do.MustInvoke[Database](injector)
```

Both resolves now succeed. `do.As[T, U]` is the fallible variant; `do.AsNamed[T, U]` registers a named alias for the same pair.

## Struct injection

`do.MustInvokeStruct[T]` populates exported fields carrying a `do:""` tag. An empty tag resolves by type; a non-empty tag resolves by service name:

```go
type App struct {
    Database *Database `do:""`
    Logger   *Logger   `do:"app-logger"`
    Config   *Config   `do:""`
}

app := do.MustInvokeStruct[App](injector)
```

Use struct injection only at the composition root to assemble aggregates; keep domain code on constructor parameters.

## Health checks

A service reports health by implementing `Healthchecker` — either `HealthCheck() error` or the context-aware `HealthCheck(ctx) error`:

```go
func (d *Database) HealthCheck(ctx context.Context) error {
    return d.conn.PingContext(ctx)
}
```

`do.HealthCheck[T](injector)` checks one service; `HealthCheckNamed` targets a named service and `do.HealthCheckWithContext[T]` bounds the check. Wire the result into the readiness endpoint:

```go
if err := do.HealthCheck[Database](injector); err != nil {
    log.Printf("database unhealthy: %v", err)
}
```

## Graceful shutdown

Services clean up by implementing `Shutdowner`. The interface accepts four shapes — with or without a context, with or without an error — so standardize on the context-plus-error variant:

```go
func (d *Database) Shutdown(ctx context.Context) error { return d.conn.Close() }
```

Bound the teardown so a stuck service cannot hang process exit:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
report := injector.ShutdownWithContext(ctx)
```

## Debugging

Introspection beats guessing when a resolution fails:

- `injector.ListProvidedServices()` — registrations with their scope names.
- `injector.ListInvokedServices()` — only the services actually resolved.
- `do.ExplainInjector(injector)` — rendered scope tree for the whole container.
- `do.ExplainService[T]()` — dependencies and status of a single service.
- `do.NameOf[T]()` — generated name of a service; depend on it sparingly.

```go
for _, svc := range injector.ListProvidedServices() {
    fmt.Printf("%s: %s\n", svc.ScopeName, svc.Service)
}
explanation := do.ExplainInjector(injector)
fmt.Println(explanation.String())
```

## Migration from manual DI

Manual constructor threading moves wholesale into providers — each provider is the old constructor with its arguments supplied by `do.MustInvoke`:

```go
func main() {
    injector := do.New()
    do.Provide(injector, func(i do.Injector) (*Config, error) {
        return &Config{Port: 8080}, nil
    })
    do.Provide(injector, NewDatabase)
    do.Provide(injector, NewUserRepository)
    do.Provide(injector, NewUserService)

    api := do.MustInvoke[*API](injector)
}
```

The composition root shrinks to container building plus one resolve.

## Quick reference

| Operation | Entry point |
| --- | --- |
| Root container | `do.New()` / `do.NewWithOpts()` |
| Child scope | `injector.Scope(name)` |
| Type alias | `do.As[T, U]` / `do.AsNamed[T, U]` |
| Health | `do.HealthCheck[T]`, `do.HealthCheckNamed`, `do.HealthCheckWithContext[T]` |
| Shutdown | `do.Shutdown[T]`, `do.ShutdownNamed`, `do.ShutdownWithContext[T]`, `do.MustShutdown[T]` |
| Introspection | `do.ExplainInjector`, `do.ExplainService[T]`, `do.NameOf[T]`, `injector.ListProvidedServices`, `injector.ListInvokedServices` |