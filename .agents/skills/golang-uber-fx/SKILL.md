---
name: golang-uber-fx
description: "Golang application framework using uber-go/fx — fx.New, fx.Provide, fx.Invoke, fx.Module, fx.Lifecycle hooks, fx.Annotate (name/group/As), fx.Decorate, fx.Supply, fx.Replace, fx.WithLogger, and signal-aware Run(). Apply when using or adopting uber-go/fx, when the codebase imports `go.uber.org/fx`, or when wiring services with fx.New. For raw DI without lifecycle, see `fabianoflorentino/golang-agent-skills@golang-uber-dig` skill."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🏭"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go architect building long-running services. You wire the graph in one place, put lifecycle in hooks rather than `init()`, and treat modules as the unit of reuse.

**Modes:**

- **Build** — standing up an fx application or adding a module.
- **Review** — auditing hook bodies for blocking work, missing flush, and decorator scope leaks.

**When to use:** any task centered on uber-go/fx. When only a wiring graph is needed — no lifecycle, no signals — drop to `golang-uber-dig`, the engine fx wraps.

## The application skeleton

```go
app := fx.New(
    fx.Provide(NewConfig, NewDatabase, NewServer),
    fx.Invoke(RegisterRoutes),
)
app.Run() // blocks on SIGINT/SIGTERM, then drains OnStop hooks
```

Boot sequence: `fx.New` validates constructor signatures without running them; `app.Start` runs every `fx.Invoke` and fires OnStart hooks in dependency order; `app.Stop` fires OnStop in reverse order. Default timeout for the whole cycle is 15 seconds — tune with `fx.StartTimeout`/`fx.StopTimeout`.

A type is only ever constructed if some `Invoke` (directly or transitively) pulls it.

## Lifecycle hooks

Inject `fx.Lifecycle` into a constructor and append hooks. Constructors should be cheap; anything long-running belongs in `OnStart`:

```go
func NewHTTPServer(lc fx.Lifecycle, addr string) *http.Server {
    srv := &http.Server{Addr: addr}
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            ln, err := net.Listen("tcp", addr)
            if err != nil {
                return err
            }
            go srv.Serve(ln) // blocking work goes in a goroutine
            return nil
        },
        OnStop: func(ctx context.Context) error {
            return srv.Shutdown(ctx)
        },
    })
    return srv
}
```

The hook contracts: OnStart returns quickly (spawned goroutines do the work) or later hooks starve; both callbacks get a context bounded by the startup/shutdown timeouts, so respect `ctx.Done()`. `fx.StartHook`/`fx.StopHook`/`fx.StartStopHook` adapt simpler signatures.

## Parameter and result objects

`fx.In`/`fx.Out` re-export `dig.In`/`dig.Out`. Needed once a constructor takes 4+ deps or needs tagged lookups:

```go
type ServerParams struct {
    fx.In

    Logger *zap.Logger
    DB     *sql.DB
    Cache  *redis.Client  `optional:"true"`
    Routes []http.Handler `group:"routes"`
}
```

## Tagging constructors with fx.Annotate

`Annotate` attaches names, groups, or interface bindings without an `fx.Out` struct — keeping the constructor itself plain and reuseable:

```go
fx.Provide(
    fx.Annotate(NewPrimaryDB, fx.ResultTags(`name:"primary"`)),
    fx.Annotate(NewPostgresDB, fx.As(new(Database))),
    fx.Annotate(NewUserHandler, fx.As(new(http.Handler)), fx.ResultTags(`group:"routes"`)),
)
```

## Value groups

Multiple constructors feed one consumer slice — routes, health checks, collectors:

```go
type RouteResult struct {
    fx.Out
    Handler http.Handler `group:"routes"`
}
```

Consumers take `Routes []http.Handler` with the `group:"routes"` tag. Append `,flatten` to unwrap slice-returning providers. Group order is not guaranteed; supply an ordered slice when sequence matters.

## Modules

`fx.Module(name, ...)` bundles providers, invokes, and decorators under one name — and scopes decorators to the module and its children:

```go
var DBModule = fx.Module("database",
    fx.Provide(NewConnection, NewUserRepository),
    fx.Decorate(func(log *zap.Logger) *zap.Logger {
        return log.Named("db")
    }),
)

fx.New(fx.Provide(NewConfig, NewLogger), DBModule).Run()
```

The logger rename stays inside `database`; a top-level `fx.Decorate` would leak everywhere. Think of a module as a liftable library whose public surface is its provided types.

## Supply, Replace, Decorate

- `fx.Supply(value, ...)` registers pre-built values (config, flags) — clearer than a no-op constructor.
- `fx.Replace` swaps an existing type at the root — the standard test fake hook.
- `fx.Decorate` wraps a provider's output for all consumers in scope.

## Best practices

1. Keep `main` to providers, modules, and `Run()`; activity lives in modules so each is testable alone.
2. Lifecycle hooks over `init()` and constructor-spawned goroutines — ordering follows graph topology, not import order.
3. `OnStart` must return promptly; block in a goroutine.
4. Respect `ctx.Done()` in hooks; an ignored cancellation times out the boot but leaks the goroutine.
5. Group by concern (HTTP, DB, metrics), each owning its providers and lifecycle.
6. Prefer `Annotate` over hand-written `fx.Out` structs to keep constructors fx-agnostic.
7. Use `Supply` for config and flags instead of `Provide`-ing trivial constructors.
8. Validate the graph in CI with `fx.New(...).Err()` — catches missing providers and cycles before deploy.

## Common mistakes

| Mistake | Failure | Fix |
| --- | --- | --- |
| Long work in OnStart | downstream hooks wait forever | goroutine inside the hook |
| `Provide` for pre-built values | no-op constructor noise | `fx.Supply` |
| Top-level `Decorate` | leaks to every consumer | scope it inside `fx.Module` |
| Assuming group order | wrong mount order | explicit ordered slice |
| Side effects in constructors | racey, lazy-executed behavior | move to OnStart |
| No `fx.Invoke` | nothing ever constructs | at least one invoke per app |

## Testing

`go.uber.org/fx/fxtest` bridges fx and `*testing.T` (failures call `t.Fatal`; `RequireStop` registers cleanup). `fx.Populate(&target)` extracts values from the graph; `fx.Replace` swaps real deps for fakes. Full patterns in [testing reference](references/testing.md).

## Advanced material

[advanced reference](references/advanced.md) — Supply/Replace/Decorate, optional deps, custom event logging, manual lifecycle for CLI embedding; [recipes reference](references/recipes.md) — end-to-end HTTP services, background workers with graceful drain, multiple implementations of one interface.

Library bugs: [uber-go/fx issues](https://github.com/uber-go/fx/issues).

## Cross-references

- `golang-uber-dig` — the underlying container and `In`/`Out` tags.
- `golang-dependency-injection` — DI concepts and library comparisons.
- `golang-samber-do` — generics-based alternative without reflection.
- `golang-google-wire` — compile-time DI, no runtime container.
- `golang-structs-interfaces` — interface design for constructor signatures.
- `golang-context` — cancellation discipline in OnStart/OnStop hooks.
- `golang-testing` — general testing patterns.
