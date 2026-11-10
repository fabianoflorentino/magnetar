# Pipelines and Worker Pools

A pipeline is a series of stages connected by channels. Each stage is a goroutine that receives from upstream, processes, and sends downstream. Pipelines decompose complex processing into composable pieces; worker pools bound the concurrency of a fixed set of tasks.

## Pipeline pattern

Each stage receives values from an upstream channel, processes each value, and sends results to a downstream channel.

```go
func generate(ctx context.Context, nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            select {
            case out <- n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            select {
            case out <- n * n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    ch := generate(ctx, 2, 3, 4)
    results := square(ctx, ch)

    for v := range results {
        fmt.Println(v)
    }
}
```

Pipeline rules:

- Every stage must select on `ctx.Done()` — without it, a cancelled context leaves goroutines blocked forever.
- The producer closes its output channel; each downstream stage closes its own output.
- Never create unbounded goroutines in pipeline stages.
- Use unbuffered channels unless measured throughput justifies a buffer.

## Fan-out / Fan-in

Fan-out: multiple goroutines read from the same channel to parallelize work. Fan-in: multiple channels merge into a single output channel.

```go
func fanOut(ctx context.Context, in <-chan Task, workers int) <-chan Result {
    out := make(chan Result)
    var wg sync.WaitGroup

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                select {
                case task, ok := <-in:
                    if !ok {
                        return
                    }
                    select {
                    case out <- process(ctx, task):
                    case <-ctx.Done():
                        return
                    }
                case <-ctx.Done():
                    return
                }
            }
        }()
    }

    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}

func fanIn(ctx context.Context, channels ...<-chan Result) <-chan Result {
    out := make(chan Result)
    var wg sync.WaitGroup

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan Result) {
            defer wg.Done()
            for v := range c {
                select {
                case out <- v:
                case <-ctx.Done():
                    return
                }
            }
        }(ch)
    }

    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}
```

## Worker pool with errgroup

`errgroup.SetLimit` replaces hand-rolled worker pools for most use cases.

```go
func processAll(ctx context.Context, tasks []Task) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10)

    for _, task := range tasks {
        g.Go(func() error {
            return process(ctx, task)
        })
    }
    return g.Wait()
}
```

Use a hand-rolled worker pool only when you need per-worker state (connections, buffers), custom backpressure or priority scheduling, or graceful draining with in-flight task completion.

## Bounded concurrency with semaphore

When you need fine-grained concurrency control without errgroup:

```go
func processAll(ctx context.Context, items []Item) error {
    sem := make(chan struct{}, 10)
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Add(1)
        sem <- struct{}{}
        go func(item Item) {
            defer wg.Done()
            defer func() { <-sem }()
            process(ctx, item)
        }(item)
    }
    wg.Wait()
    return nil
}
```

Prefer `errgroup.SetLimit` when error propagation matters.

## Pipeline alternatives

### Go 1.23+ iterators

For in-process data transformations that do not need concurrency, iterators avoid goroutine and channel overhead.

```go
func Filter[T any](seq iter.Seq[T], pred func(T) bool) iter.Seq[T] {
    return func(yield func(T) bool) {
        for v := range seq {
            if pred(v) {
                if !yield(v) {
                    return
                }
            }
        }
    }
}

func Map[T, U any](seq iter.Seq[T], f func(T) U) iter.Seq[U] {
    return func(yield func(U) bool) {
        for v := range seq {
            if !yield(f(v)) {
                return
            }
        }
    }
}
```

Use iterators when processing is CPU-bound and does not benefit from parallelism, when lazy evaluation without goroutine overhead is desired, or when the data source is already sequential (slice, database cursor).

Use goroutine pipelines when stages involve I/O that benefits from concurrency, when true parallelism across CPU cores is needed, or when stages have different throughput characteristics.

### samber/ro

`samber/ro` provides a fluent, type-safe pipeline API for read-only collections.

```go
import "github.com/samber/ro"

emails, _ := ro.Collect(
    ro.Pipe(
        ro.FromSlice(users),
        ro.Filter(func(u User) bool { return u.Active }),
        ro.Map(func(u User) string { return u.Email }),
    ),
)
```

Use `samber/ro` for sequential data transformations that benefit from a fluent API.

## Goroutine leak detection

Use `go.uber.org/goleak` in `TestMain` to catch leaked goroutines across all tests.

```go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

## Common pipeline mistakes

| Mistake | Fix |
| --- | --- |
| Missing `ctx.Done()` in pipeline stage | Always select on context to allow cancellation |
| Not closing output channel | Producer must `defer close(out)` |
| Unbounded goroutine spawning | Use `errgroup.SetLimit` or a semaphore |
| Sending mutable data through channel | Send copies or immutable values |
| Blocking send without select | Wrap channel sends in select with `ctx.Done()` |

→ See `fabianoflorentino/golang-agent-skills@golang-concurrency` skill for sync primitives and channel patterns.
