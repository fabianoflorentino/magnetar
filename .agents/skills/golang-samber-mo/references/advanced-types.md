# Advanced Types

`Option`, `Result`, and `Either` cover most monadic needs. The remaining types defer execution, represent asynchrony, or thread state — reach for them when a pipeline calls for explicit control over when effects run.

## Type map

```
Synchronous                     Asynchronous
IO[T]           (no error)  ->  Task[T]          (no error)  ->  Future[T]
IOEither[T]     (with error) -> TaskEither[T]    (with error) ->  Future[T]
```

- **IO** wraps a synchronous side-effecting computation.
- **Task** wraps an asynchronous computation that yields a Future.
- **Future** represents a value that will arrive later.
- The **Either** variants add error handling to each pair.

## `Future[T]` — asynchronous values

A promise-like value: construct it with resolver callbacks, chain onto it, and collect the result when needed.

```go
future := mo.NewFuture(func(resolve func(int), reject func(error)) {
    result, err := expensiveComputation()
    if err != nil {
        reject(err)
    } else {
        resolve(result)
    }
})
```

Chaining transforms each stage and propagates failure:

```go
future.
    Then(func(v int) (int, error) {
        return v * 2, nil
    }).
    Catch(func(err error) (int, error) {
        return 0, err
    }).
    Finally(func(v int, err error) (int, error) {
        log.Println("done")
        return v, err
    })
```

Collect blocks until resolution: `future.Collect()` returns `(value, error)`, `future.Result()` a `Result[T]`, and `future.Either()` an `Either[error, T]`. `future.Cancel()` terminates the chain.

## `IO[T]` — deferred synchronous effects

Wraps a side-effecting function that runs only when `Run()` is called. IO never fails.

```go
io := mo.NewIO(func() string { return "hello" })
result := io.Run()

// Parameterized variants IO1..IO5
io1 := mo.NewIO1(func(name string) string { return "hello " + name })
io1.Run("alice")

io2 := mo.NewIO2(func(a, b int) int { return a + b })
io2.Run(1, 2)
```

## `IOEither[T]` — fallible deferred effects

Like IO, but the callback must return `Either[error, T]` — build the branches with `mo.Left`/`mo.Right`, not `(T, error)`:

```go
readConfig := mo.NewIOEither(func() mo.Either[error, string] {
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        return mo.Left[error, string](err)
    }
    return mo.Right[error, string](string(data))
})
result := readConfig.Run() // Either[error, string]
```

`IOEither1` … `IOEither5` follow with parameter counts.

## `Task[T]` — deferred async computation

Wraps a function that returns a `*Future[T]`; `Run()` executes it lazily. Use `mo.NewTaskFromIO(io)` to lift a synchronous IO into an async Task.

```go
task := mo.NewTask(func() *mo.Future[int] {
    return mo.NewFuture(func(resolve func(int), reject func(error)) {
        time.Sleep(time.Second)
        resolve(42)
    })
})

future := task.Run()
value, err := future.Collect()
```

`Task1` … `Task5` add parameters; `NewTaskEither` is the fallible variant and additionally offers `Match`, `OrElse`, `ToEither`, and `ToTask` (`ToTask(fallback)` replaces the error branch with a fallback value).

## `State[S, A]` — threaded state

A computation that carries state `S` while producing values of type `A`:

```go
counter := mo.NewState(func(count int) (string, int) {
    return fmt.Sprintf("count=%d", count), count + 1
})
result, newState := counter.Run(0)
```

Composition helpers:

- `mo.ReturnState[S, A](value)` — yields a value without touching state.
- `.Put(newState)` — replace the state.
- `.Modify(fn)` — transform the state.
- Getter example — a state computation that returns the current state unchanged.

```go
parseChar := mo.NewState(func(s ParseState) (byte, ParseState) {
    ch := s.Input[s.Position]
    return ch, ParseState{Input: s.Input, Position: s.Position + 1}
})
```

## When to use which

| Type | Use when |
| --- | --- |
| Future | Asynchronous work needs chaining (`Then`/`Catch`/`Finally`) |
| IO | Synchronous effects must be deferred and composed |
| IOEither | Deferred effects that can fail |
| Task | Asynchronous work should run lazily |
| TaskEither | Lazy async work that can fail |
| State | State threads through a series of computations |

Most Go projects stop at Option, Result, and Either. The deferred and stateful types earn their place only when a pipeline explicitly controls when side effects execute.