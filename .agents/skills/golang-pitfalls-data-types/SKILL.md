---
name: golang-pitfalls-data-types
description: "Golang numeric types and data structures — octal literal confusion, integer overflow, floating-point approximations, slice length vs capacity, inefficient slice initialization, nil vs empty slices, emptiness checks, slice copies, append side effects, slice memory leaks, map initialization, map memory leaks, and incorrect value comparison. Distilled from mistakes #17-29 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang code dealing with numbers, slices, maps, or comparing values."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧬"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Data Types

Source material: mistakes #17-29 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when writing Go code around numeric types, slices, maps, and value comparison.

## 17. Octal literals confusion (#17)

- Integer literals starting with `0` are octal (`010` == 8 decimal).
- Use explicit `0o` prefix for octal (`0o10`) to avoid mistakes.
- Other representations: binary `0b`, hex `0x`, imaginary `i` suffix; `_` as digit separator (`1_000_000_000`, `0b00_00_01`).

## 18. Neglecting integer overflows (#18)

- Compile-time overflow/underflow of constants is an error; **runtime** overflow/underflow is silent — no panic.
- This can turn positive additions into negative results. Write your own checks when overflow matters.

```go
// silent at runtime
var counter int32 = math.MaxInt32
counter++ // wraps silently
```

## 19. Not understanding floating-points (#19)

- `float32`/`float64` are approximations, not exact real arithmetic (e.g., `1.0001 * 1.0001` prints `1.0002`, not `1.00020001`).
- Compare floats by checking the difference is within a delta (e.g., `math.Abs(a-b) <= delta`).
- Group additions/subtractions of similar order of magnitude for accuracy.
- Do multiplication/division before addition/subtraction.

## 20. Slice length vs. capacity (#20)

- **Length**: number of available elements. **Capacity**: number of elements in the backing array (from the slice's first element).
- `s := make([]int, 3, 6)` → len 3, cap 6. Accessing `s[3]` panics even though memory is allocated.
- `append` uses spare capacity; when the backing array is full, Go allocates a new array (doubling capacity), copies elements, and updates the pointer.
- Slicing `s2 := s1[1:3]` shares the backing array; both see element updates. Appending to `s2` within capacity mutates shared memory. Once an append forces a new array, `s1` and `s2` diverge.

## 21. Inefficient slice initialization (#21)

- When the length is known, preallocate with `make([]int, length)` or `make([]int, 0, capacity)`.
- Avoid multiple copies and GC pressure. `make([]int, length)` tends to be slightly faster; `make([]int, 0, cap)` + `append` can be clearer.

## 22. Confusion about nil vs. empty slice (#22)

- A nil slice equals `nil`; an empty slice has length 0 but isn't necessarily `nil`. A nil slice requires **no allocation**.
- Initialization guidance:
  - `var s []string` — when unsure about final length and slice can be empty.
  - `[]string(nil)` — syntactic sugar for a nil, empty slice.
  - `make([]string, length)` — known length.
  - Avoid `[]string{}` when initializing without elements.
- Check how libraries (e.g., `encoding/json`, `reflect`) distinguish nil vs empty to avoid surprises.

## 23. Not properly checking if a slice is empty (#23)

- Check emptiness via `len(s) == 0`; it covers both nil and empty slices (same for maps).
- Don't check `s == nil` to decide emptiness.
- Design APIs that don't distinguish nil vs empty slices/maps — returning nil or empty should mean the same thing.

## 24. Not making slice copies correctly (#24)

- `copy(dst, src)` copies `min(len(dst), len(src))` elements.
- Ensure the destination has enough length — allocate with `make([]T, len(src))` first.

## 25. Unexpected side effects with slice append (#25)

- When a slice's length is smaller than its capacity, `append` can mutate the backing array visible to other slices — a subtle side effect.
- Prevent with **full slice expression** `s[low:high:max]` (sets capacity to `max-low`) or a **slice copy**.
- Only a real copy prevents memory leaks when shrinking a large slice (see #26).

## 26. Slices and memory leaks (#26)

- **Leaking capacity**: slicing a large array/slice keeps the whole backing array alive. Use a copy to release memory.
- **Slice of pointers / structs with pointer fields**: GC won't reclaim elements excluded by slicing. Either copy or explicitly set remaining elements/fields to `nil`.

## 27. Inefficient map initialization (#27)

- When the number of elements is known up front, create with initial size: `make(map[string]int, n)`.
- Avoids map growth (reallocation + rebalancing), which is expensive.

## 28. Maps and memory leaks (#28)

- A Go map is a hash table: an array of bucket pointers, each bucket holds up to 8 key-value pairs (overflow creates chained buckets).
- **A map never shrinks** — deleting elements zeroes slots but keeps the same number of buckets. Only a growing `B` field (log2 of #buckets) is possible.
- After adding then deleting a million `[128]byte` values, heap stays ~293 MB (vs 461 MB peak) even after GC.
- Mitigations:
  - Re-create/copy the map periodically (temporarily doubles memory).
  - Use pointers in values to shrink per-entry footprint when values are small: `map[int]*[128]byte` (38 MB vs 293 MB after GC).
- Note: keys/values over 128 bytes are stored via pointers automatically.

## 29. Comparing values incorrectly (#29)

- `==`/`!=` work on comparable types: booleans, numbers, strings, channels (same `make` call or nil), interfaces (identical dynamic type & value), pointers, arrays/structs composed of comparable types.
- Slices and maps are not comparable.
- For non-comparable types use `reflect.DeepEqual` (correct but slow — reflection) or custom methods if performance matters.
- Prefer existing optimized stdlib comparators (e.g., `bytes.Compare`) over reinventing the wheel.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-data-structures` for slice/map/channel internals and usage
- → See `fabianoflorentino/golang-agent-skills@golang-performance` for allocation-aware initialization
- → See `fabianoflorentino/golang-agent-skills@golang-safety` for float comparison and nil-safety patterns
