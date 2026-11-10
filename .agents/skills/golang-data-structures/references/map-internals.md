# Map Internals

Go maps use hash tables with bucket-based collision resolution. The map header holds `count` (number of entries), `B` (log₂ of bucket count), `buckets` (pointer to bucket array), and `oldbuckets` (pointer to old buckets during growth). Each bucket holds 8 key-value pairs. Keys and values are stored in separate arrays within buckets to minimize padding waste.

## Memory growth and capacity

- Load factor threshold: 6.5 entries per bucket triggers growth (sweet spot between memory efficiency and collision performance)
- Overflow bucket chains also trigger growth if too long (prevents O(1)→O(n) degradation)
- Bucket count doubles: 2^B → 2^(B+1) (efficient rehashing with powers of 2)
- Incremental evacuation: old and new buckets coexist during growth; entries move lazily during operations to avoid GC pauses
- No `cap()` function: capacity depends on hash distribution and load factor, not a fixed limit

## Preallocation

```go
m := map[string]int{}
m := make(map[string]int, expectedSize)
```

Preallocation avoids repeated growths. The hint is approximate — Go allocates 2^B buckets where 2^B * 6.5 >= hint. Preallocation is worthwhile for large maps to avoid repeated growth cycles.

## Pointers vs values

For large value types, storing pointers reduces copy overhead.

```go
m := map[string]BigStruct{}
m := map[string]*BigStruct{}
```

Trade-off: pointer maps add GC pressure. For small structs (< 128 bytes), value maps are typically faster.

## `maps` package (Go 1.21+)

| Function | Description |
| --- | --- |
| `Clone`, `Equal`, `EqualFunc` | Shallow copy and equality comparison |
| `Keys`, `Values`, `All` (1.23+) | Iterators over keys, values, or pairs |
| `Collect`, `Insert` (1.23+) | Build maps from iterators or insert entries |

See `fabianoflorentino/golang-agent-skills@golang-safety` skill for `Clone`, `Equal`, and sorted iteration patterns.

## Map key requirements

Map keys must be comparable (`==` must work). This includes all numeric types, `string`, `bool`, pointers, channels, interfaces (compared by identity), arrays of comparable types, and structs where all fields are comparable.

Slices, maps, and functions cannot be map keys.
