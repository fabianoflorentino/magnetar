# Memory Optimization

Cutting allocations is usually the biggest win in a Go program. Every allocation is garbage the collector must later trace and reclaim, so fewer and smaller allocations mean shorter GC pauses and less CPU spent collecting.

## Diagnosing allocation pressure

- `go tool pprof -alloc_objects` — rank allocation sites by count; hot-path functions carrying thousands of allocs/op are the candidates.
- `go build -gcflags="-m -m"` — verbose escape analysis; look for `leaking param`, `too large for stack`, and `captured by closure` on values you expected to stay on the stack.
- `go test -bench=. -benchmem` — measure allocs/op and B/op before and after each change.

## Allocation patterns

### Reuse slices with `append(s[:0], ...)`

Reslicing to zero length keeps the backing array, so overwriting it is a no-op instead of a fresh allocation:

```go
mode = append(mode[:0], item) // reuses backing array, 0 allocs
```

### Prefer direct indexing over append

When the output size equals the input size, allocate once with `make` and assign by index. Per-element `append` adds bounds checks and length bookkeeping:

```go
result := make([]T, len(input))
for i := range input {
    result[i] = transform(input[i])
}
```

Keep `append` when the result can be shorter than the input (filtering) or when an early error return may discard partial results.

### Avoid double map lookups

`for k := range m { use(m[k]) }` performs two lookups per iteration. Range over key and value instead:

```go
for k, v := range in {
    result[k] = fn(v)
}
```

### Size-hint maps

`make(map[K]V, n)` allocates enough buckets up front and skips the rehash growth an empty map pays as it fills:

```go
m := make(map[string]int, len(items))
```

### Sentinel errors instead of `fmt.Errorf`

`fmt.Errorf` allocates on every call. For errors raised repeatedly in hot paths, define a package-level sentinel:

```go
var ErrNegative = errors.New("value is negative") // allocated once

func validate(x int) error {
    if x < 0 {
        return ErrNegative // zero alloc
    }
    return nil
}
```

Reserve `fmt.Errorf` for errors that genuinely need dynamic context (field names, values).

### Avoid interface boxing

Passing concrete values through `any`/`interface{}` boxes them on the heap. Keep hot signatures typed, or use generics:

```go
func sum(values []int) int { ... } // no boxing

func sum[T ~int | ~int64](values []T) T { ... } // generic, still no boxing
```

## Backing array retention

`go tool pprof -inuse_space` shows live heap by allocation site — megabytes held alive that should have been collected points at retention. `-alloc_space` shows cumulative bytes a site produced; volumes far larger than the data they finally hold are the same smell.

### Reslicing keeps the whole array

A small reslice of a large `[]byte` pins the entire backing array. Copy out when you only need a prefix:

```go
func getHeader(data []byte) []byte {
    header := make([]byte, 16)
    copy(header, data[:16]) // independent copy; caller can be freed
    return header
}
```

### Substrings share memory

`msg[:8]` aliases the original string's backing array. Since Go 1.20, `strings.Clone` produces an independent copy:

```go
func extractID(msg string) string {
    return strings.Clone(msg[:8])
}
```

### Maps never shrink

Deleting keys frees entries but not buckets. A map that once held millions of keys keeps its allocation. Rebuild when the map should stop occupying that memory:

```go
func compact(old map[string]Data) map[string]Data {
    m := make(map[string]Data, len(old))
    for k, v := range old {
        m[k] = v
    }
    return m // old map becomes eligible for GC
}
```

## Strings and bytes

`string` ↔ `[]byte` conversions copy and allocate. `go tool pprof -alloc_objects` will show `runtime.stringtoslicebyte` or `runtime.slicebytetostring` near the top when this is a problem.

- **Convert once, reuse** — don't convert inside loops.
- **Use the `bytes` package directly** — `bytes.Contains`, `bytes.HasPrefix`, `bytes.Split`, `bytes.ToUpper` mirror the `strings` API and operate on `[]byte` without conversion.

## `sync.Pool`

Reuse short-lived objects (buffers, scratch structs) in hot paths. Sites allocating the same type thousands of times per second are the classic candidates:

```go
var bufPool = sync.Pool{
    New: func() any { return new([4096]byte) },
}

func process(data []byte) {
    bp := bufPool.Get().(*[4096]byte)
    defer bufPool.Put(bp)
    buf := (*bp)[:0] // reset length, keep capacity
    // ... fill buf ...
}
```

Rules:

- **Reset state before `Put`** — clear references so the pool doesn't retain large object graphs between GC cycles.
- **Return copies** — callers must not hold references to pooled memory.
- **Skip objects >32KB** — they bypass the pool's size classes; the GC handles them directly.
- **Skip rarely used objects** — pool overhead outlives the benefit when allocations are rare.

See `golang-concurrency` for the full `sync.Pool` API reference.

## Memory layout

`fieldalignment ./...` flags structs with wasted padding; `unsafe.Sizeof`/`Alignof`/`Offsetof` confirm exact sizes before and after a reorder.

### Field ordering

The compiler pads fields to their alignment. Ordering largest first removes most padding:

```go
// 24 bytes: bool + 7 pad, int64, bool + 3 pad, int32
type Wide struct{ a bool; b int64; c bool; d int32 }

// 16 bytes: int64, int32, bool, bool + 2 pad
type Tight struct{ b int64; d int32; a bool; c bool }
```

Alignments: 1 for `bool`/`byte`, 2 for `int16`, 4 for `int32`/`float32`, 8 for `int64`/`float64`/`string`/`[]T`/pointer.

### Zero-size trailing fields

A final `struct{}` field forces word-sized trailing padding so a pointer to it can't collide with adjacent memory. Move it out of the tail position:

```go
type Entry struct{ Flag struct{}; Value int64 } // 8 bytes, vs 16 with Flag last
```

### Pointer receivers for large structs

Value receivers copy the whole struct per call. Above roughly 128 bytes, use pointer receivers — consistently across all methods of the type.

### Maps of pointers for frequently updated structs

Map values aren't addressable, so updating in place is impossible without copy-modify-reassign. For large, frequently mutated values, use `map[K]*V`:

```go
players := map[string]*Player{"alice": {Score: 100}}
players["alice"].Score += 10 // direct update, no copy
```

Each pointer is a separate heap allocation, so keep small, mostly-read maps as values.
