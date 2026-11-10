# Testing with samber/do

Tests stand in for `main` as the composition root. The container API is shaped so the graph under test mirrors production wiring with a few overrides instead of a parallel construction path.

## Clone the production injector

`injector.Clone()` copies the container — including already-resolved services — so a test starts from the real graph rather than rebuilding it:

```go
func TestUserService(t *testing.T) {
    testInjector := mainInjector.Clone()

    do.OverrideValue(testInjector, &MockDatabase{})
    service := do.MustInvoke[UserService](testInjector)
    // exercise service
}
```

`CloneWithOpts()` clones with custom container options for the rare case the copy needs different settings.

## Overriding providers

Every registration kind has an `Override` counterpart, and they are intended for tests:

- `do.Override[T]` — replace a provider; the replacement runs on next resolve.
- `do.OverrideValue[T]` — swap in a pre-built value; the most common test move.
- `do.OverrideNamed[T]` / `do.OverrideNamedValue[T]` — overrides for named services.
- `do.OverrideTransient[T]` / `do.OverrideNamedTransient[T]` — replace transient factories.

Reuse the same `Provider` functions the service uses so the test graph keeps production shape; only the leaf fake changes.

## Shared test helper

A small helper builds a container of fakes so every test starts from one wiring point:

```go
func SetupTestContainer(t *testing.T) do.Injector {
    injector := do.New()
    do.Provide(injector, func(i do.Injector) (Database, error) {
        return &MockDatabase{}, nil
    })
    return injector
}
```

Tests that need a specific fake call `do.OverrideValue` after cloning — the override replaces just that edge and nothing else.