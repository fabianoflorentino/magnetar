# `mo.Either[L, R]` Reference

A value that is exactly one of two types. The convention is Left for the alternative and Right for the primary, but — unlike `Result` — neither side means failure.

## Constructors and inspection

```go
left := mo.Left[string, int]("cached")   // Left(cached)
right := mo.Right[string, int](42)       // Right(42)

left.IsLeft()    // true
left.IsRight()   // false
```

## Extraction

| Method | Returns | Behavior |
| --- | --- | --- |
| `Left()` / `Right()` | `(T, bool)` | Value and side indicator |
| `MustLeft()` / `MustRight()` | `T` | Value or panic |
| `LeftOrElse(v)` / `RightOrElse(v)` | `T` | Value or fallback |
| `LeftOrEmpty()` / `RightOrEmpty()` | `T` | Value or zero value |
| `Unpack()` | `(L, R)` | Both; one side is the zero value |

## Transformations

`Swap` exchanges sides:

```go
e := mo.Left[string, int]("hello")
swapped := e.Swap() // Right("hello")
```

`MapLeft` / `MapRight` transform one side, returning a new Either:

```go
e := mo.Left[string, int]("hello")
upper := e.MapLeft(func(s string) mo.Either[string, int] {
    return mo.Left[string, int](strings.ToUpper(s))
})
```

The direct methods cannot change the type parameters (Go methods cannot introduce new ones). For transforms that swap types, use the `either` sub-package functions — see [pipelines.md](./pipelines.md).

`Match` branches on both cases and returns a consistent Either:

```go
e.Match(
    func(left string) mo.Either[string, int] { ... },
    func(right int) mo.Either[string, int] { ... },
)
```

`ForEach` runs the matching side effect only:

```go
e.ForEach(
    func(left string) { fmt.Println("Left:", left) },
    func(right int) { fmt.Println("Right:", right) },
)
```

## Either vs Result

| Aspect | `Either[L, R]` | `Result[T]` |
| --- | --- | --- |
| Alternative type | Any `L` | Always `error` |
| Semantics | Two valid alternatives | Success or failure |
| Fit | Cached vs fresh, strategy A vs B | An operation that may fail |
| JSON | Not supported | JSON-RPC shape |

`Result[T]` is effectively `Either[error, T]`; convert with `result.ToEither()`.

## Three+ type unions

For more than two variants the naming shifts from Left/Right to positional arguments:

```go
e := mo.NewEither3Arg1[string, int, bool]("hello")  // holds T1
e = mo.NewEither3Arg2[string, int, bool](42)        // holds T2
e = mo.NewEither3Arg3[string, int, bool](true)      // holds T3

e.IsArg1()
val, ok := e.Arg1()
v := e.MustArg1()
v := e.Arg1OrElse("fallback")
v := e.Arg1OrEmpty()
a, b, c := e.Unpack()
e.MapArg1(func(s string) mo.Either3[string, int, bool] { ... })
```

`Either4[T1, T2, T3, T4]` and `Either5[T1, T2, T3, T4, T5]` repeat the pattern with four and five `Arg` constructors, `IsArgN` predicates, and `MapArgN` transforms. Reach for them when an API returns genuinely different payload shapes per request; beyond five variants, a dedicated type is usually clearer.