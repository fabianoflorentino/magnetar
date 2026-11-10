---
name: golang-error-handling
description: "Idiomatic Golang error handling — creation, wrapping with %w, errors.Is/As, errors.Join, custom error types, sentinel errors, panic/recover, the single handling rule, structured logging with slog, HTTP request logging middleware, and samber/oops for production errors. Built to make logs usable at scale with log aggregation 3rd-party tools. Apply when creating, wrapping, inspecting, or logging errors in Go code. For samber/oops specifics → See `fabianoflorentino/golang-agent-skills@golang-samber-oops` skill; for slog handler ecosystem → See `fabianoflorentino/golang-agent-skills@golang-samber-slog` skill."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "⚠️"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Error handling in Go

**Persona:** You are a Go reliability engineer. An error is an event that must either be handled where it can be decided, or propagated with enough context to be decided one layer up. Silent discards and duplicate log lines are both defects.

**Modes:**

- **Coding** — write new error code: produce errors with context (`%w`), decide sentinel vs custom type, and apply the single-handling rule. Grep adjacent code for previous violations and fix them in place.
- **Review** — review a diff for swallowed errors, missing `%w`, log-and-return pairs, and `panic` misuse. Sequential.
- **Audit** — sweep a codebase. Split by category: error creation, wrapping/inspection, single-handling-rule violations, panic/recover, structured logging. Parallel sub-agents by category.

**When to use:** any Go code that creates, wraps, tests, or logs errors. For rich production errors with stack/attributes use `golang-samber-oops`; for slog handler tuning use `golang-samber-slog`; for HTTP error surfaces see `golang-rest`; for panic-safe boundaries see `golang-safety`.

## Decide how errors are created

| What you have | Use | Why |
| --- | --- | --- |
| An expected, recurring condition | Sentinel `errors.New("...")` | `errors.Is` matching, single allocation |
| A condition carrying structured data | Custom type + `%w` constructor | `errors.As`/`AsType` extracts fields |
| A plain, local transport failure | `fmt.Errorf("ctx: %w", err)` | context chain, no ceremony |

- Message conventions: start lowercase, no trailing punctuation, describe what happened without prescribing an action.
- Predeclare sentinels once at package scope; never allocate a new one per site.

## Wrap with chains, not strings

`fmt.Errorf("querying users: %w", err)` keeps the chain inspectable. `%v` flattens it into an opaque string and `errors.Is`/`As` stop working. The `%w` verb is the contract; use `%v` deliberately at process boundaries when you mean to hide internals (for example before logging user-visible messages).

## Inspect with errors.Is / errors.As

- `errors.Is(err, ErrNotFound)` matches sentinels across the chain.
- `errors.As(err, &target)` extracts the first matching custom type. Go 1.26+ adds `errors.AsType[T](err)` when `T` implements `error` — type-safe, no `&target` ceremony and no empty-struct trap.
- Direct `err == sentinel` and bare `err.(SomeType)` break as soon as a `%w` appears in between. Do not use them for deciding control flow.

## Combine independent failures with errors.Join

When several operations fail and you must report all of them (batch processing, multi-PR merge checks), `errors.Join(err, err2, ...)` (Go 1.20+) keeps each element inspectable with `Is`/`As`. Prefer it over string-concatenated messages or a custom slice type unless you need to append incrementally.

## The single handling rule

An error is **logged or returned, never both**. A `log-and-return` pair means the same failure is printed once at each frame that adds a `%w`, and aggregators fill with duplicate lines.

```go
if err != nil {
    slog.Error("query failed", "error", err)  // log ...
    return fmt.Errorf("users: %w", err)        // AND return -> duplicate
}

if err != nil {
    return fmt.Errorf("users: %w", err)        // return with context; log once at the boundary
}
```

Decide the boundary where an error stops being an error and becomes an event to log: typically the HTTP handler or the top of a worker loop. Below it, only wrap and return; at it, only log.

## Never use panic for business failures

`panic` belongs to genuinely unrecoverable states (invariant corruption, programmer errors). Expected conditions — validation, missing rows, closed connections — are returned errors. Inside a service, recovery belongs at the goroutine boundary and converts to `500`/termination; see `golang-rest` for the HTTP conversion, `golang-concurrency` for goroutine discipline.

## Structured logging with slog

Go 1.21+ `log/slog` is the default logger. Emit key/value pairs, not formatted prose:

- Use the `*Context` variants so `trace_id`/`span_id` ride along (`slog.InfoContext(ctx, "order created", "order_id", id)`); see `golang-observability`.
- Choose levels with intent: `Debug` dev noise, `Info` normal ops, `Warn` degraded, `Error` requires attention.
- Log HTTP traffic once per request with a middleware capturing method, route pattern, status, and duration — never from inside handlers.
- **Keep grouping low-cardinality:** stick to stable message templates and attach IDs, paths, line numbers, and counts as structured attributes. One request ID on each line is better than a unique message string per line; APM/aggregators group by template.

## Logging HTTP requests

One middleware, fields: method, `otelStatus`, route pattern (not raw path — raw paths blow up cardinality), duration, and a request id. See `golang-observability` for the full middleware and examples, and `golang-rest` for status-code mapping.

## What users see vs what you log

Never send raw technical errors to a user. Translate at the HTTP boundary: internal `%w` chain goes to the log; the response body carries a stable machine code and a safe message (RFC 9457 in `golang-rest`). Technical detail in the log, low-cardinality in the aggregation keys.

## Audit checklist

- [ ] Returned errors are never discarded with `_`
- [ ] Every `fmt.Errorf` uses `%w` unless there is a stated hiding reason
- [ ] Matching always via `errors.Is`/`As`/`AsType`, never `==`/bare assertion
- [ ] No log-and-return doubles
- [ ] `panic` absent from normal control flow
- [ ] Messages lowercase, no trailing punctuation, no PII
- [ ] Log fields keyed and low-cardinality; message templates stable

## Cross-references

- `golang-samber-oops` — stack traces, attributes, and `%+v` formatting for production errors.
- `golang-samber-slog` — handler ecosystem (tint, formatters, sampling, HTTP).
- `golang-observability` — slog setup, request middleware, trace correlation.
- `golang-rest` — RFC 9457 error mapping, never leaking internals.
- `golang-safety` — nil error interface trap, nil comparison pitfalls.
- `golang-naming` — `ErrXxx`/`XxxError` naming conventions.
- `golang-testing` — asserting error semantics with `errors.Is`/`As`.
- `golang-pitfalls-error-handling` — common misuse patterns distilled from 100 Go Mistakes.
