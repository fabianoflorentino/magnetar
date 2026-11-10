# Monads in Go

The conceptual grounding behind `samber/mo`: what functional style buys a Go codebase with generics, why a monad is a container with a policy, and where monadic structure stops paying.

## Functional basics

Functional programming treats computation as function evaluation. Four disciplines matter here:

- **Immutability** — data is transformed, never mutated in place.
- **Pure functions** — same input, same output, no hidden effects.
- **Composition** — complex behavior assembled from small functions.
- **Types as documentation** — signatures state what can happen.

Go is not a pure functional language, but Go 1.18+ generics make the patterns practical. `samber/mo` packages the proven abstractions — but they are decoration unless the codebase commits to the style.

## What a monad is

A monad wraps a value in a context, chains operations that transform the wrapped value without forcing an unwrap, and lets the context react automatically — an absent `Option` skips the transform, an `Err` short-circuits the rest of a chain.

Think railway tracks. Two run side by side:

```
Input -> [Transform A] -> [Transform B] -> [Transform C] -> Output
            | (error)       (skipped)        (skipped)
            v
         Error track --------------------------------> Error
```

Once something jumps to the bottom track, subsequent stages are skipped. `Result.Map` and `Result.FlatMap` automate exactly this switch — no `if err != nil` at every step.

## Why bother in Go

**Compile-time nil safety.** Go cannot distinguish "this pointer is always valid" from "this pointer might be nil". An `Option[T]` return forces the caller to handle `None`:

```go
func FindUser(id string) *User { ... }            // might be nil, caller must remember
func FindUser(id string) mo.Option[User] { ... }  // the type says it might be absent
```

**Error handling without the ceremony.** Where plain Go repeats the check, `Result` plus `mo.Do` short-circuits:

```go
result := mo.Do(func() Config {
    data := mo.TupleToResult(readFile(path)).MustGet()
    config := mo.TupleToResult(parseConfig(data)).MustGet()
    validated := mo.TupleToResult(validate(config)).MustGet()
    return validated
})
```

Any failed `MustGet` panics, `mo.Do` catches it, and the whole block collapses to `Err`.

**Composable pipelines.** Independent, testable stages chain through a fixed data flow; `None`/`Err` propagation is handled by the monad, not by procedural checks.

```go
import "github.com/samber/mo/option"

result := option.Pipe3(
    getUserOption(id),
    option.Map(func(u User) string { return u.Email }),
    option.FlatMap(func(email string) mo.Option[string] {
        if isValid(email) {
            return mo.Some(email)
        }
        return mo.None[string]()
    }),
    option.Map(func(email string) EmailAddress { return NewEmailAddress(email) }),
)
```

## The three core types

**`Option[T]`** encodes absence. Present → `mo.Some(v)`, absent → `mo.None[T]()`, safe access → `OrElse(default)` instead of `if u != nil`. Use it for values that may legitimately be missing — nullable DB columns, optional config, cache lookups. Do not use it where a zero value is meaningful: if empty string is a real state, a plain `string` is the honest type.

**`Result[T]`** encodes fallibility. Success → `mo.Ok(v)`, failure → `mo.Err[T](err)`, chaining → `.Map`/`.FlatMap`, defaults → `.OrElse(v)`. Use it when several fallible operations must chain with automatic error propagation. Skip it when each step inspects or rewrites the error — imperative error handling is clearer there.

**`Either[L, R]`** encodes two valid alternatives — cached vs fresh, sync vs async, strategy A vs B. Use it only when neither side is an error; the moment one side is always a failure, `Result` is the right type.

## When to use `mo` vs plain Go

Use `mo` when: building multi-step transformation pipelines, needing type-safe nullability in JSON/DB models, chaining error handling repetitively, or encoding impossible states out of the type system.

Stick with plain Go when: a one-step operation makes `if err != nil` perfectly clear, the code is a performance-critical hot path (monads add a thin per-use allocation), the team is not fluent in the style, or error recovery differs at every step.