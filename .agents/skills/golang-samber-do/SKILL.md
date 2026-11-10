---
name: golang-samber-do
description: "Dependency injection in Golang using samber/do — service containers, lifecycle management, scopes, health checks, graceful shutdown, and module organization. Apply when using or adopting samber/do, when the codebase imports github.com/samber/do or github.com/samber/do/v2, or when refactoring manual constructor injection into a DI container."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "💉"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go architect wiring a composition root. You register against interfaces, resolve once at the edges, and let provider errors surface loudly instead of silently half-constructing the graph.

**Modes:**

- **Build** — introducing a container into a new or existing service.
- **Review** — checking auto-wiring for hidden concrete-type coupling and misplaced `Invoke` calls.
- **Debug** — resolving a failure whose error text points deep into a provider chain.

**When to use:** any task centered on samber/do DI. For the broader convenience-vs-library question, load `golang-dependency-injection` first; `golang-structs-interfaces` and `golang-testing` cover the interface and test patterns this skill relies on.

## The one rule: only the composition root touches the container

`main` (and tests that stand in for it) build the container and resolve the entry-point services. Everything downstream receives its dependencies as constructor arguments. A package that calls `do.Invoke` for every `UserService` it needs has smuggled the container halfway into the domain; push it back to startup.

## Registration

Providers are functions from the injector to a value plus an error:

```go
type Provider[T any] func(i do.Injector) (T, error)
```

```go
injector := do.New()

do.Provide(injector, func(i do.Injector) (UserStore, error) {
    db := do.MustInvoke[*sql.DB](i)
    return NewUserStore(db), nil
})

do.ProvideValue(injector, Config{Port: 8080})                  // pre-built value
do.ProvideTransient(injector, func(i do.Injector) (*Logger, error) {
    return NewLogger(), nil
})                                                              // new per resolve
```

The default is lazy: the provider runs on first resolution. `ProvideValue` and eager options exist for wiring that must exist before any consumer asks for it.

## Nesting providers

Keep providers thin: resolve dependencies, hand them to a constructor, return the result. The failure mode to avoid is provider functions that reimplement constructor logic inside the container.

```go
func NewUserService(i do.Injector) (UserService, error) {
    store := do.MustInvoke[UserStore](i)
    cache := do.MustInvoke[Cache](i)
    return &userService{store: store, cache: cache}, nil
}
```

Inside providers, use the `Must*` family. A provider already returns `(T, error)`, so an `Invoke` inside it would force a pointless `if err != nil` on every line; `MustInvoke` panics and samber/do recovers that panic at the enclosing resolution, turning it back into a regular error at the call site.

## Resolving as interfaces

Register concrete implementations and resolve them through their interface with implicit aliasing — no explicit `Provide[*pg.DB] → ProvideAs[DB]` dance needed:

```go
do.Provide(injector, func(i do.Injector) (*pg.DB, error) {
    cfg := do.MustInvoke[Config](i)
    return pg.Connect(cfg.DatabaseURL)
})

db := do.MustInvokeAs[DB](injector)
```

This keeps providers free of reflection and call sites free of concrete coupling.

## Many services of one type

Give each a name when several implementations share a type:

```go
do.ProvideNamed(injector, "primary", func(i do.Injector) (*Database, error) {
    return Open("postgres://primary"), nil
})
do.ProvideNamed(injector, "replica", func(i do.Injector) (*Database, error) {
    return Open("postgres://replica"), nil
})

mainDB := do.MustInvokeNamed[*Database](injector, "primary")
```

Named resolution is the escape hatch; default to unnamed, interface-typed registrations and reach for names only when a type genuinely coexists in multiple roles.

## Structure with packages

`do.Package` groups related providers so an entire subsystem registers in one expression:

```go
package infrastructure

var Package = do.Package(
    do.Lazy(func(i do.Injector) (*pg.DB, error) {
        cfg := do.MustInvoke[Config](i)
        return pg.Connect(cfg.DatabaseURL)
    }),
    do.Lazy(func(i do.Injector) (*redis.Client, error) {
        return redis.NewClient(cfgURL(i)), nil
    }),
)
```

The root then composes subsystems as data, not as imperative `Provide` calls:

```go
injector := do.New(infrastructure.Package, repository.Package, service.Package)
server := do.MustInvoke[*http.Server](injector)
go server.ListenAndServe()
_ = injector.ShutdownOnSignalsWithContext(context.Background(), os.Interrupt)
```

## Lifecycle and health

- Let `ShutdownOnSignalsWithContext` hang the process and drain services registered with `Shutdownerf`.
- `do.HealthCheck(injector)` walks the graph and reports each service's health; wire it into the readiness endpoint.
- Scopes isolate a container subtree — a per-request scope keeps request-scoped services from leaking into each other.

## Testing

- Clone the real injector (`Clone`) instead of rebuilding the world per test.
- Override one provider (`Override`) to swap a service for a fake.
- Reuse the same `Provider` functions in test wiring so the graph under test matches production shape.

See [advanced usage](references/advanced.md) and [testing patterns](references/testing.md) for scopes, struct injection with tags, and override/clock patterns.

## Quick reference

| Operation | Function |
| --- | --- |
| Register lazy service | `do.Provide[T]` |
| Register named lazy service | `do.ProvideNamed[T]` |
| Register pre-built value | `do.ProvideValue[T]` |
| Register transient service | `do.ProvideTransient[T]` |
| Group registrations | `do.Package` |
| Resolve with error | `do.Invoke[T]` |
| Resolve, panic on error | `do.MustInvoke[T]` |
| Resolve by interface | `do.MustInvokeAs[T]` |
| Resolve by name | `do.MustInvokeNamed[T]` |
| Resolve into struct fields | `do.MustInvokeStruct[T]` |

## Common mistakes

| Mistake | Consequence | Fix |
| --- | --- | --- |
| `Invoke` inside providers | error boilerplate everywhere, breaking the parent panic-recovery contract | use `MustInvoke` — panics are caught at the outer resolution |
| Concrete types in signatures | every consumer tight to one implementation | register concrete, resolve through interface |
| Container calls scattered in handlers | hidden global state, hard to test | resolve only at the composition root |
| Silently ignoring provider errors | broken services crash later, far from the cause | let failures propagate to `MustInvoke`/`Invoke` at startup |
| One type, unnamed conflicts | last-writer-wins in resolution | `ProvideNamed` for coexistence |
| Deep chains (5+ levels) | fragile init order, opaque failures | split responsibilities; keep trees shallow |

## Cross-references

- `golang-dependency-injection` — when a container pays off, and the comparison with manual wiring, wire, and fx.
- `golang-structs-interfaces` — interface design so resolve-by-interface stays meaningful.
- `golang-testing` — fakes, overrides, and table-driven wiring tests.
- `golang-pkg-go-dev` / `golang-gopls` — consult godig for package facts and gopls for call-site navigation around the container.
