# pprof Reference

## Enabling the HTTP server

pprof endpoints leak sensitive runtime state — goroutine stacks, heap contents — and CPU profiling can be abused to DoS a service. **Never expose them without authentication**, and gate them behind an environment switch such as `PPROF_ENABLED`.

### Development (default `net/http` mux)

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // ... rest of app
}
```

### Production (authenticated, custom mux)

```go
import "net/http/pprof"

func setupPprof(mux *http.ServeMux) {
    if os.Getenv("PPROF_ENABLED") != "true" {
        return
    }
    username := os.Getenv("PPROF_USERNAME")
    password := os.Getenv("PPROF_PASSWORD")
    if username == "" || password == "" {
        panic("PPROF_USERNAME and PPROF_PASSWORD must be set when pprof is enabled")
    }
    auth := basicAuth(username, password)

    mux.Handle("/debug/pprof/", auth(http.HandlerFunc(pprof.Index)))
    mux.Handle("/debug/pprof/cmdline", auth(http.HandlerFunc(pprof.Cmdline)))
    mux.Handle("/debug/pprof/profile", auth(http.HandlerFunc(pprof.Profile)))
    mux.Handle("/debug/pprof/symbol", auth(http.HandlerFunc(pprof.Symbol)))
    mux.Handle("/debug/pprof/trace", auth(http.HandlerFunc(pprof.Trace)))

    slog.Info("pprof endpoints enabled (basic auth required)")
}

// basicAuth wraps a handler with HTTP Basic Authentication.
func basicAuth(username, password string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            u, p, ok := r.BasicAuth()
            if !ok || u != username || subtle.ConstantTimeCompare([]byte(p), []byte(password)) != 1 {
                w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Profile types

| Profile | Endpoint | What it shows |
| --- | --- | --- |
| CPU | `/debug/pprof/profile` | where CPU time is spent |
| Heap | `/debug/pprof/heap` | live objects and current allocations |
| Goroutine | `/debug/pprof/goroutine` | stack of every goroutine |
| Block | `/debug/pprof/block` | blocking operations (needs `SetBlockProfileRate`) |
| Mutex | `/debug/pprof/mutex` | lock contention (needs `SetMutexProfileFraction`) |
| Alloc | `/debug/pprof/heap` with `-alloc_space` | cumulative allocations, not the live heap |

## Capturing profiles

```bash
# CPU — sample for at least 30s; exceed your server's request timeout
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Heap snapshot
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Goroutines, human-readable (for reading, not analysis)
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutines.txt

# Goroutines, as a profile (for go tool pprof)
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof

# Goroutine leak profile — generally available since Go 1.27 (no GOEXPERIMENT)
curl http://localhost:6060/debug/pprof/goroutineleak?debug=2
go tool pprof http://localhost:6060/debug/pprof/goroutineleak

# Lock contention and blocking
curl http://localhost:6060/debug/pprof/mutex > mutex.prof
curl http://localhost:6060/debug/pprof/block > block.prof
```

## Analyzing profiles

Quick look:

```bash
go tool pprof cpu.prof                       # interactive prompts
go tool pprof -http=:8080 cpu.prof           # browser flamegraph
go tool pprof -base heap1.prof heap2.prof    # diff two heap snapshots
```

For the interpretation layer (reading `top`/`list`, flat vs cumulative cost, GC churn, leak signatures, and the compiler diagnostics that back them) → the `fabianoflorentino/golang-agent-skills@golang-benchmark` skill.

## Remote profiling

Point the same `curl` commands at the production server address and pass the basic-auth credentials. Profile captures are not free: CPU profiles sample for the requested duration, heap profiles can force extra GC work, and block/mutex profiles add runtime overhead while enabled. Overhead is low when idle, but keep the switched-on window short.

Continuous profiling in production → `fabianoflorentino/golang-agent-skills@golang-observability` (Pyroscope). Investigation-session setup and Prometheus-based performance tracking → `fabianoflorentino/golang-agent-skills@golang-benchmark`.