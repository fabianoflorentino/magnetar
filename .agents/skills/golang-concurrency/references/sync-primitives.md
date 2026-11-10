# Sync Primitives

`sync` provides the low-level building blocks for concurrent Go programs. Each primitive solves a specific coordination problem — choosing the right one depends on the access pattern, not on habit.

## sync.Mutex

Exclusive access to shared state. Hold the lock for the shortest time possible — never hold a mutex across I/O, network calls, or channel operations.

```go
type SafeCache struct {
    mu    sync.Mutex
    items map[string]string
}

func (c *SafeCache) Get(key string) (string, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    v, ok := c.items[key]
    return v, ok
}

func (c *SafeCache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = value
}
```

Embed the mutex as an unexported field, placed directly above the fields it protects.

```go
type Registry struct {
    mu      sync.Mutex // protects entries
    entries map[string]Entry
}
```

## sync.RWMutex

Use when reads greatly outnumber writes. Multiple goroutines can hold `RLock` simultaneously; `Lock` is exclusive.

```go
type Config struct {
    mu     sync.RWMutex
    values map[string]string
}

func (c *Config) Get(key string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.values[key]
}

func (c *Config) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.values[key] = value
}
```

Do not upgrade `RLock` to `Lock` — this deadlocks. Release `RLock` first, then acquire `Lock`.

## sync/atomic

Lock-free operations for simple values. Prefer over `Mutex` for simple counters and flags — faster under low contention.

```go
var requestCount atomic.Int64

func handleRequest() {
    requestCount.Add(1)
}

func getCount() int64 {
    return requestCount.Load()
}
```

```go
var shuttingDown atomic.Bool

func shutdown() {
    shuttingDown.Store(true)
}

func isRunning() bool {
    return !shuttingDown.Load()
}
```

Go 1.19+ provides typed atomics (`atomic.Int64`, `atomic.Bool`, `atomic.Pointer[T]`) — prefer these over raw `atomic.AddInt64`/`atomic.LoadInt64`.

## sync.Map

Use only for write-once/read-many patterns or when multiple goroutines read/write disjoint key sets. For other patterns, a plain `map` + `sync.RWMutex` is faster.

```go
var cache sync.Map

func Get(key string) (any, bool) {
    return cache.Load(key)
}

func Set(key string, value any) {
    cache.Store(key, value)
}

func GetOrSet(key string, compute func() any) any {
    if v, ok := cache.Load(key); ok {
        return v
    }
    v, _ := cache.LoadOrStore(key, compute())
    return v
}
```

Do not use `sync.Map` when you need to iterate, get the length, or when writes are frequent and keys overlap heavily.

## sync.Pool

Reuse temporary objects to reduce GC pressure. Objects in the pool may be reclaimed at any GC cycle — do not store persistent state.

```go
var bufPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func process(data []byte) string {
    buf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufPool.Put(buf)
    }()

    buf.Write(data)
    return buf.String()
}
```

Always `Reset()` before `Put()` — returning dirty objects causes bugs. Do not assume an object from `Get()` is zeroed — the `New` func only runs if the pool is empty. Best for short-lived, frequently allocated objects (buffers, encoders, temporary structs).

## sync.Once

One-time initialization. Executes exactly once, regardless of how many goroutines call it concurrently.

```go
type DBClient struct {
    initOnce  sync.Once
    closeOnce sync.Once
    conn      *sql.DB
}

func (c *DBClient) getConn() *sql.DB {
    c.initOnce.Do(func() {
        var err error
        c.conn, err = sql.Open("postgres", dsn)
        if err != nil {
            panic(fmt.Sprintf("db init: %v", err))
        }
    })
    return c.conn
}

func (c *DBClient) Close() error {
    var err error
    c.closeOnce.Do(func() {
        err = c.conn.Close()
    })
    return err
}
```

Go 1.21+ provides `sync.OnceFunc`, `sync.OnceValue`, and `sync.OnceValues` for simpler use cases.

```go
var loadConfig = sync.OnceValue(func() *Config {
    cfg, err := parseConfig("config.yaml")
    if err != nil {
        panic(err)
    }
    return cfg
})
```

## sync.WaitGroup

Wait for a set of goroutines to finish.

### Go 1.25+: `wg.Go`

`WaitGroup.Go` starts a goroutine, adds it to the group, and removes it when the function returns.

```go
func processAll(items []Item) {
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Go(func() {
            process(item)
        })
    }

    wg.Wait()
}
```

Rules:

- `WaitGroup.Go` is Go 1.25+, not Go 1.24.
- The function passed to `wg.Go` must not panic.
- `WaitGroup` does not propagate errors and does not cancel siblings.
- For first-error-wins, cancellation, concurrency limits, or returned values, use `golang.org/x/sync/errgroup`.

Benefits: no manual `Add`/`Done` bookkeeping, lower risk of ordering bugs, cleaner API for simple fire-and-wait work.

### Go <1.25 fallback

```go
func processAll(ctx context.Context, items []Item) {
    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            process(ctx, item)
        }(item)
    }
    wg.Wait()
}
```

```go
// ✗ Bad — Add inside the goroutine (race: Wait may return before Add runs)
go func() {
    wg.Add(1)
    defer wg.Done()
    process(item)
}()
```

## golang.org/x/sync/singleflight

Deduplicates concurrent calls for the same key. When multiple goroutines request the same resource simultaneously, only one executes; the rest wait and share the result.

```go
var group singleflight.Group

func GetUser(ctx context.Context, id string) (*User, error) {
    v, err, _ := group.Do(id, func() (any, error) {
        return db.QueryUser(ctx, id)
    })
    if err != nil {
        return nil, err
    }
    return v.(*User), nil
}
```

Use for cache stampede prevention, deduplicating expensive lookups (DB, API), rate-limited external service calls.

## golang.org/x/sync/errgroup

Goroutine group with error propagation. Returns the first error from any goroutine. With `WithContext`, cancels remaining goroutines on first error.

```go
func fetchAll(ctx context.Context, urls []string) ([]Response, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Response, len(urls))

    for i, url := range urls {
        g.Go(func() error {
            resp, err := fetch(ctx, url)
            if err != nil {
                return fmt.Errorf("fetching %s: %w", url, err)
            }
            results[i] = resp
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

### Bounded concurrency with SetLimit

Use `SetLimit` to bound concurrency and avoid unbounded goroutine spawning.

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10)

for _, task := range tasks {
    g.Go(func() error {
        return process(ctx, task)
    })
}
return g.Wait()
```

This replaces hand-rolled worker pools for most use cases.

→ See `fabianoflorentino/golang-agent-skills@golang-concurrency` skill for high-level patterns and decision trees.
