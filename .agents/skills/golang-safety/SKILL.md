---
name: golang-safety
description: "Defensive Golang coding against accidental bugs — nil panics, typed-nil interfaces, `append` backing-array aliasing, silent int64-to-int32 truncation, float `==` comparison, `defer` inside loops, defensive copies of slices and maps, and usable zero values. Use when a Go program panics on a nil map write or nil pointer dereference, when reviewing code for nil-safety, numeric conversion overflow, or resource lifecycle, or when designing a type whose zero value must be safe. Not for designing concurrent access with goroutines, channels, or sync primitives (→ See `fabianoflorentino/golang-agent-skills@golang-concurrency` skill), not for exploitable vulnerabilities such as injection, weak crypto, or leaked secrets (→ See `fabianoflorentino/golang-agent-skills@golang-security` skill), and not for debugging an already-failing program (→ See `fabianoflorentino/golang-agent-skills@golang-troubleshooting` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🛡"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Safety & defensive Go

**Persona:** You are a defensive Go engineer. Every untested assumption about nil, capacity, and numeric range is a latent crash; security is about attackers, safety is about ourselves.

**Modes:**

- **Review** — scan new or changed code for nil panics, silent truncation, aliasing, and leaky internals. Sequential.
- **Design** — shape types so their zero value is usable and their exports cannot be mutated. Sequential.
- **Debug** — a nil deref or wraparound bites: reproduce, isolate the assumption, fix the guard. Sequential.

**When to use:** anything that touches nil, integer conversions, float equality, borrowing arrays, `defer`, or exported slices/maps. Concurrency concerns → `golang-concurrency`; exploitable vulnerabilities → `golang-security`; already-failing programs → `golang-troubleshooting`.

## The traps, ranked by frequency

1. **Typed nil in an interface is not `== nil`.** An interface is only nil when both type and value are nil; returning a typed nil pointer makes `(type, nil)` and survives empty checks.
2. **Nil map writes panic.** Reading a nil map is fine (zero value), writing is a crash.
3. **`append` can reuse the backing array.** With slack capacity the two slices share memory and silently corrupt each other.
4. **Integer conversion truncates silently.** `int64(3_000_000_000)` → `int32` wraps to −1.29 billion with no error.
5. **Float equality is a lie.** `0.1 + 0.2 != 0.3` in IEEE 754.
6. **`defer` runs at function exit, not loop iteration** — resources accumulate until the function returns.
7. **Exported slices/maps are open handles** into your struct's internals.
8. **Uninitialized zero values** — a nil map field panics on first write.

## The nil interface trap

```go
func getHandler() http.Handler {
    var h *MyHandler
    if !enabled {
        return h // (type: *MyHandler, value: nil) != nil!
    }
    return h
}
```

Callers check `if handler == nil` and never trigger. Return plain `nil` for the absent case so the empty check works.

## Nil behaviors

| Type | Read | Write | Len/Cap | Range |
| --- | --- | --- | --- | --- |
| Map | zero value | **panic** | 0 | 0 iterations |
| Slice | **panic** | **panic** | 0 | 0 iterations |
| Channel | blocks forever | blocks forever | 0 | blocks forever |

```go
var m map[string]int
m["key"] = 1 // panic

func (r *Registry) Add(k string, v int) {
    if r.items == nil {
        r.items = make(map[string]int) // lazy init keeps zero value usable
    }
    r.items[k] = v
}
```

## Aliasing and defensive copies

```go
a := make([]int, 3, 5)
b := append(a, 4)
b[0] = 99 // also mutates a[0] — shared backing array
```

Force an independent allocation when the caller must not alias: `append(a[:len(a):len(a)], 4)` (full slice expression) or `slices.Clone`. On the way out of a type, never hand a caller a live reference to internals:

```go
type Config struct {
    hosts []string // unexported
}
func (c *Config) Hosts() []string {
    return slices.Clone(c.hosts) // copy at the boundary
}
```

## Numeric safety

```go
var val int64 = 3_000_000_000
i32 := int32(val) // silently -1294967296

if val > math.MaxInt32 || val < math.MinInt32 {
    return fmt.Errorf("value %d overflows int32", val)
}
i32 := int32(val)
```

Float comparisons need an epsilon:

```go
if math.Abs((a+b)-c) < 1e-9 { /* equal enough */ }
```

Integer division by zero panics; float divided by zero yields `±Inf`/`NaN`. Guard the integer case before dividing.

## Resource lifecycle

`defer` in a loop holds every resource until the function returns:

```go
for _, path := range paths {
    f, _ := os.Open(path)
    defer f.Close() // all files stay open until the function exits
    process(f)
}
```

Extract the body so `defer` runs per iteration:

```go
for _, path := range paths {
    if err := processOne(path); err != nil {
        return err
    }
}
```

## Zero-value design

Make `var x T` safe by construction: `sync.Mutex`, `bytes.Buffer`, and `io.Reader` are usable at zero; a type with a map/slice field is not. Two patterns:

```go
// Lazy init in the accessor (above)…
// …or exactly-once under concurrency:
type DB struct {
    once sync.Once
    conn *sql.DB
}
func (db *DB) connection() *sql.DB {
    db.once.Do(func() { db.conn, _ = sql.Open("postgres", connStr) })
    return db.conn
}
```

Avoid `init()` ordering games — use explicit constructors (see `golang-design-patterns`).

## Safer type assertions

- Bare `x.(T)` panics on mismatch — prefer `v, ok := x.(T)`.
- Go 1.25+ reflection: `reflect.TypeAssert[T](v)` over `value.Interface().(T)`.
- Favor generics over `any` where the type set is known — the mismatch becomes compile-time.

## Enforce with linters

`errcheck`, `forcetypeassert`, `nilerr`, `govet`, `staticcheck` catch most of this automatically — see `golang-lint` for the wiring.

## Mistakes at a glance

| Mistake | Fix |
| --- | --- |
| Bare type assertion | `v, ok := x.(T)` |
| Typed nil in interface result | Return plain `nil` for the absent case |
| Nil map write | `make(map[K]V)` or lazy init |
| Assuming `append` copies | Full slice expr or `slices.Clone` |
| `defer` in a loop | Extract the body function |
| `int64` → `int32` without range check | Guard with `math.MaxInt32`/`MinInt32` |
| `a+b == c` floats | `math.Abs(a+b-c) < epsilon` |
| Integer div without zero check | Guard denominator first |
| Returning internal slice/map | Return a copy |
| Nil channel send/receive | Initialize the channel |
| Reliance on `init()` order | Explicit constructors |

## Cross-references

- `golang-concurrency` — concurrent access and sync primitives (this skill is single-threaded by design).
- `golang-error-handling` — the nil error-interface trap overlaps with this one.
- `golang-data-structures` — slice/map internals, capacity growth, the `container/` packages.
- `golang-security` — the adversarial half (memory safety, integer overflow as a vulnerability).
- `golang-troubleshooting` — diagnosing panics and races that already happened.
- `golang-design-patterns` — why `init()` is avoided in favor of constructors.

## References

- [`references/nil-safety.md`](./references/nil-safety.md) — nil receivers, nil in generics, nil-interface performance.
- [`references/slice-map-safety.md`](./references/slice-map-safety.md) — range pitfalls, subslice memory retention, `slices.Clone`/`maps.Clone`.
