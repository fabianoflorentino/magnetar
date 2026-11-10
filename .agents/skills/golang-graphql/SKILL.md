---
name: golang-graphql
description: "Implements GraphQL APIs in Golang using gqlgen or graphql-go. Apply when building GraphQL servers, designing schemas, writing resolvers, handling subscriptions, or integrating GraphQL with existing Go HTTP services. Also apply when the codebase imports `github.com/99designs/gqlgen` or `github.com/graph-gophers/graphql-go`."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔮"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(curl:*) Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer building data-graph APIs. You design the schema before the resolvers, batch data access so every depth stays cheap, and treat query complexity caps as a production requirement, not a nicety.

**Modes:**

- **Build** — scaffolding a schema and its resolvers, or adding operations to an existing API: first grep for the project's resolver and naming conventions, then generate.
- **Review** — auditing a GraphQL codebase or PR: scout for per-field query storms (N+1), global DataLoader singletons, missing complexity limits, and introspection left on in production.
- **Debug** — a resolver that errors, deadlocks, or times out under load: trace from the client query to the resolver stack.

**When to use:** any task centered on a GraphQL API in Go. REST and gRPC consumers should use `golang-rest` or `golang-grpc` instead. Load `golang-database` and `golang-observability` alongside for the batching and tracing concerns below.

## Library choice

| Library | Style | Type safety | Build step | Pick when |
| --- | --- | --- | --- | --- |
| `github.com/99designs/gqlgen` | schema-first codegen | compile-time | `go generate` | large schemas, federation, strict types |
| `github.com/graph-gophers/graphql-go` | schema-first, reflection | parse-time | none | compact schemas, fast iteration |
| `github.com/graphql-go/graphql` | code-first | runtime | none | not recommended — verbose, no SDL |

Choose gqlgen for 100+ types, Apollo Federation, or when generated stubs and zero reflection matter. Choose graph-gophers for small/medium schemas and a minimal toolchain. Both are schema-first: you author `.graphql` SDL and bind Go resolvers.

For each library's deep-dive, see [gqlgen reference](references/gqlgen.md) and [graphql-go reference](references/graphql-go.md).

## Schema design

```graphql
type User {
  id: ID!
  email: String!
  bio: String
  posts(first: Int = 10, after: String): PostConnection!
}
```

Mark a field `!` only when the server can always return a value. A failing non-null field nulls its whole parent — one broken resolver silently blanks an object; a nullable field only blanks itself.

Use cursor connections (`Connection`/`Edge`/`PageInfo`) for lists so pagination stays stable under inserts and deletes during traversal.

Envelope mutations so business errors and partial results survive without polluting the GraphQL `errors` array:

```graphql
type CreateUserPayload {
  user: User
  errors: [UserError!]!
}
```

## Resolvers

Keep resolvers thin: translate GraphQL inputs to domain calls, and domain results back to GraphQL types. Route data access through a service layer so the resolver stays a mapper.

```go
func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.CreateUserPayload, error) {
    if err := input.Validate(); err != nil {
        return nil, err
    }
    user, err := r.users.Create(ctx, input.Email, input.Name)
    if err != nil {
        return nil, presentError(err)
    }
    return &model.CreateUserPayload{User: toGQLUser(user)}, nil
}
```

Split resolvers by type (`userResolver`, `postResolver`) instead of cramming every field's method onto one struct; gqlgen does this automatically with `resolver: true` on federated or data-backed fields.

## N+1 prevention with DataLoaders

A per-field resolver that queries the database fires one SQL round-trip per parent row. DataLoaders coalesce those calls into a single batched query per request.

The rule that matters most: **make DataLoaders per-request in middleware, never package-level globals.** A global loader caches between users — stale rows and cross-tenant leakage.

