---
name: golang-pitfalls-standard-library
description: "Golang standard library — providing a wrong time.Duration unit, time.After memory leaks, encoding/json common mistakes (embedding, monotonic clock, map[string]any numbers), database/sql connection handling and prepared statements, not closing transient io.Closer resources, forgetting the return after http.Error, and using the default HTTP client and server without timeouts. Distilled from mistakes #75-81 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang code using time, encoding/json, database/sql, io, or net/http."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📚"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Standard Library

Source material: mistakes #75-81 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when working with the Go standard library.

## 75. Providing a wrong time duration (#75)

- `time.Duration` is an alias for `int64`; **one unit = one nanosecond** (not a millisecond).
- `time.NewTicker(1000)` ticks every 1000 ns = 1 µs, not 1 second.
- Always express durations through the `time` API: `time.Second`, `1000 * time.Nanosecond`, `time.Millisecond`. Avoid raw numbers.

## 76. `time.After` and memory leaks (#76)

- Not relevant anymore from Go 1.23 — the internal timer implementation was fixed. On older Go versions, `time.After` inside a loop kept timers alive until fired; use `time.NewTimer`/`time.NewTicker` with `defer Stop()` if you target pre-1.23.

## 77. JSON handling common mistakes (#77)

- **Type embedding**: an embedded field implementing `json.Marshaler` (e.g., `time.Time`) overrides default marshaling behavior of the whole struct and can sneakily change the output/schema. Be deliberate about embedding in JSON structs.
- **Monotonic clock**: `time.Time` holds both wall and monotonic clocks; `==` compares both, so two otherwise-equal times with different monotonic readings are unequal. Strip the monotonic part (e.g., round-trip through `time.RFC3339`) before comparing or store without monotonic reading.
- **Map of `any`**: when unmarshaling JSON into `map[string]any`, all numbers become `float64` by default — expect floating-point rounding surprises. Confirm the intended numeric type or use a typed struct / `json.Decoder.UseNumber()`.

## 78. Common SQL mistakes (#78)

- `sql.Open` does **not** establish a connection. Call `Ping`/`PingContext` to validate config and reachability.
- Configure connection pooling parameters for production (limits, idle connections, lifetimes).
- Use **prepared statements** — more efficient and more secure (SQL injection protection).
- Handle nullable columns with pointers or `sql.NullXXX` types.
- After iterating `sql.Rows`, call `rows.Err()` to catch errors preparing the next row — a `nil` + premature end can hide failures.

## 79. Not closing transient resources (#79)

- Always close things implementing `io.Closer` (HTTP response bodies, `sql.Rows`, `os.File`) to avoid leaks.
- Use `defer` for close, but beware `defer` inside loops (see mistake #35) and handle close errors (see #54).

## 80. Forgetting the return after an HTTP `http.Error` (#80)

- `http.Error` does **not** stop the handler. Without `return`, the success body/status is also written, causing a superfluous `WriteHeader` and an incorrect response.

```go
if err := foo(req); err != nil {
    http.Error(w, "foo", http.StatusInternalServerError)
    return // required
}
```

## 81. Using the default HTTP client and server (#81)

- The zero-value `http.Client`/`http.Server` have no timeouts — production code should configure them explicitly:
  - Client: `Timeout`, `Transport` (dial/TLS/handshake/response-header timeouts).
  - Server: `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `ReadHeaderTimeout`, `MaxHeaderBytes`.
- Missing timeouts can hang goroutines, connections, and memory indefinitely.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-error-handling` for closing resources and defer error handling
- → See `fabianoflorentino/golang-agent-skills@golang-database` for connection pooling, transactions, and migrations
- → See `fabianoflorentino/golang-agent-skills@golang-observability` for request logging middleware and production HTTP configuration
- → See `fabianoflorentino/golang-agent-skills@golang-testing` for testing HTTP handlers and io.Reader/Writer edge cases
