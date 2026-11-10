---
name: golang-structs-interfaces
description: 'Golang struct and interface design patterns — composition, embedding, type assertions, type switches, interface segregation, dependency injection via interfaces, struct field tags, and pointer vs value receivers. Use this skill when designing Go types, defining or implementing interfaces, embedding structs or interfaces, writing type assertions or type switches, adding struct field tags for JSON/YAML/DB serialization, or choosing between pointer and value receivers. Also use when the user asks about "accept interfaces, return structs", compile-time interface checks, or composing small interfaces into larger ones.'
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧩"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
paths:
  - "**/*.go"
---

# Structs & interfaces in Go

**Persona:** You are a Go type-system designer. Small composable interfaces and concrete return types, designed for testability and clarity — abstraction paid for by a second consumer, not projected in advance.

**Modes:**

- **Design** — shape types: interfaces, receivers, tags, composition. Ask how the type is consumed before proposing the shape. Sequential.
- **Review** — scan a diff for oversized interfaces, interface returns, premature interfaces, receiver inconsistency, missing tags. Sequential.
- **Refactor** — extract an interface only when a second consumer or a test mock demands it, then apply the rename/change with `gopls`. Sequential.

**When to use:** any Go type design: interfaces, embedding, assertions/switches, serialization tags, receivers. Naming rules for interfaces → `golang-naming`; functional options/builders → `golang-design-patterns`; DI containers → `golang-dependency-injection`.

## Interface principles

1. **Keep interfaces small — 1 to 3 methods.** The bigger the interface, the weaker the abstraction. Compose larger contracts from small ones:

   ```go
   type ReadWriter interface {
       io.Reader
       io.Writer
   }
   ```

2. **Define interfaces where consumed, not implemented.** The consumer controls the contract; the provider package just exports concrete types. `notification` declares `Sender` with only the methods it calls; `email` never imports `notification`.

3. **Accept interfaces, return structs.** Parameters widen; returns should not hide the concrete type's full surface:

   ```go
   func NewService(store UserStore) *Service       // good
   func NewService(store UserStore) ServiceIf      // bad — hides everything
   ```

4. **Do not invent interfaces prematurely.** An interface written before its second implementation is a guess about which methods vary — usually wrong, and it costs indirection while it is wrong. Start concrete; extract once a second consumer, a second implementation, or a test mock demands it:

   ```go
   type UserRepository struct { db *sql.DB }   // start here
   ```

5. **Honor canonical method signatures** — `String() string` (Stringer), `Read([]byte) (int, error)` (Reader). No `ToString()`, no `ReadData()`.

## The zero value is a design target

`var x T` should work: `sync.Mutex`, `bytes.Buffer`, `io.Reader` are zero-usable. A nil map field is not — lazy-init it:

```go
type Registry struct { items map[string]Item }
func (r *Registry) Register(name string, item Item) {
    if r.items == nil {
        r.items = make(map[string]Item)
    }
    r.items[name] = item
}
```

## Prefer generics over `any`

`any` discards type safety; use it only where the type is genuinely unknown (JSON, reflection):

```go
func Contains[T comparable](slice []T, target T) bool   // not []any
```

## Compile-time interface check

```go
var _ io.ReadWriter = (*MyBuffer)(nil)
```

Place next to the type; zero runtime cost; the build fails if `MyBuffer` drifts.

## Type assertions & switches

- Comma-ok form always: `s, ok := val.(string)` — the single-value form panics.
- Type switch for dynamic dispatch; optional narrow assertion for richer implementations the declared type doesn't promise:

  ```go
  if f, ok := w.(Flusher); ok {
      f.Flush()
  }
  ```

- Ordering, nil cases, the optional-behavior pattern: [`references/type-assertions.md`](./references/type-assertions.md).

## Embedding: composition, not inheritance

```go
type Server struct {
    http.Handler     // embed → promote the full API ("is a")
    store *DataStore // named  → private dependency ("has a")
}
```

Promoted methods keep the *inner* receiver; the outer type overrides by defining the same name. Pass such structs by pointer — copying a lock-bearing struct duplicates lock state (`go vet`'s `copylocks`; `noCopy` sentinel documented in [`references/struct-fields.md`](./references/struct-fields.md)).

| Use | When |
| --- | --- |
| Embed | Promote the full API — outer "is a" enhanced version |
| Named field | Internal use only — outer "has a" dependency |

## Dependency injection via interfaces

```go
type UserStore interface {
    FindByID(ctx context.Context, id string) (*User, error)
}
type UserService struct { store UserStore }
func NewUserService(store UserStore) *UserService { return &UserService{store: store} }
```

Tests inject a stub or mock satisfying `UserStore` — no database.

## Struct tags

Exported fields in serialized structs MUST carry tags; without one, renaming the field silently changes the wire format. Full directive table, the `omitempty` vs `omitzero` trap, and vet diagnostics: [`references/struct-fields.md`](./references/struct-fields.md).

```go
type Order struct {
    ID        string    `json:"id"        db:"id"`
    Total     float64   `json:"total"     db:"total"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    Internal  string    `json:"-"         db:"-"`
}
```

## Receiver choice

| Pointer `(s *Server)` | Value `(s Server)` |
| --- | --- |
| Method mutates the receiver | Small, immutable receiver |
| Receiver carries a mutex/channel | Basic type (int, string) |
| Large struct | Read-only accessor |
| Any method on the type uses a pointer | Map/function values |

That last pointer case decides it: receiver style MUST be consistent across every method of a type.

## Common mistakes

| Mistake | Fix |
| --- | --- |
| 5+ method interfaces | Focused 1–3 method interfaces, compose if needed |
| Interface in the implementor's package | Define where consumed |
| Returning interface types | Return concrete |
| Bare assertions | `v, ok := x.(T)` |
| Embed when a delegate suffices | Named field + explicit methods |
| Missing serialization tags | Tag every exported marshaled field |
| Mixed receivers | One style per type |
| No compile-time check | `var _ Iface = (*Type)(nil)` |
| `ToString()` | `String()` |
| Interface with one implementation | Concrete now, extract on demand |
| Nil map/slice zero values | Lazy-init in methods |
| `any` where a type is known | Generics |

## Cross-references

- `golang-naming` — `-er` interfaces, canonical interface names.
- `golang-design-patterns` — functional options, constructors, builders.
- `golang-dependency-injection` — wiring DI with interfaces (and containers).
- `golang-code-style` — value vs pointer *parameters* (distinct from receivers).
- `golang-gopls` — `implementInterface` code action and safe renames across interface satisfaction.
