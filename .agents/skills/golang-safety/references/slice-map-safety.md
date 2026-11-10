# Slice and Map Safety Deep Dive

Slices and maps look like values but behave like handles: a slice carries a pointer into a shared backing array, and ranging over either has corner cases that produce silent corruption or surprising ordering. This reference covers the ones that bite.

## Range loop variable capture

**Before Go 1.22** the loop variable was one variable reused across iterations. Capturing it in a closure — or storing its address — left every reference pointing at the final value:

```go
// Pre-1.22 bug — every closure prints "c".
var funcs []func()
for _, v := range []string{"a", "b", "c"} {
    funcs = append(funcs, func() { fmt.Println(v) })
}
for _, f := range funcs {
    f()
}
```

```go
// Pre-1.22 fix — shadow the variable per iteration.
for _, v := range []string{"a", "b", "c"} {
    v := v
    funcs = append(funcs, func() { fmt.Println(v) })
}
```

**Go 1.22+** creates a fresh variable each iteration, so the closure bug is gone. The old behavior applies only if your `go.mod` still declares `go 1.21` or earlier — check it before assuming the fix is in effect.

## Storing a pointer to the loop variable

The same pre-1.22 trap applies to `&v`. Take the address of the slice element instead:

```go
type Item struct{ Name string }
items := []Item{{Name: "a"}, {Name: "b"}}
var ptrs []*Item
for _, item := range items {
    ptrs = append(ptrs, &item) // all point at the same loop variable
}
// ptrs[0].Name == "b", ptrs[1].Name == "b"

for i := range items {
    ptrs = append(ptrs, &items[i]) // each element's own address
}
```

On Go 1.22+ `&item` is safe because each iteration has its own copy — but `&items[i]` is still clearer and avoids the copy.

## Slice header vs. backing array

A slice is a three-word struct `{pointer, length, capacity}`. Multiple slices can share one backing array, which is where `append` gets dangerous:

```go
a := make([]int, 3, 5)
b := a[1:2]             // shares a's backing array
append(a, 4)            // has spare capacity — writes into the shared array
```

When the caller must not alias, force an independent allocation with the full slice expression `a[:len(a):len(a)]` (which sets `cap == len`, so the next `append` allocates) or `slices.Clone`.

## Subslices keep the whole backing array alive

A small subslice of a large slice pins the entire backing array for GC:

```go
func getHeader(data []byte) []byte {
    return data[:64] // the whole backing array stays alive
}
```

```go
func getHeader(data []byte) []byte {
    header := make([]byte, 64)
    copy(header, data[:64]) // release the large array
    return header
}

func getHeader(data []byte) []byte {
    return slices.Clone(data[:64]) // Go 1.21+
}
```

This matters when you slice a 1MB buffer and keep a 64-byte piece — without the copy, the whole megabyte stays in memory.

## Clone helpers (Go 1.21+)

`slices.Clone` and `maps.Clone` are the preferred way to make defensive copies. They are clearer than manual `make` + `copy` and handle nil input correctly (returning nil, not an empty collection):

```go
import (
    "maps"
    "slices"
)

clone := slices.Clone(original) // shallow slice copy
clone := maps.Clone(original)   // shallow map copy
```

## Map iteration order

Map iteration order is randomized by the runtime — never depend on it:

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}
for k, v := range m {
    fmt.Printf("%s=%d ", k, v) // any permutation is legal
}
```

When order matters, sort the keys (Go 1.23+):

```go
keys := slices.Sorted(maps.Keys(m))
for _, k := range keys {
    fmt.Printf("%s=%d ", k, m[k])
}
```

## Deleting during iteration

**Maps — safe by definition.** Deleting entries during `range` over a map is explicitly supported:

```go
for k, v := range m {
    if shouldDelete(v) {
        delete(m, k)
    }
}
```

**Slices — forward iteration skips elements.** Deleting mid-iteration shifts the tail left and the loop jumps over whatever moved into the deleted slot:

```go
// Buggy — skips the element after each deletion.
for i, v := range items {
    if shouldDelete(v) {
        items = append(items[:i], items[i+1:]...)
    }
}

// Correct — iterate backward.
for i := len(items) - 1; i >= 0; i-- {
    if shouldDelete(items[i]) {
        items = append(items[:i], items[i+1:]...)
    }
}

// Or delegate (Go 1.21+).
items = slices.DeleteFunc(items, shouldDelete)
```

## Comparing slices and maps

`==` does not compile for slices and is meaningless for maps — use the stdlib helpers:

```go
import (
    "maps"
    "slices"
)

slices.Equal(a, b) // element-wise
maps.Equal(m1, m2) // key-value

// Custom comparison.
slices.EqualFunc(a, b, func(x, y Item) bool {
    return x.ID == y.ID
})
```

→ See `fabianoflorentino/golang-agent-skills@golang-modernize` skill for Go 1.22+ loop semantics and the iterator idioms.
