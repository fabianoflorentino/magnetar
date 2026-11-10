---
name: golang-samber-mo
description: "Monadic types for Golang using samber/mo — Option, Result, Either, Future, IO, Task, and State types for type-safe nullable values, error handling, and functional composition with pipeline sub-packages. Apply when using or adopting samber/mo, when the codebase imports `github.com/samber/mo`, or when considering functional programming patterns as a safety design for Golang. Not for nil-safety and zero-value design without this library (→ See `fabianoflorentino/golang-agent-skills@golang-safety` skill), nor for native error wrapping with fmt.Errorf, errors.Is and errors.As (→ See `fabianoflorentino/golang-agent-skills@golang-error-handling` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🎭"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who encodes "what can happen" in the type system. You prefer a `Result[T]` over an undocumented panic, and an `Option[T]` over a comment saying a field is "sometimes nil".

**Modes:**

- **Build** — adopting monadic types at API boundaries and in domain logic.
- **Review** — checking for `MustGet` misuse, type-drifting maps, and Either used where Result belongs.
- **Refactor** — converting `(T, error)` flow into composable pipelines without moving business logic.

**When to use:** any task centered on samber/mo. For nil-safety without the library, or idiomatic Go error wrapping, prefer `golang-safety` and `golang-error-handling` respectively; this skill is about committed monadic design, not decoration.

## The core types

| Type | Encodes | Plain-Go equivalent |
| --- | --- | --- |
| `Option[T]` | a value that may be absent | `*T` plus nil checks |
| `Result[T]` | success or failure | `(T, error)` |
| `Either[L, R]` | one of two valid alternatives | a tagged struct |
| `Future[T]` | a not-yet-available value | a channel + goroutine |
| `IO[T]` / `Task[T]` | deferred sync/async effects | closures over side effects |
| `State[S, A]` | stateful computation | explicit state threading |

Start with `Option` and `Result`; the deferred types pay off only in specific pipelines and are covered in [advanced types](references/advanced-types.md).

## Option — nullable without the nil dance

```go
nickname := mo.Some("alex").OrElse("anon")        // "alex"
maybe := mo.PointerToOption(user.NicknamePtr)     // nil pointer → None
```

`Option` implements `json.Marshaler`, `sql.Scanner`, and `driver.Valuer`, so it drops straight into response structs and model scans:

```go
type User struct {
    ID    int
    Phone mo.Option[string] // null in DB → None, absent JSON
}
```

Use it for genuinely-absent values, not for zero defaults: when an empty string is meaningful, a plain `string` is the honest type. Full surface in [option reference](references/option.md).

## Result — errors as composable values

Wrap Go's `(T, error)` at the boundary, then transform inside the type:

```go
cfg := mo.TupleToResult(os.ReadFile("config.yaml"))
name := cfg.Map(bytesToString).OrElse("default")
```

The one caveat that bites: a method cannot change the type parameter, so `Result[T].Map` stays `Result[T]`. For transforms that change the payload type, the sub-package functions step in:

```go
import "github.com/samber/mo/result"

parsed := result.Pipe3(
    mo.TupleToResult(os.ReadFile("config.yaml")),
    result.Map(parseConfig), // []byte → Config
    result.FlatMap(func(c Config) mo.Result[ValidConfig] { return validate(c) }),
)
```

Rule of thumb: same-type steps chain with methods, type-changing steps chain with sub-package functions. See [result reference](references/result.md) and [pipelines reference](references/pipelines.md).

## Either — two valid branches

`Option` and `Result` both encode an absent/failed path; `Either[L, R]` encodes two legitimate possibilities — cached vs fresh, strategy A vs B — where neither side implies failure:

```go
func current(id string) mo.Either[Snapshot, Live] {
    if s, ok := snapshots.get(id); ok {
        return mo.Left[Snapshot, Live](s)
    }
    return mo.Right[Snapshot, Live](live.Query(id))
}
```

Match on it to consume. `Either3`/`Either4`/`Either5` widen the union; if you ever reach 5+ variants, a dedicated type is probably warranted. See [either reference](references/either.md).

## Do notation — imperative style, monadic safety

`mo.Do` runs a closure and turns any `MustGet` panic into a `Result` error:

```go
out := mo.Do(func() int {
    a := mo.Some(21).MustGet()
    b := mo.Ok(2).MustGet()
    return a * b // 42, wrapped in Ok
})
```

It translates straight-line Go into composable flow. Outside `Do`, keep `MustGet` for cases where presence is logically guaranteed — otherwise use `OrElse`.

## Common patterns

- API responses: `Option[T]` for optional JSON fields that must not serialize as `null` vs zero.
- DB models: `Option` for real nullable columns; scan directly.
- Error-free lookup: wrap map reads with `mo.TupleToOption(m[key])`.
- Uniform folding: `mo.Fold[E, T, U](value, onValue, onAbsent)` works across Option, Result, and Either.

## Best practices

1. Convert at the boundary — `TupleToResult`/`TupleToOption` at API edges, then chain inside the domain.
2. `Result` for success/failure, `Either` for two valid states, `Option` for absence.
3. Prefer `OrElse` over `MustGet` outside `Do`.
4. Pick the direct method or the sub-package function by whether the payload type changes.
5. Chain flat, read left-to-right; nested conditionals defeat the point.

See the [monads guide](references/monads-guide.md) for the conceptual grounding. Bug reports for the library go to [samber/mo issues](https://github.com/samber/mo/issues).

## Cross-references

- `golang-samber-lo` — collection transforms that compose with `mo` values.
- `golang-error-handling` — idiomatic Go error wrapping when monads are a poor fit.
- `golang-safety` — nil-safety and defensive design without the library.
- `golang-database` — nullable scan patterns that `Option` plugs into.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.
