# Package Guide

`samber/lo` ships five packages. All share the same functional style and differ in allocation behavior, mutability, and concurrency — so the choice is driven by measurements, not taste.

## Import paths

```go
import (
    "github.com/samber/lo"         // lo  — core, immutable
    "github.com/samber/lo/parallel" // lop — concurrent transforms
    "github.com/samber/lo/mutable"  // lom — in-place mutations
    "github.com/samber/lo/it"       // loi — lazy iterators (Go 1.23+)
    "github.com/samber/lo/exp/simd" // experimental SIMD
)
```

## `lo` — core, immutable

300+ functions that return fresh collections and never touch their input. The mental model is JavaScript's `map`/`filter`/`reduce`, type-checked by generics and free of reflection.

**Characteristics:**

- Every helper allocates a new result slice or map.
- Inputs stay untouched, so concurrent readers of a shared slice are safe.
- Output feeds straight into the next helper, enabling chains.

**Use when:** always start here. The other packages exist for measured bottlenecks, not fashion. `active := lo.Filter(users, func(u User, _ int) bool { return u.Active })` leaves `users` intact.

## `lo/parallel` (`lop`) — concurrent transforms

Parallel variants of `Map`, `ForEach`, `Times`, `GroupBy`, and `PartitionBy`. Each element runs through its own goroutine under an internal pool, and results keep original order despite the concurrency.

**Use when:** the transform is CPU-bound and the dataset is large (roughly 1000+ items) — parsing, hashing, or heavy compute. **Do not use for** small slices (goroutine overhead dominates), I/O-bound work (a context-aware `errgroup` is the right tool), or trivial transforms such as field access (plain `lo.Map` is faster).

```go
parsed := lop.Map(rawDocs, func(doc []byte, _ int) *Document {
    return parseDocument(doc)
})
```

**Diagnose first:** `go tool pprof -cpu` should show the transform dominating the profile before you switch.

## `lo/mutable` (`lom`) — in-place mutations

`Filter`, `Map`, `Shuffle`, `Reverse`, and `Replace` operate on the source slice with zero allocation. `lom.Filter` shortens the slice, `lom.Map` rewrites elements in place, and `Shuffle` uses Fisher-Yates.

**Use when:** pprof `-alloc_objects` names `lo.Filter`/`lo.Map` as top allocators on a hot path, or GC pressure from a very large slice is measurable. **Do not use when** the original data is still needed, the slice is read by multiple goroutines, or readability outweighs the saving.

```go
items = lom.Filter(items, func(item Item, _ int) bool {
    return item.Price > 0
})
```

**Diagnose:** confirm with `go tool pprof -alloc_objects` which calls allocate most, then `go build -gcflags="-m"` to see which results escape to the heap.

## `lo/it` (`loi`) — lazy iterators

Go 1.23+ `range`-over-func iterators that defer work until consumption, so a `Map → Filter → Take` chain never materializes intermediate slices. Sub-modules (`channel`, `find`, `intersect`, `map`, `math`, `seq`, `string`, `tuples`, `type_manipulation`) mirror the eager functions.

**Use when:** chaining 3+ transforms over large data, or consuming only a subset (`Take`/`TakeWhile`). **Do not use when** the Go version is below 1.23, the transform is a single step (plain `lo.Map` is clearer), or you need random access to intermediate results.

```go
for name := range loi.Map(
    loi.Filter(users, func(u User) bool { return u.Active }),
    func(u User) string { return u.Name },
) {
    fmt.Println(name)
}
```

## `lo/exp/simd` — experimental SIMD

SIMD-accelerated numeric bulk operations for amd64. The package sits outside semver guarantees and may change between minor releases, so pin the version before production use and adopt it only when benchmarks prove a numeric bottleneck.

## Decision flowchart

```
Start with plain lo (immutable, safe)
  ├─ Allocation pressure? ─────────────► lom for the offending calls
  ├─ CPU-bound transform, large data? ─► lop
  ├─ 3+ chained transforms, Go 1.23+? ─► loi (lazy iterators)
  └─ Infinite/time-driven streams? ────► samber/ro (different library)
```

## Comparison table

| Aspect | `lo` | `lop` | `lom` | `loi` | `simd` |
| --- | --- | --- | --- | --- | --- |
| Allocations | New slice/map | New slice/map | Zero (in-place) | Zero (lazy) | Varies |
| Goroutines | None | One per element | None | None | None |
| Order preserved | Yes | Yes | Yes | Yes | Yes |
| Input modified | No | No | Yes | No | Varies |
| Read-safe concurrently | Yes | Yes | No | Yes | Varies |
| Stability | Stable | Stable | Stable | Stable | Experimental |
| Go version | 1.18+ | 1.18+ | 1.18+ | 1.23+ | 1.25+ |