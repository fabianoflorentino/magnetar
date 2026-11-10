# uber-go/dig + uber-go/fx - Reflection-Based DI

`dig` is the low-level reflection container; `fx` is the application framework built on top of it. Together they bring containers, lifecycle hooks, and modular organization to large Go services. The trade for that power is that resolution happens by reflection at runtime, so miswired graphs fail at startup rather than at `go build`. Docs: [github.com/uber-go/dig](https://github.com/uber-go/dig) and [fx](https://uber-go.github.io/fx/).

## dig - the container

Provide constructors, then invoke anything - dig resolves the chain:

```go
container := dig.New()

_ = container.Provide(NewConfig)
_ = container.Provide(NewDatabase)
_ = container.Provide(NewUserStore)
_ = container.Provide(NewUserService)

err := container.Invoke(func(svc *UserService) {
    svc.Run()
})
```

### Named dependencies

One type, several roles - differentiate by name:

```go
type DatabaseParams struct {
    dig.In

    Primary *sql.DB `name:"primary"`
    Replica *sql.DB `name:"replica"`
}

_ = container.Provide(NewPrimaryDB, dig.Name("primary"))
_ = container.Provide(NewReplicaDB, dig.Name("replica"))

_ = container.Provide(func(p DatabaseParams) *UserService {
    return &UserService{writer: p.Primary, reader: p.Replica}
})
```

### dig tradeoffs

- Reflection means type mismatches are runtime errors, not compile errors.
- `dig.In`/`dig.Out` structs add boilerplate on large graphs.
- No lifecycle or app orchestration - that is fx's job.
- `dig.Group` collects multiple implementations into one injectable slice.

## fx - the application framework

### Basic application

fx runs the wiring, blocks until a signal, and routes teardown through hooks:

```go
app := fx.New(
    fx.Provide(NewConfig, NewDatabase, NewUserStore, NewUserService),
    fx.Invoke(RegisterRoutes),
    fx.Invoke(StartServer),
)
app.Run() // starts, blocks on signal, runs stop hooks
```

### Lifecycle hooks

Give constructors `fx.Lifecycle` and register start/stop behavior:

```go
func NewDatabase(lc fx.Lifecycle, cfg *Config) (*Database, error) {
    db := &Database{}
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return db.Connect(cfg.URL)
        },
        OnStop: func(ctx context.Context) error {
            return db.Close()
        },
    })
    return db, nil
}
```

### Modules

Group providers by domain:

```go
var InfraModule = fx.Module("infra",
    fx.Provide(NewConfig),
    fx.Provide(NewDatabase),
    fx.Provide(NewCache),
)

var ServiceModule = fx.Module("service",
    fx.Provide(NewUserService),
    fx.Provide(NewOrderService),
)

app := fx.New(InfraModule, ServiceModule, fx.Invoke(StartServer))
```

### Testing with fxtest

`fxtest` starts a real (mini) app in the test process:

```go
func TestUserService(t *testing.T) {
    var svc *UserService

    app := fxtest.New(t,
        fx.Provide(NewMockUserStore),
        fx.Provide(NewUserService),
        fx.Populate(&svc),
    )
    app.RequireStart()
    defer app.RequireStop()

    // exercise svc
}
```

### fx tradeoffs

- Full startup/shutdown/signal orchestration out of the box.
- Reflection-based - wiring mistakes surface at startup.
- Noticeable learning surface: `fx.In`, `fx.Out`, `fx.Annotate`, `fx.Decorate`, options.
- `OnStart`/`OnStop` hooks are the endorsed lifecycle seam.
- Modules group providers, keeping large apps navigable.

Register start/stop via `fx.Lifecycle` hooks; group related providers with `fx.Module`; prefer `fxtest` over hand-rolling test containers.
