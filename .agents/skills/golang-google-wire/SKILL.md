---
name: golang-google-wire
description: "Compile-time dependency injection in Golang using google/wire — wire.NewSet, wire.Build, wire.Bind (interface→concrete), wire.Struct, wire.Value, wire.InterfaceValue, wire.FieldsOf, cleanup functions, //go:build wireinject injector files, and generated wire_gen.go. Apply when using or adopting google/wire, when the codebase imports `github.com/google/wire`, or when wiring an application graph at compile time via `wire.Build`. For runtime DI with reflection, see `fabianoflorentino/golang-agent-skills@golang-uber-dig` skill."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🪡"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - wire
    install:
      - kind: go
        package: github.com/google/wire/cmd/wire@latest
        bins: [wire]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(wire:*) Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go architect using wire for compile-time DI. You let the compiler find missing dependencies, treat `wire_gen.go` as committed source, and re-run `wire ./...` after every graph change.

**Dependencies:**

- wire: `go install github.com/google/wire/cmd/wire@latest`

**When to use:** any task centered on google/wire. Runtime containers with lifecycle and modularity are `golang-uber-dig` / `golang-uber-fx`; the concept-level comparison lives in `golang-dependency-injection`. Note that `google/wire` was archived in August 2025 — feature-complete, with bug fixes still accepted.

## How it differs from runtime DI

| | wire | dig / fx / samber/do |
| --- | --- | --- |
| Resolution | compile-time codegen | runtime reflection/generics |
| Failure timing | `wire ./...` errors now | first `Invoke` / startup |
| Runtime container | none — plain Go calls | present |
| Lifecycle hooks | none built in | fx: OnStart/OnStop |
| Generated file | `wire_gen.go`, committed | none |

The payoff: the whole graph either builds or the tool tells you exactly which dependency is missing.

## Providers and sets

A provider is any function whose parameters are dependencies and whose results are provided types. It may return a value, `(value, error)`, or `(value, cleanup, error)`:

```go
func NewConfig() *Config { return &Config{Addr: ":8080"} }
func NewDB(cfg *Config) (*sql.DB, error) { return sql.Open("postgres", cfg.DSN) }
func NewRedis(cfg *Config) (*redis.Client, func(), error) {
    c := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
    return c, func() { c.Close() }, nil
}
```

`wire.NewSet` bundles providers for reuse and may reference other sets:

```go
var InfraSet = wire.NewSet(NewConfig, NewDB, NewRedis)
var ServiceSet = wire.NewSet(NewUserRepo, NewUserService,
    wire.Bind(new(UserStore), new(*UserRepo)),
)
```

Keep sets stable — library sets are a contract; adding inputs or removing outputs breaks downstream injectors. One set per package is a sane default.

## Injector files and `//go:build wireinject`

The injector file declares the function wire will generate the body for:

```go
//go:build wireinject

package main

import "github.com/google/wire"

func InitApp() (*App, func(), error) {
    wire.Build(InfraSet, ServiceSet, NewApp)
    return nil, nil, nil // replaced by codegen
}
```

The build tag keeps the stub out of the binary — only the untagged `wire_gen.go` compiles. Drop the tag and both files define the same function (a duplicate-symbol error). If the dummy returns annoy you, `panic(wire.Build(...))` works too.

Run `wire ./...` after every constructor-signature change; `wire check ./...` validates without regenerating (cheap for CI). Add `//go:generate go run github.com/google/wire/cmd/wire` so `go generate ./...` covers it, and commit `wire_gen.go`.

## Interface bindings

Wire refuses to infer interface satisfaction — you must declare it, so a new implementor elsewhere can't silently make the graph ambiguous:

```go
wire.Bind(new(UserStore), new(*PostgresUserRepo))
```

## Structs, values, fields

- `wire.Struct(new(Server), "Logger", "DB")` fills named fields from the graph (`"*"` fills all non-`wire:"-"` fields).
- `wire.Value(Foo{X: 42})` supplies a constant (no calls or channels).
- `wire.InterfaceValue(new(io.Reader), os.Stdin)` supplies an interface-typed literal.
- `wire.FieldsOf(new(Config), "DSN")` promotes a field as a graph node.

See [advanced](references/advanced.md) for the `wire:"-"` tag and field promotion details.

## Duplicate types

One provider per type, strictly. Disambiguate with named types rather than two functions returning `*sql.DB`:

```go
type PrimaryDSN string
type ReplicaDSN string
```

## Cleanup chains

Cleanup providers return `(T, func(), error)`; wire chains them in reverse order and, if construction fails midway, runs only the cleanups already built. In `main`, guard the cleanup against nil:

```go
app, cleanup, err := InitApp()
if err != nil {
    log.Fatal(err)
}
if cleanup != nil {
    defer cleanup()
}
```

## Full shape

```go
//go:build wireinject
package main

func InitApp() (*App, func(), error) {
    wire.Build(config.ConfigSet, infra.InfraSet, service.ServiceSet, NewApp)
    return nil, nil, nil
}

// main.go
func main() {
    app, cleanup, err := InitApp()
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()
    app.Run()
}
```

A complete decorated example with per-package sets and cleanup-heavy graphs: [recipes](references/recipes.md).

## Best practices

1. Never hand-edit `wire_gen.go`; it is a committed build artifact.
2. Build-tag every injector file, first line, or suffers duplicate symbols.
3. Named types over raw types for same-underlying duplicates.
4. Keep library sets additive and backward-compatible.
5. Return `(T, func(), error)` from cleanup providers and let wire order it.
6. One injector per file; delegate to per-package sets instead of fat `wire.Build` lists.

## Common mistakes

| Mistake | Fix |
| --- | --- |
| Editing `wire_gen.go` | change providers, re-run `wire ./...` |
| Missing `//go:build wireinject` | tag the first line of injector files |
| Two `*sql.DB` providers | named struct types, one provider each |
| Interface without `wire.Bind` | add the bind to the set |
| Forgetting re-run after changes | wire before `go build`; wire a Makefile target |
| Unguarded `cleanup()` | nil-safe: `if cleanup != nil { defer cleanup() }` |

## Testing

wire emits plain constructors, so tests inject by hand — nothing to clone or reset. Test injectors that swap real providers for fakes, plus a CI stale check for `wire_gen.go`: [testing](references/testing.md).

## Cross-references

- `golang-dependency-injection` — when DI pays off and the full library matrix.
- `golang-uber-dig` / `golang-uber-fx` — runtime reflection-based containers.
- `golang-samber-do` — generics-based container, no codegen.
- `golang-structs-interfaces` — interface design for constructor inputs/outputs.
- `golang-testing` — general testing patterns.

Library bugs: [google/wire issues](https://github.com/google/wire/issues).
