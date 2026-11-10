# Testing with uber-go/fx

`go.uber.org/fx/fxtest` adapts fx to `*testing.T`: failures fail the test instead of crashing the runner, and teardown is registered automatically.

## Table of Contents

- [Extracting a value with `fx.Populate`](#extracting-a-value-with-fxpopulate)
- [Swapping a dependency with `fx.Replace`](#swapping-a-dependency-with-fxreplace)
- [Standalone lifecycle for a unit test](#standalone-lifecycle-for-a-unit-test)
- [Asserting wire-time errors](#asserting-wire-time-errors)
- [Validating the production graph in CI](#validating-the-production-graph-in-ci)
- [Capturing fx events for assertions](#capturing-fx-events-for-assertions)
- [Testing a lifecycle hook in isolation](#testing-a-lifecycle-hook-in-isolation)

## Extracting a value with `fx.Populate`

```go
func TestUserService_Create(t *testing.T) {
    var svc *UserService

    app := fxtest.New(t,
        fx.Provide(
            func() Database { return &fakeDatabase{} },
            NewUserService,
        ),
        fx.Populate(&svc),
    )
    defer app.RequireStop()
    app.RequireStart()

    require.NoError(t, svc.Create(context.Background(), "alice@example.com"))
}
```

`fx.Populate(&svc)` fills `svc` with the resolved value, replacing the old `fx.Invoke(func(s *UserService) { svc = s })` pattern.

## Swapping a dependency with `fx.Replace`

`fx.Replace` overrides a type even when the original provider is buried inside a module:

```go
func TestServer_HandlesDBError(t *testing.T) {
    var srv *http.Server
    fakeDB := &erroringDatabase{}

    app := fxtest.New(t,
        ProductionModule, // the real wiring
        fx.Replace(fx.Annotate(fakeDB, fx.As(new(Database)))),
        fx.Populate(&srv),
    )
    defer app.RequireStop()
    app.RequireStart()

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/users", nil)
    srv.Handler.ServeHTTP(rec, req)
    require.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

No module rewrite needed — the resolved type is overridden graph-wide.

## Standalone lifecycle for a unit test

`fxtest.NewLifecycle(t)` hands you an `fx.Lifecycle` without the full `fx.New` machinery. The lightest way to test a constructor that registers hooks:

```go
func TestWorker_StartStop(t *testing.T) {
    lc := fxtest.NewLifecycle(t)

    worker := NewWorker(lc, zaptest.NewLogger(t))
    require.NotNil(t, worker)

    lc.RequireStart() // runs OnStart hooks
    require.True(t, worker.IsRunning())

    lc.RequireStop() // runs OnStop hooks
    require.False(t, worker.IsRunning())
}
```

## Asserting wire-time errors

When you expect wiring to fail, use `fx.New` — `fxtest.New` would call `t.Fatal`:

```go
func TestWiring_MissingDependency(t *testing.T) {
    app := fx.New(
        fx.Provide(NewServer), // depends on *sql.DB, which is not provided
        fx.NopLogger,
    )
    require.Error(t, app.Err())
    require.Contains(t, app.Err().Error(), "missing type: *sql.DB")
}
```

## Validating the production graph in CI

```go
func TestProductionGraph(t *testing.T) {
    app := fx.New(
        ProductionOptions(), // every fx.Provide / fx.Module the binary uses
        fx.NopLogger,
    )
    require.NoError(t, app.Err())
}
```

`fx.New` validates the type graph without starting. Missing providers, cycles, and annotation mismatches fail the build before deploy.

## Capturing fx events for assertions

Route events into a zap observer to assert lifecycle behavior:

```go
// go.uber.org/zap/zaptest/observer
core, recorded := observer.New(zap.InfoLevel)
log := zap.New(core)

app := fxtest.New(t,
    fx.WithLogger(func() fxevent.Logger {
        return &fxevent.ZapLogger{Logger: log}
    }),
    fx.Provide(NewWorker),
    fx.Invoke(func(*Worker) {}),
)
defer app.RequireStop()
app.RequireStart()

require.NotEmpty(t, recorded.FilterMessage("OnStart hook executed").All())
```

## Testing a lifecycle hook in isolation

A constructor that returns a value and registers a hook deserves coverage of both halves — pre-bind the port so the bind error is deterministic:

```go
func TestNewServer_OnStartFailsBindError(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)
    defer listener.Close()
    addr := listener.Addr().String()

    cfg := &Config{Addr: addr}
    lc := fxtest.NewLifecycle(t)

    NewHTTPServer(lc, zaptest.NewLogger(t), cfg)

    // Use Start, not RequireStart, so the error can be asserted.
    require.Error(t, lc.Start(context.Background()))
}
```
