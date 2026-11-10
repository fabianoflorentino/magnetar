# samber/do - Generics-Based DI

`samber/do` does dependency injection with Go generics: type-safe, no reflection, no code generation, and a small API surface. It fits projects that want container convenience without a generation toolchain or a full framework. Docs: [do.samber.dev](https://do.samber.dev) and [github.com/samber/do](https://github.com/samber/do). For the complete API, patterns, and advanced features, see the `golang-samber-do` skill.

## Core pattern

```go
// Register
injector := do.New()
do.Provide(injector, func(i do.Injector) (*UserService, error) {
    db := do.MustInvoke[*Database](i)
    return NewUserService(db), nil
})

// Invoke lazily - the service is built on first use
svc := do.MustInvoke[*UserService](injector)

// Dependencies are their own constructors; container order is derived
// from constructor signatures, not hand-maintained sequencing.

// Shutdown - every registered Shutdowner is closed
injector.ShutdownOnSignalsWithContext(ctx, os.Interrupt)
```

## Why samber/do

- **No code generation** - nothing to run after edits, no generated files to keep in sync.
- **No reflection** - graph problems surface at compile time, not first boot.
- **Strongly typed** - generics carry types through `Provide`/`Invoke`, so no `interface{}` casting at consumption points.
- **Built-in lifecycle** - health checks and graceful shutdown are discovered automatically from services that implement the interfaces.
- **Container cloning** - tests clone the production container and override only the boundaries, so wiring is tested, not reinvented.
- **Package system** - domain-oriented organization without manual wiring order.
- **Minimal API** - `Provide`, `Invoke`, `Shutdown` cover the common cases.

`New(nil)` returns a fresh container; `do.New` is the root starting point, and `MustInvoke` is the lazy resolver that exercises the graph on demand.
