---
name: golang-design-patterns
description: "Idiomatic Golang design patterns — functional options, constructor APIs, `init()` and global-state avoidance, enums, panic vs error decisions, resource management and lifecycle, graceful shutdown, timeouts and retries, streaming and iterators, and architecture styles (clean, hexagonal, DDD, flat). Apply when choosing between architectural patterns, implementing functional options, designing constructor APIs, setting up graceful shutdown, applying resilience patterns, or asking which idiomatic Go pattern fits a specific problem. Not for wiring a DI container or comparing DI libraries (→ See `fabianoflorentino/golang-agent-skills@golang-dependency-injection` skill), nor for error wrapping, `errors.Is`/`As`, or logging mechanics (→ See `fabianoflorentino/golang-agent-skills@golang-error-handling` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🏗"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
paths:
  - "**/*.go"
---

# Design patterns & idioms in Go

**Persona:** You are a Go architect who values simplicity and explicitness. A pattern earns its place by solving a real problem, not by indicating sophistication — and premature abstraction gets pushed back on.

**Modes:**

- **Design** — new APIs, packages, structure: ask about the developer's architecture preference first, favor the smallest pattern that satisfies the requirement. Sequential.
- **Review** — audit for `init()` abuse, unbounded resources, missing timeouts, implicit global state; report before refactoring. Sequential.

**When to use:** architecture and pattern choices — constructors, options, enums, panic-vs-error, lifecycles, resilience, streaming, and clean/hexagonal/DDD/flat styles. For DI containers → `golang-dependency-injection`; for error mechanics → `golang-error-handling`.

## The rules that carry the most weight

1. **Functional options for constructors** — one function per option, additive without breaking changes. Options MUST return an error when validation can fail, so bad config is caught at construction.
2. **Avoid `init()`** — it runs implicitly, cannot return errors, and makes tests unpredictable. Explicit constructors are injectable and order-independent.
3. **Enums start at an `Unknown` sentinel (0)** — the zero value must not silently be a real state.
4. **Panic is for bugs, not expected failures.** Expected conditions return errors; a panic crashes the process out from under handlers.
5. **`defer Close()` immediately after opening** — later edits cannot silently skip cleanup.
6. **`runtime.AddCleanup` over `SetFinalizer`** — finalizers are unpredictable and can resurrect objects.
7. **Every external call has a timeout** — a slow upstream hangs your goroutine indefinitely (`golang-context`).
8. **Limit everything** — pool sizes, queue depths, buffers. Unbounded resources grow until they crash.
9. **Retries check `ctx.Err()` between attempts** and back off via `select` on `ctx.Done()`.
10. **A little recode beats a big dependency** — every dependency adds attack surface and maintenance.

## Functional options

```go
type Server struct {
    addr         string
    readTimeout  time.Duration
    writeTimeout time.Duration
    maxConns     int
}
type Option func(*Server)

func WithReadTimeout(d time.Duration) Option { return func(s *Server) { s.readTimeout = d } }
func WithMaxConns(n int) Option              { return func(s *Server) { s.maxConns = n } }

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{addr: addr, readTimeout: 5 * time.Second,
        writeTimeout: 10 * time.Second, maxConns: 100}
    for _, o := range opts {
        o(s)
    }
    return s
}
```

Reach for an explicit builder only when configuration steps genuinely validate against each other.

## init() and global state

`init()` order (file declaration then filename order) is fragile, errors either panic or `log.Fatal`, and side effects before `main` make tests unpredictable:

```go
var db *sql.DB
func init() { var err error; db, err = sql.Open("postgres", os.Getenv("DATABASE_URL")); ... } // bad

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }       // good
```

## Enums

```go
type Status int
const (
    StatusUnknown Status = iota // 0 = unset
    StatusActive
    StatusInactive
)
```

## Compile regexp once; embed static assets

```go
var emailRegex = regexp.MustCompile(`^...$`)   // package level, once

//go:embed templates/*
var templateFS embed.FS
```

## Panic vs error, concretely

- **Return error:** network failures, missing rows, invalid input — anything a caller can act on.
- **Panic:** nil where the code proved it impossible, violated invariants, `Must*` at init time.
- **Close/Flush errors:** read-only cleanup can `defer f.Close()`, but durability-sensitive writes must report close/flush errors.

## string vs []byte vs []rune

| Type | Default for | Reach for it |
| --- | --- | --- |
| `string` | Display, keys | Immutability and safety |
| `[]byte` | I/O | `io.Writer`, building, mutation |
| `[]rune` | Unicode ops | `len()` must mean characters |

Conversions allocate — stay in one type until you genuinely need another.

## Iterators and streaming

Go 1.23+ `range`-able iterators give lazy evaluation — don't load a million rows into memory. Stream large transfers (DB to HTTP) so memory stays flat; see the data-handling reference for shapes.

## Timeouts and retries

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
resp, err := httpClient.Do(req.WithContext(ctx))
```

Retry loops must check `ctx.Err()` between attempts and park on `ctx.Done()` during backoff. Long loops check `ctx.Err()` periodically.

## Graceful shutdown and pools

Serve/worker loops tie their lifecycle to a `context.Context`; on cancel, stop accepting work, drain in-flight requests, then exit. Pool/lifetime patterns and `runtime.AddCleanup` usage: [`references/resource-management.md`](./references/resource-management.md).

## Database patterns

Wiring, transactions, nullables, pools, repository shapes: `golang-database`.

## Architecture

Ask which style the project uses — clean, hexagonal, DDD, or flat — and don't impose ceremony on a small project. Non-negotiables regardless of style:

- Domain layer stays free of framework imports.
- Validate at boundaries (`fail fast`), trust internal code.
- Make illegal states unrepresentable with types.
- Respect 12-factor principles (`golang-project-layout`).

Per-style guides: [`architecture.md`](./references/architecture.md), [`clean-architecture.md`](./references/clean-architecture.md), [`hexagonal-architecture.md`](./references/hexagonal-architecture.md), [`ddd.md`](./references/ddd.md).

## Reference files

- [`references/data-handling.md`](./references/data-handling.md) — iterators, streaming, string/byte choices.
- [`references/resource-management.md`](./references/resource-management.md) — lifecycle, pools, graceful shutdown, cleanup.
- Architecture references listed above.

## Cross-references

- `golang-data-structures` — structure selection and internals.
- `golang-error-handling` — wrapping, sentinels, single handling rule.
- `golang-structs-interfaces` — interface design and composition.
- `golang-concurrency` — goroutine lifecycle and graceful shutdown.
- `golang-context` — timeout and cancellation patterns.
- `golang-project-layout` — architecture and directory structure.
- `golang-refactoring` — staging a migration toward these patterns safely.
