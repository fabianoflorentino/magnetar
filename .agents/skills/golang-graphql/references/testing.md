# Testing GraphQL in Go

## Table of Contents

- [gqlgen — client harness](#gqlgen--client-harness)
- [gqlgen — exercising DataLoaders](#gqlgen--exercising-dataloaders)
- [gqlgen — subscription tests](#gqlgen--subscription-tests)
- [graph-gophers — gqltesting](#graph-gophers--gqltesting)
- [Verifying error extensions](#verifying-error-extensions)
- [gqlgen — auth directive tests](#gqlgen--auth-directive-tests)
- [Table-driven query tests](#table-driven-query-tests)

## gqlgen — client harness

`github.com/99designs/gqlgen/client` drives the full stack — directives, middleware, resolvers — through an `http.Handler`:

```go
func TestCreateUser(t *testing.T) {
    srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
        Resolvers: &graph.Resolver{DB: testDB},
    }))

    c := client.New(srv)

    var resp struct {
        CreateUser struct {
            User struct {
                ID    string
                Email string
            }
            Errors []struct{ Message string }
        }
    }

    c.MustPost(`
        mutation CreateUser($email: String!, $name: String!) {
            createUser(input: {email: $email, name: $name}) {
                user { id email }
                errors { message }
            }
        }
    `, &resp,
        client.Var("email", "alice@example.com"),
        client.Var("name", "Alice"),
        client.AddHeader("Authorization", "Bearer test-token"),
    )

    require.Empty(t, resp.CreateUser.Errors)
    require.Equal(t, "alice@example.com", resp.CreateUser.User.Email)
}
```

For a single resolver, call its method directly with a constructed `Resolver` and a real `context.Context` — no HTTP overhead, no serializer in the middle.

## gqlgen — exercising DataLoaders

Wrap the test server in the real DataLoader middleware so the batching path is what runs:

```go
srv := handler.NewDefaultServer(es)
h := dataloaders.Middleware(testDB, srv)

c := client.New(h)
```

## gqlgen — subscription tests

`client.Subscription` streams events from a subscription resolver:

```go
sub := c.Subscription(`subscription { messageAdded(room: "general") { content } }`)
defer sub.Close()

publishMessage("general", "hello") // trigger the event

var event struct{ MessageAdded struct{ Content string } }
err := sub.Next(&event)
require.NoError(t, err)
require.Equal(t, "hello", event.MessageAdded.Content)
```

## graph-gophers — gqltesting

```go
func TestUser(t *testing.T) {
    gqltesting.RunTests(t, []*gqltesting.Test{
        {
            Schema:         schema,
            Query:          `{ user(id: "1") { name email } }`,
            ExpectedResult: `{"user":{"name":"Alice","email":"alice@example.com"}}`,
        },
        {
            Schema:         schema,
            Query:          `{ user(id: "999") { name } }`,
            ExpectedErrors: []*gqlerrors.QueryError{
                {Message: "user not found", Extensions: map[string]any{"code": "NOT_FOUND"}},
            },
        },
    })
}
```

At the HTTP layer, drive the relay handler with `httptest`:

```go
func TestRelayHandler(t *testing.T) {
    body := `{"query":"{ user(id: \"1\") { name } }"}`
    req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    relay.Handler{Schema: schema}.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), `"Alice"`)
}
```

## Verifying error extensions

Assert the machine-readable code, not just the message:

```go
var resp struct {
    Errors []struct {
        Message    string
        Extensions struct{ Code string }
    }
}
c.Post(`{ user(id: "999") { name } }`, &resp)
require.Equal(t, "NOT_FOUND", resp.Errors[0].Extensions.Code)
```

## gqlgen — auth directive tests

Exercise the directive function directly — no query round-trip needed:

```go
func TestHasRoleDirective(t *testing.T) {
    ctx := context.WithValue(context.Background(), userKey, &domain.User{Role: "USER"})
    _, err := HasRole(ctx, nil, func(ctx context.Context) (any, error) {
        return "ok", nil
    }, model.RoleAdmin)
    require.Error(t, err)

    var gqlErr *gqlerror.Error
    require.True(t, errors.As(err, &gqlErr))
    require.Equal(t, "FORBIDDEN", gqlErr.Extensions["code"])
}
```

## Table-driven query tests

```go
func TestUserQueries(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        vars     map[string]any
        wantCode string
        wantName string
    }{
        {"existing user", `query($id:ID!){user(id:$id){name}}`, map[string]any{"id": "1"}, "", "Alice"},
        {"missing user", `query($id:ID!){user(id:$id){name}}`, map[string]any{"id": "999"}, "NOT_FOUND", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var resp struct {
                User   *struct{ Name string }
                Errors []struct{ Extensions struct{ Code string } }
            }
            c.Post(tt.query, &resp, client.Var("id", tt.vars["id"]))
            if tt.wantCode != "" {
                require.Equal(t, tt.wantCode, resp.Errors[0].Extensions.Code)
            } else {
                require.Equal(t, tt.wantName, resp.User.Name)
            }
        })
    }
}
```

For general testing habits, see the `fabianoflorentino/golang-agent-skills@golang-testing` skill.
