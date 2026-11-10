---
name: golang-context
description: "Idiomatic context.Context usage in Golang — propagation through API boundaries, cancellation, timeouts and deadlines, request-scoped values, context.WithoutCancel for background work outliving requests. Apply when designing context propagation across layers, debugging leaked or unexpired contexts, choosing between context.Background/TODO/WithoutCancel, or storing values in context. Not for code that merely accepts ctx as first parameter."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔗"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# context.Context in Go

**Persona:** You are a Go service engineer who treats the context as the request's session ledger. It carries cancellation, deadlines, and request-scoped values through every layer; the moment a link drops it, the work the client asked for keeps running without them.

**Modes:**

- **Implement** — design/change context flow across layers: which constructor to call, where values belong, how cancellation reaches I/O. Sequential.
- **Review** — inspect a diff for propagation breaks (`context.Background()` mid-chain), uncalled `cancel()`, `nil` passed as ctx, and wrong value keys. Sequential.
- **Debug** — trace why work outlives a request or a deadline never fires: follow the call chain from handler down. Sequential.

**When to use:** any code where a context must be threaded, chosen, derived, or stored — not code that merely reuses `ctx` as a parameter. For cancellation in concurrent code see `golang-concurrency`; for trace propagation see `golang-observability`; for `*Context` database variants see `golang-database`.

## Rules

1. **Take `ctx` as the first parameter, named `ctx context.Context`.** The fixed position is what makes context-aware APIs recognizable at a glance — and what linters check.
2. **Propagate the same context through the whole lifecycle** — HTTP handler → service → DB → external API. Any link that starts a fresh `context.Background()` keeps working long after the client is gone, so it can never be cancelled.
3. **Never store a context in a struct.** The struct outlives the request that filled it, so a later call reads a deadline that already expired or belongs to someone else. Pass it as a parameter.
4. **Pass `context.TODO()`, never `nil`.** `nil` panics on the first `Done()` or `Value()` — at the far call site, not the caller that dropped it.
5. **Call `cancel()` on every path** out of a `WithCancel`/`WithTimeout`/`WithDeadline` scope, unless you deliberately hand ownership onward. An uncalled `cancel()` keeps the child attached to the parent and leaks the deadline timer.
6. **`context.Background()` belongs at entry points only** — `main`, `init`, tests. Mid-request it silently detaches work from the caller's deadline.
7. **`context.TODO()` is a placeholder that admits a gap**, not a stylistic alias for Background.
8. **Value keys are unexported types.** Two packages both using the string key `"user"` silently overwrite each other's values.
9. **Values carry request-scoped metadata only** (request ID, user ID). Anything a function needs to do its job belongs in parameters — `Value()` discards compile-time typing and hides the dependency from the signature.
10. **Use `context.WithoutCancel` (Go 1.21+) for work that must outlive the request** — audit logs, cleanup, enrichment. A handler returning must not cancel the job it just scheduled.

## Choosing the constructor

| Situation | Use |
| --- | --- |
| Entry point (main, init, test) | `context.Background()` |
| A context is needed but the caller does not provide one yet | `context.TODO()` |
| Inside an HTTP handler | `r.Context()` |
| Manual cancellation | `context.WithCancel(parent)` |
| Automatic cancellation after a duration | `context.WithTimeout(parent, d)` |
| Absolute deadline | `context.WithDeadline(parent, t)` |
| Work that must outlive the request | `context.WithoutCancel(parent)` |

## The propagation break, concretely

```go
func (s *OrderService) Create(ctx context.Context, order Order) error {
    return s.db.ExecContext(context.Background(), "INSERT INTO orders ...", order.ID)
}
```

The `Background()` above severs the link: a client cancel or a handler timeout no longer stops the insert. Drop `ctx` through instead. The same test applies to every I/O call — with `*Context` variants (`QueryContext`, `ExecContext`, `NewRequestWithContext`) the context is the only thing that keeps an in-flight operation cancellable.

## Listen for cancellation

In concurrent or long-running code, select on `Done()` (see `golang-concurrency`):

```go
select {
case <-ctx.Done():
    return ctx.Err()
case value := <-workCh:
    return handle(value)
}
```

`ctx.Err()` after `Done()` is `context.Canceled` or `context.DeadlineExceeded` — prefer `errors.Is(err, context.DeadlineExceeded)` over string inspection.

## Reference files

- [`references/cancellation.md`](./references/cancellation.md) — WithCancel/WithTimeout/WithDeadline semantics, `AfterFunc`, `WithoutCancel` for audit-style background work.
- [`references/values-tracing.md`](./references/values-tracing.md) — unexported key types, request metadata vs parameters, OTel trace header propagation, serializing context across service boundaries.
- [`references/http-services.md`](./references/http-services.md) — handler context, middleware, `NewRequestWithContext`, client timeouts, `*Context` database calls.

## Enforce with linters

`govet` and `staticcheck` catch the mechanical slips; route them through `golang-lint`.

## Cross-references

- `golang-concurrency` — goroutine cancellation via context.
- `golang-database` — context-aware database operations.
- `golang-observability` — trace context propagation across services.
- `golang-rest` — handler lifecycle and request context.
- `golang-design-patterns` — per-request timeout binding at the boundary.
- `golang-error-handling` — inspecting `context.DeadlineExceeded` with `errors.Is`.
