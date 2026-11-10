---
name: golang-data-structures
description: "Golang data structures — slices (internals, capacity growth, preallocation, slices package), maps (internals, hash buckets, maps package), arrays, container/list/heap/ring, strings.Builder vs bytes.Buffer, generic collections, pointers (unsafe.Pointer, weak.Pointer), and copy semantics. Use when choosing or optimizing Go data structures, implementing generic containers, using container/ packages, unsafe or weak pointers, or questioning slice/map internals. Not for applying optimization patterns once profiling has identified a bottleneck (→ See `fabianoflorentino/golang-agent-skills@golang-performance` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🗃"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__* mcp__context7__resolve-library-id mcp__context7__query-docs
paths:
  - "**/*.go"
---

# Data structures in Go

**Persona:** You are a Go engineer who reasons about memory layout, allocation cost, and access patterns. You pick the structure for the job — not the most familiar one — and you know what each structure costs before you reach for it.

**Modes:**

- **Choose** — given the access pattern and sizes, decide the structure and its preallocation. Sequential.
- **Implement** — generic containers and tight-constraint types. Sequential.
- **Review** — audit allocations, growth assumptions, and copy semantics in a diff. Sequential.

**When to use:** selecting or tuning Go structures, generic containers, `container/` packages, unsafe/weak pointers, or slice/map internals. For optimization patterns after profiling, `golang-performance`; for nil/aliasing pitfalls, `golang-safety`.

## Selection rules

1. **Preallocate what you can size** — `make([]T, 0, n)` and `make(map[K]V, n)` avoid repeated O(n) growth copies and rehashing.
2. **Arrays only for fixed, compile-time-known sizes** — digests, IPs, small matrices. Everything else: slices.
3. **Never depend on slice growth timing** — the growth algorithm changed between versions and will again; code must not rely on when a new backing array appears.
4. **`container/heap` for priority queues, `container/list` only for frequent middle edits, `container/ring` for fixed circular buffers** — and remember these wrap `any`, so generic wrappers restore type safety.
5. **`strings.Builder` to build strings; `bytes.Buffer` for bidirectional I/O** (it is an `io.Reader` and `io.Writer`).
6. **Tightest generic constraint** — `comparable` for keys, `cmp.Ordered` for sorting, a custom interface only for domain ordering.
7. **`unsafe.Pointer` follows the spec's six conversion shapes only** — never hold it as a `uintptr` across statements.
8. **`weak.Pointer[T]` (Go 1.24+) for caches and canonicalization maps** — GC can reclaim entries without you managing reaping.

## Slices

A slice is a three-word header: pointer, length, capacity. Two slices can share a backing array — that is the source of the aliasing traps in `golang-safety`.

Growth: below 256 elements capacity doubles; at 256+ it grows by ~25% (`newcap += (newcap + 3*256) / 4`); every growth copies the whole backing array.

```go
users := make([]User, 0, len(ids))        // exact size
results := make([]Result, 0, est)         // estimate is fine
s = slices.Grow(s, additionalNeeded)      // pre-grow before bulk append (1.21+)
```

`slices` package (Go 1.21+): `Sort`/`SortFunc`, `BinarySearch`, `Contains`, `Compact`, `Grow`. `Clone`, `Equal`, `DeleteFunc` → `golang-safety`. Full depth in [`references/slice-internals.md`](./references/slice-internals.md).

## Maps

Hash tables with 8-entry buckets and overflow chains; assigning a map copies the pointer, not the data.

```go
m := make(map[string]*User, len(users))   // avoids rehash while populating
```

Go 1.23+ `maps` iterators: `Collect`, `Insert`, `All`, `Keys`, `Values`. `Clone`, `Equal` → `golang-safety`. Internals, why maps never shrink, and what that costs: [`references/map-internals.md`](./references/map-internals.md).

## Arrays

Value types: assignment copies the whole array. Comparable — usable as map keys:

```go
type Digest [32]byte
grid := [3][3]int{}
cache := map[[2]int]Result{}
```

## container/ packages

| Package | Data structure | Best for |
| --- | --- | --- |
| `container/list` | Doubly-linked list | LRU, middle insertions |
| `container/heap` | Priority queue | Top-K, scheduling, Dijkstra |
| `container/ring` | Circular buffer | Rolling windows, round-robin |
| `bufio` | Buffered I/O | Many small reads/writes |

## Builder choice

`strings.Builder` avoids the `String()` copy — use it for pure concatenation. `bytes.Buffer` when you need bytes, `io.Reader`/`io.Writer` semantics, or direct mutation. Both support `Grow(n)`. [`references/containers.md`](./references/containers.md).

## Generic collections

The tightest constraint that works is the right one:

```go
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T)          { s[v] = struct{}{} }
func (s Set[T]) Contains(v T) bool { _, ok := s[v]; return ok }
```

Constraint satisfaction and domain-specific generics: [`references/generics.md`](./references/generics.md).

## Pointer types

| Type | Use | Zero |
| --- | --- | --- |
| `*T` | Indirection, mutation, optional values | `nil` |
| `unsafe.Pointer` | FFI, low-level layout (6 spec shapes only) | `nil` |
| `weak.Pointer[T]` (1.24+) | Caches, canonicalization | n/a |

## Copy semantics

| Type | Copy |
| --- | --- |
| `int`, `float`, `bool`, `string` | Deep, independent |
| `array`, `struct` | Deep, independent |
| `slice` | Header copied; backing array shared → `slices.Clone` |
| `map` | Reference copied → `maps.Clone` |
| `channel` | Reference copied (same channel) |
| `*T` | Address copied (same value) |
| `interface` | Value pair copied; independence depends on held type |

## Common mistakes

| Mistake | Fix |
| --- | --- |
| Appending into a loop without preallocation | `make([]T, 0, n)` or `slices.Grow` |
| `container/list` where a slice suffices | List nodes are separate allocations, poor locality — benchmark |
| `bytes.Buffer` for pure string building | `strings.Builder` avoids the copy |
| `unsafe.Pointer` kept as `uintptr` across statements | GC may move the object; dangling reference |
| Large struct values in maps | `map[K]*V` avoids copying the whole value on access |

## Feature richness

For third-party structures (trees, sets, deques): `emirpasic/gods`, `deckarep/golang-set`, `gammazero/deque`. Verify symbols/versions/vulnerabilities via `golang-pkg-go-dev` (`godig`) before reaching for Context7; navigate their use in your code with `golang-gopls` (`gopls`); Context7 stays a fallback for docs not on pkg.go.dev.

## Cross-references

- `golang-performance` — field alignment, layout, cache locality once a bottleneck is profiled.
- `golang-safety` — nil maps, `append` aliasing, defensive copies, `Clone`/`Equal`.
- `golang-concurrency` — channels, `sync.Map`, `sync.Pool`.
- `golang-design-patterns` — string vs []byte vs []rune, iterators, streaming.
- `golang-structs-interfaces` — composition, embedding, generics vs `any`.
- `golang-code-style` — slice/map initialization style.

## External references

- [Go Data Structures](https://research.swtch.com/godata) — Russ Cox's anatomy of the built-ins.
- [The Go Memory Model](https://go.dev/ref/mem)
- [Effective Go](https://go.dev/doc/effective_go)
