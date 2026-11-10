# Advanced — uber-go/dig

Detail topics referenced from `SKILL.md`. Each section is self-contained.

## Table of Contents

- [Decorate](#decorate)
- [Scopes](#scopes)
- [Optional dependencies](#optional-dependencies)
- [Error handling](#error-handling)
- [Visualization](#visualization)
- [Quick reference](#quick-reference)
  - [Container](#container)
  - [Provide options](#provide-options)
  - [Container options](#container-options)
  - [Errors](#errors)

## Decorate

`Decorate` rewrites a value already in the container: the decorator receives the existing instance and returns its replacement. Typical uses: enriching a logger with context, tagging a metrics scope, swapping a real client for a recording one inside a test scope.

```go
c.Decorate(func(log *zap.Logger) *zap.Logger {
    return log.Named("worker")
})
```

A decorator reaches only the scope where it was registered and that scope's descendants. Apply it at scope or package wiring boundaries, not in `main`, so the change stays local.

## Scopes

A `Scope` is a child container: it inherits providers from its parent and can add or override its own. That is how request-, tenant-, or module-level dependencies coexist with shared singletons:

```go
root := dig.New()
root.Provide(NewLogger)
root.Provide(NewDatabase)

requestScope := root.Scope("request")
requestScope.Provide(NewRequestContext) // only visible inside requestScope
requestScope.Decorate(func(l *zap.Logger) *zap.Logger {
    return l.With(zap.String("scope", "request"))
})
```

Providers registered to a scope are private to it and its children by default. Pass `dig.Export(true)` to expose one to the parent:

```go
requestScope.Provide(NewSharedCache, dig.Export(true))
```

## Optional dependencies

`optional:"true"` lets a consumer resolve when no provider exists. Use it sparingly — optional deps hide configuration mistakes. Justified for genuinely optional concerns (a tracing exporter, an in-memory cache), wrong for core services like a database.

```go
type Params struct {
    dig.In

    Logger *zap.Logger
    Tracer trace.Tracer `optional:"true"`
}
```

## Error handling

dig wraps a constructor error with the dependency path, so the failure names the graph edge:

```go
if err := c.Invoke(run); err != nil {
    // err chains back through the graph: "could not build *http.Server: ..."
}
```

Useful helpers:

- `errors.As(err, &dig.Error{})` — true when the error originated inside dig.
- `dig.RootCause(err)` — unwrap to the original constructor error from user code.
- `dig.IsCycleDetected(err)` — true when the graph forms a cycle (surfaced at first `Invoke` unless `DeferAcyclicVerification` is set).
- `errors.As(err, &dig.PanicError{})` — a constructor panic surfaces as this typed error when `RecoverFromPanics` is enabled.

## Visualization

dig emits the graph in DOT format — the way to reason about wiring once it outgrows reading code:

```go
f, _ := os.Create("graph.dot")
_ = dig.Visualize(c, f)
// then: dot -Tpng graph.dot -o graph.png
```

`dig.VisualizeError(err)` marks the failed edges of a failed `Invoke` — the fastest path to a "missing type" fix in a deep graph.

## Quick reference

### Container

| Function / method | Purpose |
| --- | --- |
| `dig.New(opts...)` | create a root container |
| `c.Provide(ctor, opts...)` | register a constructor |
| `c.Invoke(fn, opts...)` | run a function with injected dependencies |
| `c.Decorate(fn, opts...)` | modify a provided value within a scope |
| `c.Scope(name, opts...)` | fork a child scope (private providers by default) |
| `c.String()` | human-readable provider summary (for DOT use `dig.Visualize`) |

### Provide options

| Option | Purpose |
| --- | --- |
| `dig.Name("...")` | disambiguate same-typed providers |
| `dig.Group("...")` | add the result to a value group |
| `dig.As(new(I))` | serve the concrete value as one or more interfaces |
| `dig.Export(true)` | make a scope provider visible from the parent |
| `dig.FillProvideInfo(&info)` | capture provider metadata for tooling |

### Container options

| Option | Purpose |
| --- | --- |
| `dig.DeferAcyclicVerification()` | defer the cycle check to the first `Invoke` |
| `dig.RecoverFromPanics()` | turn constructor panics into `dig.PanicError` |
| `dig.DryRun(true)` | validate the graph without running constructors |

### Errors

| Helper | Purpose |
| --- | --- |
| `dig.RootCause(err)` | unwrap to the user-returned error |
| `dig.IsCycleDetected(err)` | true when the graph has a cycle |
| `errors.As(err, &dig.PanicError{})` | detect a recovered constructor panic |
| `dig.Visualize(c, w, opts...)` | write the graph in DOT format |
