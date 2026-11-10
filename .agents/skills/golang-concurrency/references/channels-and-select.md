# Channels and Select

Channels synchronize goroutines and pass values between them. A channel without a sender blocks the receiver; a channel without a receiver blocks the sender. The `select` statement multiplexes across multiple channel operations so a goroutine never blocks on one channel when another is ready.

## Goroutine lifecycle

Every goroutine must answer one question before it starts: how does it stop? A goroutine that cannot be cancelled leaks when the caller moves on.

```go
// ✗ Bad — fire-and-forget, no way to stop or wait
func startWorker() {
    go func() {
        for {
            doWork()
        }
    }()
}

// ✓ Good — respects context cancellation, caller can wait
func startWorker(ctx context.Context) *sync.WaitGroup {
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            select {
            case <-ctx.Done():
                return
            default:
                doWork(ctx)
            }
        }
    }()
    return &wg
}
```

Recover panics at goroutine boundaries — a panic in any goroutine crashes the entire process.

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("goroutine panic: %v", r)
        }
    }()
    doWork(ctx)
}()
```

## Channel direction

Specify direction in function signatures so the compiler rejects misuse.

```go
func produce(ch chan<- int) // send-only
func consume(ch <-chan int) // receive-only
```

## Channel closing

Only the sender closes a channel. A receiver that closes a channel panics if the sender writes after the close.

```go
func generate(ctx context.Context) <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch)
        for i := 0; ; i++ {
            select {
            case ch <- i:
            case <-ctx.Done():
                return
            }
        }
    }()
    return ch
}
```

## Buffer size

| Size | When to use |
| --- | --- |
| 0 (unbuffered) | Default. Synchronizes sender and receiver — use when handoff guarantees matter |
| 1 | Signal channels (`done := make(chan struct{}, 1)`), or when the sender must not block on a single pending item |
| N > 1 | Only with measured justification — document why N and what happens when the buffer fills |

```go
ch := make(chan Result)             // unbuffered for synchronous handoff
done := make(chan struct{}, 1)      // buffered 1 for signal
ch := make(chan Task, 1000)         // ✗ why 1000? what if it fills?
```

## Select with ctx.Done()

Every `select` in a long-lived goroutine must include `ctx.Done()` — without it, the goroutine leaks when the context is cancelled.

```go
func process(ctx context.Context, in <-chan Task, out chan<- Result) {
    for {
        select {
        case <-ctx.Done():
            return
        case task, ok := <-in:
            if !ok {
                return
            }
            result := handle(ctx, task)
            select {
            case out <- result:
            case <-ctx.Done():
                return
            }
        }
    }
}
```

## Avoid repeated `time.After` in hot loops

`time.After` allocates a new timer on every call. In a loop, that is GC churn.

```go
// ✗ Bad — new timer every iteration
for {
    select {
    case msg := <-ch:
        handle(msg)
    case <-time.After(5 * time.Second):
        handleTimeout()
    }
}

// ✓ Good (Go 1.23+) — reuse the timer
timer := time.NewTimer(5 * time.Second)
defer timer.Stop()
for {
    select {
    case msg := <-ch:
        timer.Stop()
        timer.Reset(5 * time.Second)
        handle(msg)
    case <-timer.C:
        handleTimeout()
        timer.Reset(5 * time.Second)
    }
}
```

Before Go 1.23, drain a possible stale value from `timer.C` before `Reset` when `Stop` returns false. Go 1.23+ guarantees that receiving from `timer.C` after `Stop` returns blocks rather than returning a stale value.
