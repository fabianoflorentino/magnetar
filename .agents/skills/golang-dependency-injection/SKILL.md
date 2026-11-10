---
name: golang-dependency-injection
description: "Comprehensive guide for dependency injection (DI) in Golang. Covers why DI matters (testability, loose coupling, separation of concerns, lifecycle management), manual constructor injection, and DI library comparison (google/wire, uber-go/dig, uber-go/fx, samber/do). Use this skill when designing service architecture, setting up dependency injection, refactoring tightly coupled code, managing singletons or service factories, or when the user asks about inversion of control, service containers, or wiring dependencies in Go. For a specific DI library, → See `fabianoflorentino/golang-agent-skills@golang-google-wire`, `fabianoflorentino/golang-agent-skills@golang-uber-dig`, `fabianoflorentino/golang-agent-skills@golang-uber-fx`, or `fabianoflorentino/golang-agent-skills@golang-samber-do` skills."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔌"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion
paths:
  - "**/*.go"
---

**Persona:** You are a Go software architect. You pick the simplest DI approach that solves the problem, guide teams toward testable designs, and never over-engineer a wiring problem.

**Modes:**

- **Design** — for a new service or an existing DI setup: assess the graph and lifecycle needs, pick manual injection or a library from the decision table, then generate the wiring.
- **Refactor** — for a coupled codebase: fan out up to three parallel sub-agents — one maps global variables and `init()` service setup, one maps concrete-type dependencies that should become interfaces, one locates service-locator anti-patterns (container passed as an argument) — then consolidate into a migration plan.

**When to use:** any task about wiring dependencies, inversion of control, or service containers. Library-specific usage lives in `golang-google-wire`, `golang-uber-dig`, `golang-uber-fx`, and `golang-samber-do`.

## What DI is for

DI means passing dependencies in rather than letting a component create or find them. That pays off as:

- **Testability** — fakes plug in at the seam.
- **Looseness** — components depend on interfaces, not each other's internals.
- **Predictable bootstrap** — wiring happens once, up front, instead of drifting through `init()`.

It earns its keep in applications with many interconnected services. A 3-function script is fine with manual wiring — do not over-engineer.

## Manual constructor injection first

Small projects (< ~10 services) should stay manual:

```go
type UserService struct {
    store  UserStore
    mailer Mailer
    log    *slog.Logger
}

func NewUserService(store UserStore, mailer Mailer, log *slog.Logger) *UserService {
    return &UserService{store: store, mailer: mailer, log: log}
}

func main() {
    db := postgres.NewUserStore(connStr)
    svc := NewUserService(db, smtp.NewMailer(smtpAddr), slog.Default())
    api := NewAPI(svc, slog.Default())
    api.ListenAndServe(":8080")
}
```

A hidden dependency in a constructor is the failure this pattern prevents:

```go
// ✗ service discovers its own DB behind a magic environment read
func NewUserService() *UserService {
    db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    return &UserService{store: db}
}
```

Manual DI starts to strain around 15+ services with cross-dependencies, lifecycle needs, lazy init, or scoped containers.

## Library comparison

| Dimension | Manual | google/wire | dig + fx | samber/do |
| --- | --- | --- | --- | --- |
| Size sweet spot | small | medium-large | large | any |
| Type safety | compile-time | compile-time (codegen) | runtime (reflection) | compile-time (generics) |
| Codegen | none | `wire_gen.go` | none | none |
| Lazy loading | manual | eager | fx built-in | built-in |
| Lifecycle / shutdown | manual | manual | fx built-in | built-in |
| Scopes / modules | manual | sets | fx modules | hierarchical |
| Health checks | manual | manual | manual | built-in |
| Minimum Go | any | any | any | 1.18+ |

The same small graph, wired by hand vs by container:

```go
// Manual — you maintain the order; new deps require editing call sites downstream
cfg := NewConfig()
db := NewDatabase(cfg)
svc := NewUserService(NewUserStore(db))
api := NewAPI(svc)

// Container — order derives from constructor signatures
i := do.New()
do.Provide(i, NewConfig, NewDatabase, NewUserStore, NewUserService)
api := do.MustInvoke[*API](i)
defer i.Shutdown()
```

- **wire** compiles that manual sequence at build time from a `wire.Build` list — cleanup flows through returned `func()`, no lifecycle hooks.
- **dig + fx** resolve by reflection at runtime; fx adds `OnStart`/`OnStop` hooks, modules, and a signal-aware loop.
- **samber/do** stays compile-time via generics with built-in health checks and shutdown.

Per-library wires: [google-wire](references/google-wire.md), [uber-dig-fx](references/uber-dig-fx.md), [samber-do](references/samber-do.md).

## Testing with DI

DI delivers its value when tests swap implementations at the boundary:

```go
type MockStore struct{ users map[string]*User }

func (m *MockStore) FindByID(ctx context.Context, id string) (*User, error) {
    u, ok := m.users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return u, nil
}
```

`NewUserService(mock, nil, slog.Default())` then drives the service without a database. Container-based tests clone the real injector and override only the boundary (see `golang-samber-do`'s `Clone`/`Override` and fx's `fx.Replace`/`fxtest`).

## When to adopt a library

| Signal | Action |
| --- | --- |
| < 10 services, simple deps | manual constructor injection |
| 10–20 services, some cross-cutting concerns | consider a container |
| 20+ services or lifecycle/health needs | strongly consider a container |
| Team unfamiliar with DI | start manual, migrate incrementally |

## Common mistakes

| Mistake | Fix |
| --- | --- |
| Global variables as dependencies | pass through constructors |
| `init()` for service setup | explicit wiring in `main()` |
| Depending on concrete types | accept interfaces at consumption boundaries |
| Passing the container everywhere | inject the specific dependencies |
| Deep chains (A→B→C→D→E) | depend on repositories/config directly |
| New container per request | one container per app; scopes for per-request state |

## Cross-references

- `golang-samber-do` — generics-based containers.
- `golang-google-wire` — compile-time codegen.
- `golang-uber-dig` / `golang-uber-fx` — reflection runtime, fx lifecycle.
- `golang-structs-interfaces` — interface design and composition.
- `golang-testing` — testing patterns that DI unlocks.
- `golang-project-layout` — where the composition root lives.
