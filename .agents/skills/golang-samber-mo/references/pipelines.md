# Pipeline Sub-packages

Sub-packages `option`, `result`, `either`, `either3`, `either4`, and `either5` exist for one reason: direct methods cannot change a monad's type parameter. Standalone generic functions can, and they also compose into readable `Pipe` chains.

## Why they exist

`Option[int].Map` is fixed to `func(T) (T, bool) -> Option[T]`. It can never return `Option[string]`. The sub-packages lift the transform out of the method:

```go
import "github.com/samber/mo/option"

strOpt := option.Map(func(v int) string {
    return strconv.Itoa(v)
})(mo.Some(42))
// Some("42")
```

## `option` package

Type-changing transforms:

| Function | Purpose |
| --- | --- |
| `option.Map` | `func(I) O -> func(Option[I]) Option[O]` |
| `option.FlatMap` | `func(I) Option[O] -> func(Option[I]) Option[O]` |
| `option.Match` | Branch on value/None, changing type |
| `option.FlatMatch` | Branch returning Options |

`Pipe` chains several steps left to right:

```go
result := option.Pipe3(
    mo.Some(42),
    option.Map(func(v int) string { return strconv.Itoa(v) }),
    option.Map(func(s string) []byte { return []byte(s) }),
    option.FlatMap(func(b []byte) mo.Option[string] {
        if len(b) > 0 {
            return mo.Some(string(b))
        }
        return mo.None[string]()
    }),
)
```

`option.Pipe1` through `option.Pipe10` bound the chain length.

## `result` package

Same shape, for the fallible side:

| Function | Purpose |
| --- | --- |
| `result.Map` | Transform the success value, changing type |
| `result.FlatMap` | Chain a Result-producing function |
| `result.Match` | Branch on value/error, changing type |
| `result.FlatMatch` | Branch returning Results |

```go
parsed := result.Pipe2(
    mo.TupleToResult(os.ReadFile("config.yaml")),
    result.Map(func(data []byte) Config {
        var cfg Config
        yaml.Unmarshal(data, &cfg)
        return cfg
    }),
    result.FlatMap(func(cfg Config) mo.Result[ValidConfig] {
        return validateConfig(cfg)
    }),
)
```

`result.Pipe1` through `result.Pipe10`.

## `either` package

Operates on both sides:

| Function | Purpose |
| --- | --- |
| `either.MapLeft` | Transform the left type |
| `either.MapRight` | Transform the right type |
| `either.FlatMapLeft` / `either.FlatMapRight` | Chain Either-producing functions |
| `either.Match` | Branch on either side |
| `either.Swap` | Exchange sides |

```go
result := either.Pipe2(
    mo.Right[error, int](42),
    either.MapRight(func(v int) string { return strconv.Itoa(v) }),
    either.MapRight(func(s string) []byte { return []byte(s) }),
)
```

`either.Pipe1` through `either.Pipe10`.

## `either3`, `either4`, `either5` packages

Each mirrors its type: `Match` with per-argument handlers, `MapArg1` … `MapArgN`, and `Pipe1` through `Pipe10`.

## Pipe vs direct methods

| Scenario | Reach for |
| --- | --- |
| Same type in, same type out | Direct method (`.Map`) |
| Type changes across a step | Sub-package function |
| 3+ chained type-changing steps | `Pipe3` and up |
| Single type-changing step | Standalone sub-package call |
| Mixed same-type and cross-type | Direct for same-type, pipe for the rest |

Mixed usage stays coherent — start with a method, hand off to a pipe when the type changes:

```go
opt := mo.Some(42).
    Map(func(v int) (int, bool) { return v * 2, true })

result := option.Pipe2(
    opt,
    option.Map(func(v int) string { return strconv.Itoa(v) }), // type change
    option.Map(func(s string) User { return User{Name: s} }),
)
```