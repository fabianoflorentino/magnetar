# Testing — google/wire

Wire generates plain Go constructor calls, so tests operate on the constructor layer directly — there is no container runtime to learn, mock, or reset.

## Table of Contents

- [Unit tests: plain constructor injection](#unit-tests-plain-constructor-injection)
- [Test injectors: swapping providers](#test-injectors-swapping-providers)
- [Mocks as injector arguments](#mocks-as-injector-arguments)
- [CI: stale `wire_gen.go` detection](#ci-stale-wire_gengo-detection)
- [Testing interface bindings](#testing-interface-bindings)
- [Table-driven tests without wire](#table-driven-tests-without-wire)

## Unit tests: plain constructor injection

The generated code has no wire dependency, so build the unit under test by hand:

```go
func TestUserService_GetUser(t *testing.T) {
    mockStore := &MockUserStore{users: map[int64]*User{1: {ID: 1, Name: "Alice"}}}
    cache := newTestRedis(t)
    svc := service.NewUserService(mockStore, cache)

    u, err := svc.GetUser(context.Background(), 1)
    require.NoError(t, err)
    assert.Equal(t, "Alice", u.Name)
}
```

No wire, no container, no codegen — mocks are just constructor arguments.

## Test injectors: swapping providers

For component tests that want the full graph with selected dependencies replaced, write a test-only injector in a `_test.go` file under the `wireinject` tag.

```go
// app_test.go
//go:build wireinject

package main

import "github.com/google/wire"

// TestSet swaps real infra for in-memory fakes.
var TestSet = wire.NewSet(
    NewTestConfig,
    NewInMemoryUserStore,
    wire.Bind(new(repo.UserStore), new(*InMemoryUserStore)),
    NewTestRedis,
)

func InitTestApp(t *testing.T) (*App, func(), error) {
    wire.Build(TestSet, service.ServiceSet, NewApp)
    return nil, nil, nil
}
```

```go
// app_integration_test.go
//go:build !wireinject // compiles when the wireinject tag is NOT set

package main

func TestApp_GetUser(t *testing.T) {
    app, cleanup, err := InitTestApp(t)
    require.NoError(t, err)
    defer cleanup()

    u, err := app.GetUser(context.Background(), 1)
    require.NoError(t, err)
    assert.NotNil(t, u)
}
```

Run `wire ./...` to generate `wire_gen.go`; the test injector is included because `_test.go` files compile with the package under `go test`. Prefer a dedicated test provider set over passing mocks as parameters — the set keeps the test injector composable.

## Mocks as injector arguments

A lighter alternative: hand the mock to the injector as a parameter, which wire treats as a pre-built value.

```go
//go:build wireinject

func InitTestApp(store repo.UserStore) (*App, func(), error) {
    wire.Build(config.ConfigSet, service.ServiceSet, NewApp)
    return nil, nil, nil
}

// Test
func TestApp(t *testing.T) {
    mock := &MockUserStore{}
    app, cleanup, err := InitTestApp(mock)
    require.NoError(t, err)
    defer cleanup()
    // ...
}
```

Use this when only one or two dependencies need swapping and a full `TestSet` is overkill.

## CI: stale `wire_gen.go` detection

A provider change without regeneration builds fine but wires a wrong graph. Enforce freshness in CI:

```bash
# Option 1: regenerate and require a clean diff
wire ./...
git diff --exit-code -- '**/wire_gen.go'
```

```yaml
# .github/workflows/ci.yml
- name: Check wire_gen.go is up-to-date
  run: |
    go install github.com/google/wire/cmd/wire@v0.7.0
    wire ./...
    git diff --exit-code -- '**/wire_gen.go'
```

```bash
# Option 2: validate without touching files
wire check ./...
```

`wire check` exits non-zero on an inconsistent graph but leaves `wire_gen.go` alone — a fast validity gate for CI.

## Testing interface bindings

`wire.Bind` works inside test sets to pin a fake to the same interface:

```go
type FakeMailer struct{ sent []string }

func (f *FakeMailer) Send(to, body string) error {
    f.sent = append(f.sent, to)
    return nil
}

var TestMailerSet = wire.NewSet(
    NewFakeMailer,
    wire.Bind(new(notification.Mailer), new(*FakeMailer)),
)

var TestSet = wire.NewSet(TestMailerSet, realServiceSet) // everything else real
```

Only the mailer is faked; the rest of the graph stays real.

## Table-driven tests without wire

Once the graph is built, per-service behavior needs no wire. Build dependencies directly per case:

```go
func TestUserService(t *testing.T) {
    cases := []struct {
        name  string
        id    int64
        users map[int64]*User
        want  string
        err   bool
    }{
        {"found", 1, map[int64]*User{1: {Name: "Alice"}}, "Alice", false},
        {"not found", 99, nil, "", true},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            svc := service.NewUserService(&MockUserStore{users: tc.users}, nil)
            u, err := svc.GetUser(context.Background(), tc.id)
            if tc.err {
                require.Error(t, err)
                return
            }
            assert.Equal(t, tc.want, u.Name)
        })
    }
}
```

Wire's only job is building the graph in `main` (or an integration test); unit tests skip the injector entirely.
