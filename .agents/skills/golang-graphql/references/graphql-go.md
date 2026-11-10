# graph-gophers/graphql-go Reference

Schema-first and reflection-based: write SDL, bind Go resolver structs, and get parse-time validation — no codegen step. The contract is checked when the schema loads, so mismatches surface at process start.

## Table of Contents

- [Setup](#setup)
- [Resolver structure](#resolver-structure)
- [Type mapping](#type-mapping)
- [Nullable vs non-null arguments](#nullable-vs-non-null-arguments)
- [Custom scalars](#custom-scalars)
- [Interfaces and unions](#interfaces-and-unions)
- [DataLoaders](#dataloaders)
- [Error handling](#error-handling)
- [OpenTelemetry tracing](#opentelemetry-tracing)
- [Subscriptions](#subscriptions)
- [Disabling introspection](#disabling-introspection)
- [Testing](#testing)
- [graph-gophers vs gqlgen](#graph-gophers-vs-gqlgen)

## Setup

```go
import (
    "github.com/graph-gophers/graphql-go"
    "github.com/graph-gophers/graphql-go/relay"
    "github.com/graph-gophers/graphql-go/trace/otel"
)

schema := graphql.MustParseSchema(sdlString, &RootResolver{},
    graphql.MaxDepth(10),
    graphql.MaxParallelism(10),
    graphql.UseFieldResolvers(), // expose exported struct fields without explicit methods
    graphql.Tracer(otel.DefaultTracer()),
)

http.Handle("/graphql", &relay.Handler{Schema: schema})
```

`MustParseSchema` panics on invalid SDL or a resolver mismatch — fail fast at startup, not per request. Use `ParseSchema` when you want the error handled instead.

## Resolver structure

One exported method per schema field; names match case-insensitively:

```go
type RootResolver struct{ db *sql.DB }

type QueryResolver struct{ db *sql.DB }

func (r *RootResolver) Query() *QueryResolver { return &QueryResolver{db: r.db} }

// A struct groups the field's arguments.
func (r *QueryResolver) User(ctx context.Context, args struct{ ID graphql.ID }) (*UserResolver, error) {
    user, err := r.db.GetUser(ctx, string(args.ID))
    if err != nil {
        return nil, err
    }
    return &UserResolver{user: user}, nil
}
```

Return resolver wrapper structs rather than domain models — it keeps the GraphQL projection separate from persistence.

## Type mapping

| GraphQL | Go | Notes |
| --- | --- | --- |
| `ID` | `graphql.ID` | string alias |
| `Int` | `int32` | **NOT `int`** — a mismatch fails parsing |
| `Float` | `float64` | |
| `String` | `string` | |
| `Boolean` | `bool` | |
| `[T]` | `[]*T` or `[]T` | |
| Nullable `T` | `*T` | pointer expresses nullability |
| Non-null `T!` | `T` | plain value |
| Custom scalar | `UnmarshalGraphQL(input any) error` + `MarshalJSON() ([]byte, error)` | |
| Enum | typed string alias | |
| Input | exported struct | tags optional |
| Interface / union | Go interface; `ToConcreteType() (*T, bool)` discriminators | |

The recurring mistake: `int` for an `Int!` field is rejected at parse time.

## Nullable vs non-null arguments

```go
// ✓ Good — pointer arg = nullable in SDL
func (r *QueryResolver) Users(ctx context.Context, args struct {
    Role  *string // nullable: Role in SDL
    Limit int32   // non-null: Limit! in SDL
}) ([]*UserResolver, error) { ... }
```

Forgetting the `*` on a nullable argument makes unmarshalling fail when a client sends `null`.

## Custom scalars

```go
type DateTime struct{ time.Time }

func (d *DateTime) UnmarshalGraphQL(input any) error {
    s, ok := input.(string)
    if !ok {
        return fmt.Errorf("DateTime must be a string")
    }
    t, err := time.Parse(time.RFC3339, s)
    if err != nil {
        return err
    }
    d.Time = t
    return nil
}

func (d DateTime) MarshalJSON() ([]byte, error) {
    return json.Marshal(d.Time.Format(time.RFC3339))
}
```

## Interfaces and unions

```graphql
interface Node {
  id: ID!
}
union SearchResult = User | Post
```

```go
// One discriminator per concrete variant.
type SearchResultResolver struct{ result any }

func (r *SearchResultResolver) ToUser() (*UserResolver, bool) {
    u, ok := r.result.(*domain.User)
    return &UserResolver{u}, ok
}

func (r *SearchResultResolver) ToPost() (*PostResolver, bool) {
    p, ok := r.result.(*domain.Post)
    return &PostResolver{p}, ok
}
```

## DataLoaders

`github.com/graph-gophers/dataloader` drives batched loads per request:

```go
func DataLoaderMiddleware(db *sql.DB, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        loader := dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
            ids := make([]string, len(keys))
            for i, k := range keys {
                ids[i] = k.String()
            }
            posts, err := batchPostsByUserID(ctx, db, ids)
            // map results back to key order ...
            return results
        })
        ctx := context.WithValue(r.Context(), postsLoaderKey, loader)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (r *UserResolver) Posts(ctx context.Context) ([]*PostResolver, error) {
    thunk := ctx.Value(postsLoaderKey).(*dataloader.Loader).Load(ctx, dataloader.StringKey(r.user.ID))
    result, err := thunk()
    // ...
}
```

Build the loader per request in middleware — a package-level loader caches across users.

## Error handling

Implement `ResolverError` to attach machine-readable extensions:

```go
type ResolverError interface {
    error
    Extensions() map[string]any
}

type AppError struct{ msg, code string }

func (e *AppError) Error() string      { return e.msg }
func (e *AppError) Extensions() map[string]any {
    return map[string]any{"code": e.code}
}

// In a resolver
return nil, &AppError{msg: "user not found", code: "NOT_FOUND"}
```

Resolver panics are caught automatically and surfaced as GraphQL errors.

## OpenTelemetry tracing

```go
schema := graphql.MustParseSchema(sdl, &RootResolver{},
    graphql.Tracer(otel.DefaultTracer()),
)
```

Emits a span per request, validation, and field resolution, tagged with the operation name and field path.

## Subscriptions

```go
func (r *SubscriptionResolver) MessageAdded(ctx context.Context, args struct{ Room string }) <-chan *MessageResolver {
    ch := make(chan *MessageResolver, 1)
    go func() {
        defer close(ch)
        sub := r.pubsub.Subscribe(args.Room)
        defer sub.Unsubscribe()
        for {
            select {
            case <-ctx.Done():
                return
            case msg := <-sub.Chan():
                select {
                case ch <- &MessageResolver{msg: msg}:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return ch
}
```

A WebSocket transport is not bundled — pair with `gorilla/websocket` or mount the relay handler on a WebSocket-aware mux.

## Disabling introspection

```go
schema := graphql.MustParseSchema(sdl, &RootResolver{},
    graphql.DisableIntrospection(),
)
```

## Testing

`gqltesting.RunTests` drives the schema in-process:

```go
func TestUser(t *testing.T) {
    gqltesting.RunTests(t, []*gqltesting.Test{
        {
            Schema:         schema,
            Query:          `{ user(id: "1") { name email } }`,
            ExpectedResult: `{ "user": { "name": "Alice", "email": "alice@example.com" } }`,
        },
    })
}
```

For HTTP-level coverage, exercise `relay.Handler` with `httptest.NewRecorder()`.

## graph-gophers vs gqlgen

| Concern | graph-gophers | gqlgen |
| --- | --- | --- |
| Type safety | parse-time reflection | compile-time codegen |
| Build complexity | none | `go generate` step |
| Performance | slower (reflection) | faster (static dispatch) |
| Federation | manual | first-class (v2) |
| File uploads | manual | built-in MultipartForm |
| Fits | small/medium schemas | large schemas, strict teams |
