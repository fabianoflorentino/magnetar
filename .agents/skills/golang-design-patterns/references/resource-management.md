# Resource Management and Lifecycle

## Close as soon as you open

Place `defer Close()` immediately after a successful open so a later edit adding an early return cannot skip cleanup. This holds for files, `*sql.Rows`, HTTP response bodies, and anything else implementing `io.Closer`:

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

```go
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()

rows, err := db.QueryContext(ctx, query)
if err != nil {
    return err
}
defer rows.Close()
```

A stated exception: durability-sensitive writes must report their close or flush error rather than discarding it.

## Prefer runtime.AddCleanup over runtime.SetFinalizer

Finalizers are unpredictable, run at the GC's discretion, and can resurrect objects. `runtime.AddCleanup` (Go 1.24+) removes the sharp edges: several cleanups per object, a copy of the value passed to the callback (no resurrection), and correct behavior for objects in cycles.

```go
type Resource struct {
    handle uintptr
}

func NewResource() *Resource {
    r := &Resource{handle: acquireHandle()}
    runtime.AddCleanup(r, releaseHandle, r.handle)
    return r
}
```

Treat a cleanup as a safety net, never as the primary release path — explicit `Close()` stays the contract.

## Bounded resource pools

Cap every pool. A channel with fixed capacity gives bounded allocation plus cancellation-aware checkout:

```go
type ConnPool struct {
    conns chan *Conn
}

func NewConnPool(maxSize int, factory func() (*Conn, error)) (*ConnPool, error) {
    pool := &ConnPool{conns: make(chan *Conn, maxSize)}
    for range maxSize {
        conn, err := factory()
        if err != nil {
            return nil, fmt.Errorf("creating connection: %w", err)
        }
        pool.conns <- conn
    }
    return pool, nil
}

func (p *ConnPool) Get(ctx context.Context) (*Conn, error) {
    select {
    case conn := <-p.conns:
        return conn, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (p *ConnPool) Put(conn *Conn) {
    select {
    case p.conns <- conn:
    default:
        conn.Close() // pool full — discard instead of growing
    }
}
```

For short-lived objects, `sync.Pool` is the lighter-weight alternative; see `golang-concurrency` for its rules.

## Graceful shutdown

Drive the lifecycle from a context derived from OS signals. On cancellation, stop accepting work, drain in-flight requests within a budget, then close remaining resources in a fixed order:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    srv := &http.Server{Addr: ":8080", Handler: router}
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
        }
    }()

    <-ctx.Done()
    slog.Info("shutting down")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        slog.Error("shutdown error", "error", err)
    }

    db.Close()
    slog.Info("shutdown complete")
}
```

The same shape serves gRPC servers, message consumers, and background workers: capture the signal, run the service, block on cancellation, drain with a timeout, close the rest. For goroutine-level shutdown see `golang-concurrency`.