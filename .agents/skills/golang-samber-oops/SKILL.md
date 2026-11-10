---
name: golang-samber-oops
description: "Structured error handling in Golang with samber/oops — error builders, stack traces, error codes, error context, error wrapping, error attributes, user-facing vs developer messages, panic recovery, and logger integration. Apply when using or adopting samber/oops, or when the codebase already imports github.com/samber/oops."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "💥"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who treats errors as structured data. Every error carries enough context — domain, attributes, a trace — for an on-call engineer to diagnose it without hunting the developer.

**Modes:**

- **Build** — adding structured context to errors in a codebase.
- **Review** — checking that variable data lives in attributes (not messages), that layers wrap at package boundaries, and that goroutines recover.

**When to use:** any task centered on samber/oops. General error-wrapping philosophy lives in `golang-error-handling`; embedding `slog` values and APM conventions in `golang-observability`.

## The core idea

`samber/oops` upgrades standard errors with machine-readable structure. The headline rule: **variable data goes in attributes, never in the message**. A message like "failed to process user u-123 in tenant acme" shatters APM grouping in Datadog/Loki/Sentry; the static message plus `With("user_id", u)` attributes keeps every instance of the same failure aggregated.

Attributes also travel with the error through the call stack — unlike adding `slog` attributes only at the log site, they survive wrapping and arrive fully-stocked at the error boundary.

## The fluent builder

```go
err := oops.
    In("user-service").
    Tags("database", "postgres").
    Code("network_failure").
    User("user-123", "email", "foo@bar.com").
    With("query", query).
    Errorf("failed to fetch user")
```

Terminal methods: `Errorf` (new error), `Wrap`/`Wrapf` (wrap an existing error), `Join` (combine several), `Recover`/`Recoverf` (panic → error).

Builder slots worth knowing:

| Method | Purpose |
| --- | --- |
| `.With(k, v)` | custom attribute; lazy `func() any` values supported |
| `.WithContext(ctx, ks...)` | pull values out of a Go context into attributes |
| `.In(domain)` | feature/service tag |
| `.Tags(...)` | categorization — query later with `err.HasTag` |
| `.Code(slug)` | machine-readable ID |
| `.Public(msg)` | user-safe message, decoupled from technical detail |
| `.Hint(...)` | runbook/pointer for the developer |
| `.Owner(team)` | accountability marker |
| `.User(id, ...)` / `.Tenant(id, ...)` | subject context |
| `.Trace(id)` / `.Span(id)` | correlation ids (ULID when omitted) |
| `.Request(r, bool)` / `.Response(r, bool)` | attach HTTP traffic |
| `.Since(t)` / `.Duration(d)` | timing for the error |
| `oops.FromContext(ctx)` | resume from a builder stashed in the context |

## Layering: wrap once per boundary

`Wrap*` returns `nil` when given `nil`, so the nil-check wrapper is pure noise:

```go
// ✓
return oops.Wrapf(err, "product lookup failed")
// ✗
if err != nil {
    return oops.Wrapf(err, "product lookup failed")
}
return nil
```

Add context once per architectural layer, not on every function call:

```go
func Handler() error { return oops.In("http").Trace(traceID).Wrapf(Service(), "create user failed") }
func Service() error { return oops.In("service").With("op", "create_user").Wrapf(Repo(), "db failed") }
func Repo() error    { return oops.In("repo").Tags("postgres").Errorf("connection timeout") }
```

Low-cardinality everywhere:

```go
// ✗ high-cardinality; every value re-fragments the group
oops.Errorf("failed on user %s in tenant %s", uid, tid)
// ✓ static message + structured attributes
oops.With("user_id", uid).With("tenant_id", tid).Errorf("failed to process user")
```

## Panics at goroutine boundaries

`Recover` converts a panic into an error, and it is non-negotiable anywhere a goroutine starts:

```go
func Process(data []byte) (err error) {
    return oops.In("data-processor").Code("panic_recovered").
        With("input_bytes", len(data)).
        Recover(func() { risky(data) })
}
```

## Reading the result

`oops` errors still implement `error`. Peel the wrapper to reach the metadata:

```go
var e oops.OopsError
if errors.As(err, &e) {
    code, dom := e.Code(), e.Domain()
    tags, ctx := e.Tags(), e.Context()
    trace := e.Stacktrace()
}
public := oops.GetPublic(err, "Something went wrong")
```

Serialization is plug-and-play: `%+v` prints the stack, `json.Marshal` yields fields for log pipelines, and `slog.Error(msg, slog.Any("error", err))` embeds everything into one record.

## Carrying builders through middleware

Stash a builder in the request context and let handlers start from it:

```go
b := oops.In("http").Request(r, false).Trace(r.Header.Get("X-Trace-Id"))
ctx := oops.WithBuilder(r.Context(), b)

return oops.FromContext(ctx).Tags("handler", "users").Errorf("something failed")
```

Assertions, config, additional logger wiring: [advanced patterns](references/advanced.md).

## Best practices

1. Attributes for variable data, static messages for grouping.
2. Wrap once per layer; let the chain accumulate context.
3. `Recover` at every goroutine boundary.
4. `Public` for anything crossing into an HTTP response; keep internals out.
5. Give errors codes and owners so alerting can page the right team.

## Cross-references

- `golang-error-handling` — sentinel/wrapping idioms and the log-or-return rule.
- `golang-observability` — slog handlers, APM grouping, and trace propagation.
- `golang-samber-slog` — shipping `oops` errors through a slog pipeline.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.
