# Concurrency Debugging

## Goroutine leaks

**Symptoms:** memory creeping up, goroutine count rising, no CPU spike.

**Diagnosis:** dump goroutine stacks (`curl localhost:6060/debug/pprof/goroutine?debug=2`, see [pprof.md](./pprof.md)) and hunt for goroutines parked in `chan receive`. The dedicated leak profile has shipped generally since Go 1.27 (no build flag) and is served like any other standard profile:

```bash
curl http://localhost:6060/debug/pprof/goroutineleak?debug=2
go tool pprof http://localhost:6060/debug/pprof/goroutineleak
```

Keep the existing toolset: `go.uber.org/goleak` in tests, `runtime.NumGoroutine()` for coarse monitoring, and `go test -race ./...`.

**Common causes:**

```go
// 1. Blocked on a channel the sender never closes
for {
    select {
    case job, ok := <-jobs:
        if !ok {
            return
        }
        process(job)
    case <-ctx.Done():
        return
    }
}

// 2. Unclosed HTTP response bodies — every call leaks a connection

// 3. time.After in a loop allocates a fresh timer per iteration
ticker := time.NewTicker(time.Second)
defer ticker.Stop()
for {
    select {
    case <-ticker.C:
        do()
    case <-ctx.Done():
        return
    }
}
```

## Race conditions

**Symptoms:** intermittent failures that pass locally, differ between machines, or flip with reruns.

**Diagnosis:** race conditions are timing bugs — run them under the detector, never by eyeballing:

```bash
go test -race ./...
go run -race main.go
```

The detector costs roughly 10x runtime but finds data races reliably. The usual suspects:

- shared map without a mutex
- shared variable without atomic access
- publishing a reference before initialization completes
- goroutines closing over outer variables without synchronization

## Deadlocks

**Symptoms:** the program hangs, goroutines stuck in `chan receive` or a lock wait.

**Diagnosis:** stack dump of every goroutine:

```bash
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

Or from code: `runtime.Stack(buf, true)`.

**Common patterns:**

1. **Circular wait** — A waits on B while B waits on A.
2. **Missing send** — the goroutine the receiver is waiting on exited early.
3. **Inconsistent lock order** — acquire locks in the same order everywhere, or you get a wedge under the right interleaving.