---
name: golang-uber-dig
description: "Implements dependency injection in Golang using uber-go/dig — reflection-based container, Provide/Invoke, dig.In/dig.Out parameter and result objects, named values, value groups, optional dependencies, scopes, and Decorate. Apply when using or adopting uber-go/dig, when the codebase imports `go.uber.org/dig`, or when wiring an application graph at startup. For higher-level lifecycle and modules, see `fabianoflorentino/golang-agent-skills@golang-uber-fx` skill."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "⛏"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go architect who wires a graph at the composition root. You depend on interfaces, register with constructor functions, and let dig surface missing providers at startup instead of at first request.

**Modes:**

- **Build** — assembling an object graph with dig.
- **Review** — auditing `dig` usage for container leakage, name collisions, and unordered group assumptions.

**When to use:** any task centered on uber-go/dig — the wiring engine behind `golang-uber-fx`. Prefer that skill when the target needs lifecycle, modules, and signal handling; reach for dig alone for CLIs, libraries, and test harnesses.

## How dig works

`dig` is a reflection-based container. Register constructors with `Provide`; pull values with `Invoke` — by naming the types you want as function parameters. Constructors are lazy and memoized: a type is built at most once per container.

```go
c := dig.New()

must(c.Provide(func(cfg *Config) (*sql.DB, error) {
    return sql.Open("postgres", cfg.DSN)
}))

err := c.Invoke(func(db *sql.DB) error { return db.Ping() })
```

A constructor is any function whose parameters are dependencies and whose results are provided types; `error` as the last return signals construction failure. dig wraps returned errors with the dependency path that triggered them.

## Grouping 4+ deps: `dig.In`

Past a handful of parameters, collect them in a struct:

```go
type HandlerParams struct {
    dig.In

    Logger *zap.Logger
    DB     *sql.DB
    Cache  *redis.Client `optional:"true"`
    RO     *sql.DB       `name:"readonly"`
    Routes []http.Handler `group:"routes"`
}
```

Tags on fields: `name:"..."`, `optional:"true"`, `group:"..."`.

## Returning several values: `dig.Out`

```go
type ConnResult struct {
    dig.Out

    Primary *sql.DB `name:"primary"`
    Readonly *sql.DB `name:"readonly"`
}

func NewConnections(cfg *Config) (ConnResult, error) { ... }
```

## Same type, multiple instances: names

Two constructors of the same type collide. Disambiguate at registration:

```go
c.Provide(NewPrimaryDB, dig.Name("primary"))
c.Provide(NewReadOnlyDB, dig.Name("readonly"))
```

Consumers request a specific one via the `name:"..."` tag on a `dig.In` field.

## Many providers, one consumer: groups

Groups wire several constructors into one slice — routes, health checks, migrations:

```go
c.Provide(NewUserHandler, dig.Group("routes"))
c.Provide(NewPostHandler, dig.Group("routes"))

err := c.Invoke(func(routes []http.Handler) { /* mount */ })
```

`group:"routes"` yields `[][]http.Handler` when providers each return a slice; append `,flatten` (`group:"routes,flatten"`) to unwrap it. Groups are unordered — if sequence matters, provide one ordered slice from a single constructor.

## Exposing interfaces: `dig.As`

Register a concrete factory and publish it under interfaces without an adapter struct:

```go
c.Provide(NewPostgresDB, dig.As(new(Database), new(io.Closer)))
```

Consumers ask for `Database` or `io.Closer`; `*PostgresDB` itself stays private.

## Decorate, scopes, validation

- `Decorate` replaces an existing type with a wrapped version — last writer wins per provider graph, used for test fakes or adding behavior.
- `Scope` forks the container for per-request/service graphs.
- `DryRun(true)` validates the graph without running constructors — ideal for CI.
- `DeferAcyclicVerification()` trades a startup check for faster registration; `RecoverFromPanics()` converts constructor panics into typed errors.

See [advanced reference](references/advanced.md) for the mechanics, plus the full series of recipes in [recipes reference](references/recipes.md).

## Best practices

1. Own the container in `main` (composition root); never pass it into domain code — that is service-locator territory and kills testability.
2. Accept interfaces, return structs; let `dig.As` narrow wide structs.
3. Switch to `dig.In` structs at 4+ deps — adding one dependency becomes a field, not a signature break.
4. Keep `Provide` calls grouped per module file so refactoring is a per-module concern.
5. Validate eagerly: `Invoke` against the root in CI (or use `DryRun(true)`) to catch missing providers before deploy.
6. Return errors from constructors rather than panicking — dig enriches them with the path.

## Common mistakes

| Mistake | Failure | Fix |
| --- | --- | --- |
| Container passed to services | tests must build a real container | inject the typed deps only |
| Two providers of one type, unnamed | `Provide` errors | `Name` them, or merge results in a `dig.Out` |
| Ignored `Provide` errors | cryptic missing-type later | wrap registrations in a `must` helper |
| Order-dependent groups | groups are unordered | explicit ordered slice from one constructor |
| Side effects in imports | init races | do all work inside constructors |

## Testing

Containers are cheap, so build one per test; swap dependencies with `Decorate`; drive the system through `Invoke`. For per-test wiring helpers, graph validation in CI, and panic-recovery checks, see [testing reference](references/testing.md).

## Cross-references

- `golang-uber-fx` — lifecycle, modules, and the signal-aware run loop built on dig.
- `golang-dependency-injection` — DI concepts and library comparisons.
- `golang-samber-do` — a generics-based container without reflection.
- `golang-google-wire` — compile-time DI, no runtime container.
- `golang-structs-interfaces` — interface design for constructor signatures.
- `golang-testing` — general testing patterns.

Library bugs: [uber-go/dig issues](https://github.com/uber-go/dig/issues).
