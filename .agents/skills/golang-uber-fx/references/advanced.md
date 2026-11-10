# Advanced — uber-go/fx

Detail topics referenced from `SKILL.md`. Each section is self-contained.

## Table of Contents

- [Supply, Replace, Decorate](#supply-replace-decorate)
- [Optional dependencies](#optional-dependencies)
- [Logging fx events](#logging-fx-events)
- [Manual lifecycle control](#manual-lifecycle-control)
- [Quick reference](#quick-reference)
  - [Application](#application)
  - [Wiring](#wiring)
  - [Annotations](#annotations)
  - [Lifecycle](#lifecycle)
  - [Logging and testing](#logging-and-testing)

## Supply, Replace, Decorate

| Option | Purpose |
| --- | --- |
| `fx.Supply(values...)` | register pre-built values — config, secrets, parsed flags |
| `fx.Replace(values...)` | swap an already-provided type; the standard test seam |
| `fx.Decorate(fn)` | wrap or modify an existing value; scoped to the surrounding module |

```go
fx.Supply(cfg, secret)

// Replace inside fxtest
fx.Replace(fx.Annotate(&fakeDB{}, fx.As(new(Database))))

// Decorate, confined to a module
fx.Module("worker",
    fx.Decorate(func(s metrics.Scope) metrics.Scope {
        return s.Tagged(map[string]string{"component": "worker"})
    }),
)
```

## Optional dependencies

`optional:"true"` lets a consumer resolve when no provider is registered. Reserve it for genuinely optional infrastructure (a tracer, a cache) — a core service hidden behind `optional` becomes a nil-pointer panic at first use.

```go
type Params struct {
    fx.In

    Logger *zap.Logger
    Tracer trace.Tracer `optional:"true"`
}
```

## Logging fx events

fx reports provide/invoke/hook events through an `fxevent.Logger`. The default writes to stderr; route it into a real logger, or silence it in tests:

```go
fx.New(
    fx.Provide(NewZapLogger),
    fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
        return &fxevent.ZapLogger{Logger: log}
    }),
    // Or silence everything: fx.NopLogger
)
```

## Manual lifecycle control

`app.Run()` is one-shot. For tests, custom signal handling, or embedding fx inside a larger program, drive Start and Stop explicitly:

```go
app := fx.New(/* ... */)

startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := app.Start(startCtx); err != nil {
    log.Fatal(err)
}

<-app.Done() // closes on SIGINT/SIGTERM

stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := app.Stop(stopCtx); err != nil {
    log.Fatal(err)
}
```

`fx.StartTimeout` and `fx.StopTimeout` set the defaults; an explicit context overrides them per call.

## Quick reference

### Application

| Function | Purpose |
| --- | --- |
| `fx.New(opts...)` | build the application graph |
| `app.Run()` | start, wait for signal, stop — one call |
| `app.Start(ctx)` | run OnStart hooks in dependency order |
| `app.Stop(ctx)` | run OnStop hooks in reverse order |
| `app.Done()` | channel closed on SIGINT/SIGTERM |
| `app.Err()` | wiring error from `fx.New` — validate without starting |

### Wiring

| Option | Purpose |
| --- | --- |
| `fx.Provide(ctors...)` | register constructors |
| `fx.Invoke(fns...)` | run functions during Start |
| `fx.Supply(values...)` | register pre-built values |
| `fx.Replace(values...)` | overrides a previously provided type (tests) |
| `fx.Decorate(fn)` | wrap an existing value (module-scoped) |
| `fx.Module(name, opts...)` | group providers, invokes, and decorators |
| `fx.Options(opts...)` | bundle options into a single value |
| `fx.Populate(targets...)` | extract typed values from the graph (tests) |

### Annotations

| Function | Purpose |
| --- | --- |
| `fx.Annotate(fn, opts...)` | tag or interface-wrap a constructor |
| `fx.ParamTags("...")` | tag parameters of an annotated constructor |
| `fx.ResultTags("...")` | tag results of an annotated constructor |
| `fx.As(new(I))` | provide as one or more interfaces |
| `fx.From(types...)` | bind annotated parameters to specific provided types |

### Lifecycle

| Helper | Purpose |
| --- | --- |
| `fx.Hook{OnStart, OnStop}` | full hook with context-aware callbacks |
| `fx.StartHook(fn)` | adapt a simple start function |
| `fx.StopHook(fn)` | adapt a simple stop function |
| `fx.StartStopHook(start, stop)` | pair of simple start/stop functions |
| `fx.StartTimeout(d)` / `fx.StopTimeout(d)` | override the 15s lifecycle defaults |
| `fx.ErrorHook(h)` | intercept lifecycle errors (failed OnStart) for alerting |

### Logging and testing

| Helper | Purpose |
| --- | --- |
| `fx.WithLogger(fn)` | plug in a custom `fxevent.Logger` |
| `fx.NopLogger` | silence fx event logging |
| `fxevent.ZapLogger{Logger: log}` | bridge fx events into zap |
| `fxevent.SlogLogger{Logger: log}` | bridge fx events into log/slog |
| `fxtest.New(t, opts...)` | app that fails the test on errors |
| `app.RequireStart()` / `app.RequireStop()` | start/stop with `t.Fatal` on failure |
| `fxtest.NewLifecycle(t)` | standalone lifecycle for unit tests |
