# Production Debugging

## When paged: capture before touching the process

Do **not** restart first. A restart destroys the evidence. Collect, in order:

1. **Profiles** — against the production pprof endpoint (see [pprof.md](./pprof.md)): goroutine dump (`?debug=2`), heap, 30s CPU, mutex.
2. **System metrics**:

   ```bash
   ps aux | grep myapp
   lsof -p PID | wc -l           # open file descriptors
   ss -s                          # socket summary
   netstat -an | grep ESTABLISHED | wc -l
   ```

3. **Analyze offline** — pull the `.prof` files down and inspect locally with `go tool pprof`.

## Logging and observability

### Place logs at component boundaries

Log where data crosses layers, not everywhere. Entry/exit with key parameters, around external calls, and at decision points — so you can see which component corrupts or drops the data:

```go
func ProcessOrder(ctx context.Context, orderID string) error {
    log.Printf("ProcessOrder: start orderID=%s", orderID)
    defer log.Printf("ProcessOrder: done orderID=%s", orderID)
    // ...
}

log.Printf("calling payment API for order %s", orderID)
resp, err := paymentClient.Charge(ctx, req)
if err != nil {
    log.Printf("payment API: err=%v", err)
}
```

### Structured logging

Use `log/slog` for queryable, key/value-shaped output (Go 1.21+):

```go
slog.Info("processing request",
    "method", r.Method,
    "path", r.URL.Path,
    "user_id", userID,
)

slog.Error("database query failed",
    "err", err,
    "query", query,
    "duration_ms", elapsed.Milliseconds(),
)
```

### Request ID tracing

Correlate the log lines of a single request across layers:

```go
type ctxKey string

func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, ctxKey("request_id"), id)
}

func RequestID(ctx context.Context) string {
    id, _ := ctx.Value(ctxKey("request_id")).(string)
    return id
}
```

Attach the ID in middleware, read it from `ctx` at every log site.

### Fetch: connecting visibility to the evidence

The gap between "capture logs" and "find the failing request" is spanned by the `golang-observability` skill — metrics, traces, and alerting produce the failing request ID, and the request-ID middleware above lets you pull its full path. Use both halves: the observability layer points at the candidate, the boundary logs confirm it.

## HTTP client issues

```go
// 1. Never use the zero http.Client — it has no timeout
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        DialContext:          (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
        TLSHandshakeTimeout: 5 * time.Second,
        IdleConnTimeout:     90 * time.Second,
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

// 2. Always close the body
resp, err := client.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()

// 3. Read the body on error status — it usually says what broke
if resp.StatusCode >= 400 {
    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("API error %d: %s", resp.StatusCode, body)
}

// 4. Dump the wire traffic when the shape is in doubt
dump, _ := httputil.DumpRequestOut(req, true)
log.Printf("request:\n%s", dump)
dump, _ = httputil.DumpResponse(resp, true)
log.Printf("response:\n%s", dump)
```