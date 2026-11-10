# I/O & Networking Optimization

Network and disk bottlenecks show up as goroutines parked on syscalls. The levers: connection reuse, explicit timeouts, and streaming instead of buffering.

## Diagnosing I/O bottlenecks

- `go tool pprof` goroutine/block profile — goroutines stuck in `dialConn` or `readLoop` signal connection-pool exhaustion.
- `fgprof` — off-CPU time dominating wall clock while CPU sits idle means waiting, not working.
- `go_goroutines` in production — a steady climb under stable load suggests leaked connections or goroutines.

## HTTP transport configuration

The zero-value `http.Client` and `http.Server` are either too conservative or wide open — both are wrong.

- **Defaults** — `MaxIdleConnsPerHost` is 2. Under high concurrency, requests queue for connections instead of running in parallel. Tune for service-to-service traffic:

```go
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:          100,
        MaxIdleConnsPerHost:   20, // default is 2
        MaxConnsPerHost:       50,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   5 * time.Second,
        ResponseHeaderTimeout: 10 * time.Second,
    },
}
```

- **Crawlers** — when hitting many different hosts, disable keep-alives (`DisableKeepAlives: true`) so idle connections don't accumulate.
- **Server timeouts** — without `ReadTimeout`/`WriteTimeout`/`IdleTimeout` a slow or malicious peer holds connections open indefinitely (Slowloris). Always set them, plus `ReadHeaderTimeout` and `MaxHeaderBytes`.
- **Drain response bodies** — connections return to the pool only after the body is fully read. `Close()` alone isn't enough:

```go
resp, err := client.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
_, _ = io.Copy(io.Discard, resp.Body) // enable connection reuse
```

## Streaming over buffering

`io.ReadAll` pulls a whole stream into memory — fine for payloads you know are small (<1MB), wrong for large ones.

- Stream line-by-line with a `bufio.Scanner` for O(1) memory.
- Copy writer-to-reader with `io.Copy` (32KB internal buffer) instead of staging a byte slice.
- For large JSON, decode incrementally rather than `json.Unmarshal`-ing the whole body:

```go
dec := json.NewDecoder(r)
for dec.More() {
    var item Item
    if err := dec.Decode(&item); err != nil {
        return err
    }
    process(item) // one item at a time
}
```

## JSON performance

The stdlib `encoding/json` reflects over struct fields at runtime; at high throughput that shows up in both CPU and allocations. Options, in rough order:

- **Custom `MarshalJSON`/`UnmarshalJSON`** — eliminate reflection for hot-path types.
- **Code generation** — `easyjson`, `ffjson` produce typed marshal/unmarshal methods at build time.
- **Drop-in replacements** — `goccy/go-json`, `json-iterator/go`, `bytedance/sonic` claim 2-5x over stdlib.
- **`encoding/json/v2`** — default JSON implementation since Go 1.27 (experimental since Go 1.25 behind `GOEXPERIMENT=jsonv2`). It is stricter: duplicate keys and invalid UTF-8 are rejected, so re-run tests against real payloads before relying on it in a hot path.

Check each library's docs for current signatures.

## cgo overhead

Each Go↔C crossing costs roughly 50-100ns, plus a goroutine pinned to an OS thread, plus blocked inlining. Calling C per element pays the crossing per element:

```go
// in a tight loop, prefer pure Go (math.Sqrt is inlineable and fast)
for i, v := range values {
    values[i] = math.Sqrt(v)
}

// if C is unavoidable, batch the crossing
C.batch_sqrt((*C.double)(&values[0]), C.int(len(values)))
```

## Buffered I/O

Unbuffered writes issue a syscall per operation. `bufio.Writer` coalesces small writes into large chunks — 3-10x fewer syscalls on file-heavy work:

```go
w := bufio.NewWriter(f)
for _, line := range lines {
    w.WriteString(line + "\n")
}
w.Flush()
```

## Concurrent multi-stage pipelines

Rare but real: when each stage saturates a *different* resource — A compresses (CPU-bound), B writes disk (I/O-bound), C uploads (network-bound) — running them concurrently with bounded channel buffers keeps all three busy instead of one at a time.

Rules:

- **Only when resources don't overlap.** If two stages compete for the same resource, concurrency adds context switches and nothing else.
- **Only when order doesn't matter** — concurrency reorders records.
- **Per-record latency goes up**; this is a throughput play.
- Throughput becomes `min(A, B, C)` — benchmark the sequential base case first; it's often simpler and still wins.

See `golang-concurrency` for channel patterns and when worker pools fit better.

## Batch operations

Batching amortizes per-operation overhead (syscalls, round-trips, transaction costs) across many items.

- **Database** — one multi-row write beats 1,000 single-row `Exec` calls by orders of magnitude; use multi-row `VALUES` or COPY-style protocols (`pq.CopyIn`). See `golang-database`.
- **HTTP** — when the API supports it, POST the whole batch once instead of one GET per id.
- **Channels** — accumulate items and flush on batch size *or* a ticker, so a slow trickle still gets processed:

```go
func batchProcessor(in <-chan Item, size int) {
    batch := make([]Item, 0, size)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case item, ok := <-in:
            if !ok {
                flush(batch)
                return
            }
            batch = append(batch, item)
            if len(batch) >= size {
                flush(batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                flush(batch)
                batch = batch[:0]
            }
        }
    }
}
```
