# gqlgen Reference

gqlgen is a schema-first code-generation library: author SDL, run `go generate`, fill in the resolver bodies. Generated output is not hand-edited.

## Table of Contents

- [Project setup](#project-setup)
- [gqlgen.yml](#gqlgenyml)
- [Resolver structure](#resolver-structure)
- [DataLoaders](#dataloaders)
- [Authentication directives](#authentication-directives)
- [Middleware hooks](#middleware-hooks)
- [Error presenter](#error-presenter)
- [Subscriptions](#subscriptions)
- [File uploads](#file-uploads)
- [Apollo Federation v2](#apollo-federation-v2)
- [Production handler setup](#production-handler-setup)

## Project setup

```bash
# Scaffold a new project
go run github.com/99designs/gqlgen init

# Pin the generator in go.mod for reproducible builds (Go 1.24+)
go get -tool github.com/99designs/gqlgen@latest

# Regenerate after every schema change
go tool gqlgen generate
```

For modules targeting Go <1.24, pin the tool through the legacy `tools.go` blank-import file instead. Never hand-edit generated files (`generated.go`, `models_gen.go`) — the next `generate` overwrites them.

## gqlgen.yml

```yaml
schema:
  - graph/schema/*.graphql

exec:
  filename: graph/generated.go
  package: graph

model:
  filename: graph/model/models_gen.go
  package: model

resolver:
  layout: follow-schema # one resolvers file per schema file
  dir: graph
  package: graph
  filename_template: "{name}.resolvers.go"

autobind:
  - github.com/me/app/internal/domain # reuse existing structs

models:
  User:
    model: github.com/me/app/internal/domain.User
    fields:
      posts:
        resolver: true # custom resolver — required for DataLoader fields

omit_slice_element_pointers: true
struct_fields_always_pointers: false
resolvers_always_return_pointers: true
```

Knobs that matter:

- `autobind` maps structs to GraphQL types; field names must match case-insensitively.
- `models.<T>.model` overrides which Go type backs a GraphQL type.
- `fields.<f>.resolver: true` forces a custom resolver instead of plain field access — any field that should batch through a DataLoader needs it.
- `struct_fields_always_pointers` / `resolvers_always_return_pointers` choose `*T` vs `T` in generated signatures; match your domain conventions.

## Resolver structure

The generated `Config` carries the `Resolvers` interface you implement. You own the resolver files; the graph wiring lives in `resolver.go`.

```go
type Resolver struct {
    db          *sql.DB
    userService *service.UserService
    loaders     *dataloaders.Loaders // injected per-request
}

type queryResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type userResolver struct{ *Resolver }

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) { ... }
func (r *userResolver) Posts(ctx context.Context, obj *model.User) ([]*model.Post, error) { ... }
```

`obj` is the parent object — the hook where graph walking starts.

## DataLoaders

Use `github.com/vikstrous/dataloadgen` (generics, fast) or `github.com/graph-gophers/dataloader`. Build loaders per request in middleware — never as globals:

```go
func Middleware(db *sql.DB, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        loaders := &Loaders{
            PostsByUserID: dataloadgen.NewLoader(func(ctx context.Context, ids []string) ([][]*domain.Post, []error) {
                return batchPostsByUserID(ctx, db, ids) // one []Post per user ID
            }, dataloadgen.WithWait(1*time.Millisecond)),
        }
        ctx := context.WithValue(r.Context(), loadersKey, loaders)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (r *userResolver) Posts(ctx context.Context, obj *model.User) ([]*model.Post, error) {
    return loaders.For(ctx).PostsByUserID.Load(ctx, obj.ID)
}
```

A `wait` of 1–2ms lets concurrent resolvers register their keys before the batch fires.

## Authentication directives

Declare the directive in SDL, implement it, and register it at bootstrap:

```graphql
directive @hasRole(role: Role!) on FIELD_DEFINITION

type Query {
  adminStats: Stats! @hasRole(role: ADMIN)
}
```

```go
func HasRole(ctx context.Context, obj any, next graphql.Resolver, role model.Role) (any, error) {
    user := auth.UserFromContext(ctx)
    if user == nil || user.Role != role {
        return nil, &gqlerror.Error{
            Message:    "access denied",
            Extensions: map[string]any{"code": "FORBIDDEN"},
        }
    }
    return next(ctx)
}

// registration
c := generated.Config{
    Resolvers: &graph.Resolver{ /* ... */ },
    Directives: generated.DirectiveRoot{HasRole: HasRole},
}
```

## Middleware hooks

Per-operation and per-field hooks keep cross-cutting logic out of resolvers:

```go
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
    // log the operation name, open a trace span
    return next(ctx)
})
srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
    // per-field timing and tracing
    return next(ctx)
})
```

## Error presenter

Never leak internals. Present a safe message and log the real error:

```go
srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
    var gqlErr *gqlerror.Error
    if errors.As(err, &gqlErr) {
        return gqlErr
    }
    log.Ctx(ctx).Error("resolver error", "err", err)
    return gqlerror.Errorf("internal server error")
})

srv.SetRecoverFunc(func(ctx context.Context, err any) error {
    log.Ctx(ctx).Error("panic in resolver", "err", err)
    return fmt.Errorf("internal server error")
})
```

For recoverable field failures, use `graphql.AddError(ctx, err)` so the handler can still return partial data.

## Subscriptions

Subscriptions live on WebSocket connections. Authenticate at connection time and constrain the origin, then `graphql-ws` (legacy) and `graphql-transport-ws` (current) subprotocols are both served:

```go
srv.AddTransport(transport.Websocket{
    KeepAlivePingInterval: 10 * time.Second,
    Upgrader: websocket.Upgrader{
        // Restrict to your own origin; `true` for all is dev-only.
        CheckOrigin: func(r *http.Request) bool {
            return r.Header.Get("Origin") == "https://app.example.com"
        },
    },
    InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
        token := initPayload.Authorization()
        user, err := validateToken(token)
        if err != nil {
            return ctx, nil, err
        }
        return context.WithValue(ctx, userKey, user), &initPayload, nil
    },
})
```

Always couple the event stream to `ctx.Done()` so a disconnected client cannot leak a goroutine.

## File uploads

```go
srv.AddTransport(transport.MultipartForm{
    MaxUploadSize: 10 << 20, // 10 MB total
    MaxMemory:     5 << 20,  // 5 MB in memory; the rest spills to disk
})
```

```graphql
scalar Upload

type Mutation {
  uploadAvatar(file: Upload!): User!
}
```

The resolver receives `graphql.Upload{File io.Reader, Filename string, Size int64, ContentType string}`.

## Apollo Federation v2

Enable the federation plugin and use the standard `@key`/`@shareable`/`@external` link:

```yaml
federation:
  filename: graph/federation.go
  version: 2
```

```graphql
extend schema
  @link(
    url: "https://specs.apollo.dev/federation/v2.3"
    import: ["@key", "@shareable", "@external"]
  )

type User @key(fields: "id") {
  id: ID!
  name: String!
}
```

Implement `FindUserByID` on the generated entity resolver; the schema composes under Apollo Router and Cosmo.

## Production handler setup

Assemble the transport stack and the safety caps up front:

```go
srv := handler.New(es)
srv.AddTransport(transport.Options{})
srv.AddTransport(transport.GET{})
srv.AddTransport(transport.POST{})
srv.AddTransport(transport.MultipartForm{MaxUploadSize: 10 << 20, MaxMemory: 5 << 20})
srv.AddTransport(transport.Websocket{KeepAlivePingInterval: 10 * time.Second})

srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
if os.Getenv("ENV") != "production" {
    srv.Use(extension.Introspection{})
}
srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})
srv.Use(extension.FixedComplexityLimit(200))
```

Introspection stays off outside development; complexity and persisted-query caps constrain the query surface in production.
