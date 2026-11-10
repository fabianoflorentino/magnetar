# Structure: Cycles, Boundaries, Type Moves, API Evolution

Go enforces a few structural rules at compile time that other languages relegate to convention or linting. This file covers the load-bearing ones: why import cycles are a hard error, how to design a boundary that will not need redesigning, how to move a type across packages without breaking every caller at once, and how to evolve an exported API without a flag day.

## Breaking import cycles

Go compiles package-by-package in dependency order: to type-check `X`, the compiler needs the compiled type information of everything `X` imports. An import cycle — `X` imports `Y`, `Y` imports `X` directly or transitively — has no valid compilation order, so `go build` rejects it outright as `import cycle not allowed`. There is no fallback behavior to fall back to.

Four strategies fix a cycle, in preference order:

| # | Strategy | Call-site cost | Best when |
| --- | --- | --- | --- |
| 1 | Consumer-side interface | None — no call-site changes anywhere | The consumer calls one or two methods on the producer's type |
| 2 | Extract shared type to a leaf package | Import-path changes on both sides | Both packages genuinely need the same concrete type, not just its behavior |
| 3 | `internal/` package | Import-path changes for the shared code only | The shared code must never become public API |
| 4 | Mediator/bridge package | New package, both sides delegate to it | 1-3 do not fit the shape of the coupling |

### 1. Consumer-side interface

The idiomatic first move, and the cheapest: it requires zero changes at any call site. If package `x` only *uses* behavior from `y` — it calls a method or two on `y`'s concrete type, but does not need the type itself — define a small interface in `x` naming just those methods. Go's implicit interface satisfaction means `y`'s existing type already satisfies it without `y` importing anything from `x` or even knowing the interface exists:

```go
// package x (consumer)
type Storer interface {
    Store(ctx context.Context, key string, val []byte) error
}

func Process(s Storer) error { /* uses s.Store; no import of package y */ }
```

```go
// package y — unchanged; already satisfies x.Storer implicitly
type Store struct{ /* ... */ }

func (s *Store) Store(ctx context.Context, key string, val []byte) error { /* ... */ }
```

`x` stops importing `y` entirely, and one direction of the dependency graph turns out to have never been real — `x` needed a name for behavior, not `y`'s concrete type.

### 2. Extract shared types to a leaf package

Go's package model is flat within a module — a nested subdirectory is still a fully distinct, independently importable package. Pulling the types both `x` and `y` need into a small new leaf package breaks the cycle by construction, because the leaf imports neither. Be honest about the cost: a single cycle can span five or more packages once you trace every shared type, so this is the "correct but sometimes painful" option next to the surgical consumer-side interface.

### 3. `internal/` packages

Anything rooted under an `internal/` directory is importable only by packages rooted at the parent of that directory. That lets sibling packages share implementation detail through a common `internal/` package without either becoming part of the module's public surface. For example, both `order/billing` and `order/shipping` may import `order/internal/model`, but nothing outside the `order` tree may:

```
order/
├── billing/
│   └── billing.go        // imports order/internal/model
├── shipping/
│   └── shipping.go       // imports order/internal/model
└── internal/
    └── model/
        └── order.go      // shared type, invisible outside order/
```

### 4. Mediator/bridge package

A last resort: a new package holding the shared functionality both `x` and `y` delegate to, absorbing the coupling neither side wants to own. Reach for it only after confirming a consumer-side interface cannot express the relationship — it usually can.

## Package boundary design

**Accept interfaces, return structs.** Accepting a narrow interface lets any caller pass anything satisfying it — including a test fake implementing only the one or two methods the function calls — without the function importing the caller's concrete types. Returning a concrete struct preserves full type information at the call site, which matters because *that* caller's consumers get to define their own narrow interface later, on their side.

**Define interfaces where they are consumed, not where they are implemented.** This is the rule that actually prevents cycles rather than just describing good taste. The consumer declares an interface naming only the methods it calls; the producer never imports it, never knows it exists, and satisfies it structurally. This is exactly the consumer-side-interface mechanism above, applied proactively during design instead of reactively during a cycle break.

**Caveat: this is a heuristic, not dogma.**

- Do not mechanically split every struct into an interface-plus-implementation pair "for testability" — a concrete struct with no interface is simpler, and an interface with one implementation and no test double is pure indirection.
- Do not return an interface from a constructor "for future flexibility" — that flexibility has a name (YAGNI) and a cost (the caller loses type information it might have wanted).

→ See `fabianoflorentino/golang-agent-skills@golang-project-layout` for directory/package conventions, and `fabianoflorentino/golang-agent-skills@golang-design-patterns` for judging when an interface is the right call versus premature abstraction.

### Splitting a god package

Before a full package split, gopls's `refactor.extract.toNewFile` code action handles the lighter case — moving a top-level declaration to a new file in the *same* package, often enough to make a bloated package navigable without touching its import graph. For a real split, gopls's experimental `source.splitPackage` action assigns top-level declarations to acyclic components. Treat its output as a draft partition to review, not a final answer — it cannot know which grouping matches the domain boundaries you want.

## Moving types across packages: aliases for gradual repair

This is the single most load-bearing Go-specific refactoring technique, and it exists because of a problem unique to Go's type system: type identity is tied to the fully qualified name. `pkg2.T` is a genuinely different type from `pkg1.T` even when their definitions are byte-for-byte identical — you cannot assign one where the other is expected, and — unlike a moved function (re-exportable as a thin wrapper) or a moved variable (re-declarable pointing at the new location) — there was no way to migrate callers gradually. Before Go 1.9, moving a type meant one atomic commit touching every call site, because no intermediate state kept both names working.

