# `mo.Result[T]` Reference

The success/failure type: construction, do-notation, inspection, transforms, conversion, and its JSON shape.

## Constructors

| Function | Behavior |
| --- | --- |
| `mo.Ok[T](v)` | Successful result |
| `mo.Err[T](err)` | Failed result |
| `mo.Errf[T](format, a...)` | Failed result with a formatted message |
| `mo.TupleToResult[T](v, err)` | `(T, error)` → Ok/Err |
| `mo.Try[T](f)` | Run `f func() (T, error)` and wrap the outcome |

## Do notation

`mo.Do` runs a closure and converts any `MustGet` panic into `Err`, giving imperative-looking straight-line code monadic error propagation:

```go
result := mo.Do(func() int {
    a := mo.Ok(10).MustGet() // panic on Err, caught by Do
    b := mo.Ok(32).MustGet()
    return a + b
}) // Ok(42)
```

`MustGet` outside `Do` panics without a catcher — use `OrElse` there.

## Inspection and extraction

| Method | Returns | Behavior |
| --- | --- | --- |
| `IsOk()` | `bool` | Success check |
| `IsError()` | `bool` | Failure check |
| `Error()` | `error` | Error, nil when Ok |
| `Get()` | `(T, error)` | Go-style destructuring |
| `MustGet()` | `T` | Value or panic; inside `mo.Do` only |
| `OrElse(fallback)` | `T` | Value or fallback on Err |
| `OrEmpty()` | `T` | Value or zero value on Err |

## Transforms

`Map` transforms the success value and can convert Ok → Err by returning an error:

```go
res := mo.Ok(42).Map(func(v int) (int, error) {
    return v * 2, nil
}) // Ok(84)

res := mo.Err[int](errors.New("fail")).Map(func(v int) (int, error) {
    return v * 2, nil // skipped
}) // Err("fail")
```

`MapValue` transforms without the error slot:

```go
mo.Ok(42).MapValue(func(v int) int { return v * 2 }) // Ok(84)
```

`MapErr` rewrites the error state:

```go
mo.Err[int](errors.New("fail")).MapErr(func(err error) (int, error) {
    return 0, fmt.Errorf("wrapped: %w", err)
}) // Err("wrapped: fail")
```

`FlatMap` chains Results of the same type:

```go
res := parseAge("25").FlatMap(func(age int) mo.Result[int] {
    return validateAge(age)
}) // Ok(25)
```

**Type-change caveat:** methods cannot alter the type parameter, so `Result[[]byte] → Result[Config]` needs the sub-package (`result.Map`, `result.FlatMap`) or `mo.Do` — see [pipelines.md](./pipelines.md).

`Match` branches on both cases; `ForEach` runs a side effect only on Ok:

```go
res.Match(
    func(v int) (int, error) { fmt.Println("ok", v); return v, nil },
    func(err error) (int, error) { return 0, err },
)
res.ForEach(func(v int) { fmt.Println("got", v) })
```

## Conversion

`ToEither()` maps Ok → Right and Err → Left, producing `Either[error, T]`:

```go
either := res.ToEither()
```

## JSON serialization

`Result` marshals to the JSON-RPC envelope:

```go
// Ok(42)   -> {"result": 42}
// Err(fail)-> {"error": {"message": "fail"}}
```