```go
type Loaders struct {
    PostsByAuthorID *PostsByAuthorIDLoader
}

func LoaderMiddleware(db *sql.DB, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := context.WithValue(r.Context(), loadersKey, &Loaders{
            PostsByAuthorID: newPostsByAuthorIDLoader(r.Context(), db),
        })
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

In gqlgen, force a dedicated resolver on batched fields via `resolver: true` in `gqlgen.yml`; wiring details live in [gqlgen reference](references/gqlgen.md).

## Authentication and authorization

Stack two layers:

1. HTTP middleware validates tokens and stashes identity in the context.
2. The schema (gqlgen directives such as `@hasRole`) or resolver checks (graph-gophers) enforce per-field authorization.

Middlewares stay transport-aware; directives keep policy next to the schema instead of scattered through resolvers.

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        user, err := validateToken(token)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), actorKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Error handling

Never let internal errors reach the client: SQL fragments, stack traces, and service details belong in logs, not in responses. Present a safe message and a machine-readable code.

```go
srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
    var gqlErr *gqlerror.Error
    if errors.As(err, &gqlErr) {
        return gqlErr
    }
    slog.Error("resolver error", "err", err)
    return gqlerror.Errorf("internal error")
})

return nil, &gqlerror.Error{
    Message: "user not found",
    Extensions: map[string]any{"code": "NOT_FOUND"},
}
```

For graph-gophers, implement `ResolverError` to attach `Extensions()`. Use `graphql.AddError(ctx, err)` in gqlgen for recoverable field failures so the handler can still return partial data.

## Subscriptions

Subscriptions ride long-lived WebSocket connections; a goroutine leaked per disconnected client silently exhausts the process. Always couple the stream to `ctx.Done()`.

```go
func (r *subscriptionResolver) MessageAdded(ctx context.Context, room string) (<-chan *model.Message, error) {
    ch := make(chan *model.Message, 1)
    sub := r.pubsub.Subscribe(room)
    go func() {
        defer close(ch)
        for {
            select {
            case <-ctx.Done():
                return
            case msg := <-sub:
                select {
                case ch <- msg:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return ch, nil
}
```

## Complexity and safety caps

Without explicit limits a few nested fields compound into a denial-of-service query. Wire caps into the handler from day one.

```go
srv := handler.NewDefaultServer(es)
srv.Use(extension.FixedComplexityLimit(200))

if !isProduction() {
    srv.Use(extension.Introspection{})
}
```

For graph-gophers, pass `graphql.MaxDepth(10)` and `graphql.MaxParallelism(10)` at schema parse time. In production also consider persisted queries so clients cannot submit arbitrary query strings.

## Common mistakes

| Mistake | Why it hurts | Fix |
| --- | --- | --- |
| Per-row queries in child resolvers | O(n) DB calls per parent set | per-request DataLoader |
| Package-global DataLoader | stale + cross-request cache | build loaders in HTTP middleware |
| Hand-editing generated files | next `go generate` erases edits | `autobind` / `model` config in `gqlgen.yml` |
| Skipping `go generate` after schema edits | stale stubs, compile breakage | regenerate in the same change |
| `int` in graph-gophers resolvers | `Int` scalar needs `int32` | use `int32` |
| Introspection on in production | full schema exposed | gate by environment |
| No complexity cap | nesting DoS the CPU | `FixedComplexityLimit` / `MaxDepth` |
| Leaking resolver errors | SQL and internals hit clients | `ErrorPresenter` / `ResolverError` |
| Goroutines without `ctx.Done()` | subscription leaks on disconnect | `defer close(ch)` + select on context |

## Deep dives

- [gqlgen reference](references/gqlgen.md) — codegen workflow, `gqlgen.yml`, DataLoaders, Federation v2, directives.
- [graphql-go reference](references/graphql-go.md) — reflection resolver model, type mapping, tracing.
- [Testing](references/testing.md) — client harnesses and `httptest` patterns.

## Cross-references

- `golang-context` — context propagation in resolvers and subscriptions.
- `golang-error-handling` — wrapping and sentinel errors behind presenters.
- `golang-testing` — table-driven and integration test structure.
- `golang-observability` — tracing and metrics per resolver.
- `golang-security` — input validation and injection prevention.
- `golang-database` — N+1 patterns and batch loading at the data layer.