The fix is `type A = B` — a **type alias**, not a new named type. It declares `A` and `B` the *same* type, not merely convertible: code written against the old name and code against the new name interoperate exactly, with zero runtime cost and no wrapper anywhere. The language added it, per its design proposal, specifically "to enable gradual code repair during large-scale refactorings, in particular moving a type from one package to another."

The migration recipe:

```go
// Step 1 — new package: the real definition in its new home.
package newpkg

type NewName struct {
    // ... real fields
}
```

```go
// Step 2 — old package: replace the original declaration with an alias
// and mark it deprecated so tooling and editors surface the migration.
package oldpkg

// Deprecated: use newpkg.NewName instead.
type OldName = newpkg.NewName
```

1. Introduce the type with its real definition in the new home package.
2. In the old package, replace the original declaration with an alias to the new one, marked `// Deprecated: use newpkg.NewName instead`.
3. Migrate callers to the new import path incrementally — one PR or one package at a time. Both names remain fully valid and interchangeable throughout; there is no flag day, no big-bang commit, no window where some callers are broken.
4. Delete the alias once nothing references the old name.

Go 1.24 extended type aliases to carry type parameters, so the recipe applies unchanged to generic types.

## Exported API surface and versioning

**Deprecate before deleting.** A doc comment beginning `// Deprecated: ...` is recognized by tooling and editors and surfaces as a warning at every call site, giving callers time to migrate. Deleting an exported identifier outright breaks every downstream module at its next build with no warning.

**Prefer additive changes.** Go's compatibility promise sets the default posture: add (new function, optional field, method) rather than change an existing signature, because additive changes never break an existing caller. A change that must break callers is not a minor bump — it is a new major version.

**Semantic import versioning** is how Go expresses that: a v2+ module carries a `/vN` suffix in both its module path and every importer's import path (e.g. `example.com/mod/v2`). That suffix is what lets v1 and v2 coexist in the same build — to the toolchain they are simply different packages with different paths — and what lets callers migrate one package at a time across major versions, the same gradual property a type alias gives within one version. During a transition, write the v2 implementation as a thin wrapper over v1 (or vice versa) to avoid maintaining two divergent copies of the same behavior.

**`retract` directives** mark an already-shipped defective version as unfit for use: `go get` and `go list -m -u` surface the retraction to dependents without deleting the broken version from the module proxy:

```
module example.com/mod

go 1.24

retract (
    v1.2.0 // published with a data-loss bug, see #123
    [v1.2.1, v1.2.3] // range — a whole span of bad releases
)
```

## `init()`, global state, and package-level vars

Mutable package-level state is hidden coupling: from a call site you cannot see that a function's behavior depends on another package's `init()` having run, or on a global an unrelated path mutated earlier. That coupling lives outside the signature, so it never shows up in a diff, and it is exactly the dependency that makes code unsafe to move or test in isolation.

The refactor is toward explicit construction: a constructor returning a struct, with dependencies passed as parameters rather than reached through a package-level variable or an `init()`-populated singleton. This does not remove the dependency — the code still needs what it needed — it makes the dependency a visible seam in the signature instead of an implicit one buried in `init()`. → See `fabianoflorentino/golang-agent-skills@golang-design-patterns` for constructor/DI patterns, and [safety-net.md](safety-net.md) for how the same seam is what a Feathers-style characterization test exploits.

## Generics — when a refactor toward them is warranted

Introduce a type parameter only when the *logic* is genuinely identical across types, not merely similar:

- When behavior differs per type — even by one branch — that is an interface, not a generic. A generic with a type switch inside it is usually an interface wearing a disguise.
- A practical litmus test: reach for a generic when a type parameter would eliminate a type assertion currently in the code, and the constraint stays narrow — `comparable`, `cmp.Ordered`, or a small one-method interface.
- Write the concrete version first; go generic only once the duplication is real and already committed in two or more places, not anticipated for a future third caller.
- Go has no method-level type parameters, which blocks fluent generic method chaining (a `Map` returning a differently typed receiver) — plan around that limitation rather than discovering it mid-refactor.
- Verify a generics migration like any other refactor: `go build ./... && go vet ./...` plus the full test suite for the touched packages.

## Common mistakes

| Mistake | Fix | Why |
| --- | --- | --- |
| `type OldName NewName` (new named type) instead of `type OldName = NewName` | Use `=` — a real alias | Without `=`, `OldName` is a *distinct* type; existing values of the old type stop assigning to it — exactly the break the alias was meant to avoid |
| Breaking a cycle by moving the producer's concrete type into the consumer | Define the interface in the consumer; leave the producer's type where it is | Moving the concrete type usually just relocates the cycle to whatever else the producer's type depends on |
| Deleting an exported symbol in the same PR that deprecates it | Deprecate, land, wait a release cycle, delete later | Downstream callers cannot react to a deprecation notice they never saw |
| Bumping a module to v2 without adding `/v2` to the module path | Add `/v2` to both the `module` line and every import path | Without the suffix, module resolution cannot distinguish the major versions, and v1 importers get pulled onto breaking code at the next `go get -u` |
| Introducing a generic on the second similar-looking function | Wait for a third real occurrence with identical logic, or an existing assertion the generic removes | Two occurrences are often coincidentally similar; a premature generic ossifies an abstraction around a coincidence |

## Cross-references

- `fabianoflorentino/golang-agent-skills@golang-project-layout` — directory/package layout conventions.
- `fabianoflorentino/golang-agent-skills@golang-design-patterns` — when an interface is right versus premature abstraction, and constructor/DI patterns.
- [catalog.md](catalog.md) — the Fowler entries (Extract Interface, Change Function Declaration, Move Function) these moves build on.
- [workflow.md](workflow.md) — staging a cross-package move or package split as ordered small PRs.
