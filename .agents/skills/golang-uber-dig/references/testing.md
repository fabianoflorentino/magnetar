# Testing with uber-go/dig

Containers are cheap: build a fresh one per test, swap what needs swapping, and drive the system through `Invoke`.

## Table of Contents

- [Per-test container](#per-test-container)
- [Shared test wiring](#shared-test-wiring)
- [Validating the production graph in CI](#validating-the-production-graph-in-ci)
- [Catching cycles before deploy](#catching-cycles-before-deploy)
- [Asserting a constructor's error path](#asserting-a-constructors-error-path)
- [Recovering from constructor panics](#recovering-from-constructor-panics)

## Per-test container

```go
func TestUserService_Create(t *testing.T) {
    c := dig.New()

    fakeDB := &fakeDatabase{}
    require.NoError(t, c.Provide(func() Database { return fakeDB }))
    require.NoError(t, c.Provide(NewUserService))

    require.NoError(t, c.Invoke(func(s *UserService) {
        err := s.Create(context.Background(), "alice@example.com")
        require.NoError(t, err)
    }))

    require.Len(t, fakeDB.inserted, 1)
}
```

The fake becomes the last assertion: behavior is checked after the call through `Invoke`, not just at the call site.

## Shared test wiring

Factor repeated providers into a helper that accepts overrides:

```go
func newTestContainer(t *testing.T, overrides ...func(*dig.Container)) *dig.Container {
    t.Helper()
    c := dig.New()
    require.NoError(t, c.Provide(NewTestLogger))
    require.NoError(t, c.Provide(NewInMemoryCache))
    require.NoError(t, c.Provide(func() Database { return &fakeDatabase{} }))
    require.NoError(t, c.Provide(NewUserService))

    for _, override := range overrides {
        override(c)
    }
    return c
}

func TestUserService_NotFound(t *testing.T) {
    c := newTestContainer(t, func(c *dig.Container) {
        // Swap the default DB for one returning sql.ErrNoRows.
        require.NoError(t, c.Decorate(func(db Database) Database {
            return &notFoundDB{Database: db}
        }))
    })

    require.NoError(t, c.Invoke(func(s *UserService) {
        _, err := s.Get(context.Background(), "missing")
        require.ErrorIs(t, err, ErrUserNotFound)
    }))
}
```

`Decorate` is the cleanest test seam — the test reads like production wiring plus one line.

## Validating the production graph in CI

Mirror the composition root and validate it structurally:

```go
func TestProductionGraph(t *testing.T) {
    c := dig.New(dig.DryRun(true))

    require.NoError(t, registerAll(c)) // every Provide() from main()

    require.NoError(t, c.Invoke(func(*http.Server, *Worker, *MetricsExporter) {}))
}
```

`DryRun(true)` skips constructor execution, so missing providers and type mismatches surface without opening database connections.

## Catching cycles before deploy

```go
func TestNoCycles(t *testing.T) {
    c := dig.New()
    require.NoError(t, registerAll(c))

    err := c.Invoke(func(*App) {})
    require.False(t, dig.IsCycleDetected(err), "cycle in dependency graph: %v", err)
}
```

Cycle detection runs at `Invoke` by default (or at `Provide` time if deferred verification is disabled).

## Asserting a constructor's error path

dig wraps the returned error with the dependency path — unwrap with `RootCause` to reach the original text:

```go
func TestDBProvider_BadDSN(t *testing.T) {
    c := dig.New()
    require.NoError(t, c.Provide(func() *Config {
        return &Config{DSN: "not a dsn"}
    }))
    require.NoError(t, c.Provide(NewDB))

    err := c.Invoke(func(*sql.DB) {})
    require.Error(t, err)
    require.ErrorContains(t, dig.RootCause(err), "invalid connection string")
}
```

## Recovering from constructor panics

Wrap the whole graph so a panicking constructor reports as a typed error instead of crashing the test runner:

```go
c := dig.New(dig.RecoverFromPanics())

require.NoError(t, c.Provide(func() *App {
    panic("intentionally broken")
}))

err := c.Invoke(func(*App) {})

var pe dig.PanicError
require.True(t, errors.As(err, &pe))
require.Contains(t, pe.Error(), "intentionally broken")
```
