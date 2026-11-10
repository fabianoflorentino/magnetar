# Caching Patterns

The fastest code never runs. Caching precomputed results, de-duplicating concurrent fetches, and dropping unnecessary computation are often higher-leverage than any micro-optimization.

## Compiled patterns

**Diagnose:** `regexp.Compile`, `regexp.MustCompile`, or `template.Parse*` appearing in a hot profile means patterns are compiled per call instead of once.

- **Regex** — compiling costs ~5,700ns versus ~450ns to match: a 10-12x waste. Compile once at package level; a compiled `*regexp.Regexp` is safe for concurrent use and matching is linear-time (no backtracking):

```go
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`)

func isValid(email string) bool {
    return emailRegex.MatchString(email)
}
```

`MustCompile` panics on an invalid pattern — fine for a startup-time constant, wrong for user-provided patterns (use `regexp.Compile` there).

- **Templates** — parsing is equally expensive. Parse once at startup: `template.Must(template.ParseFiles("templates/report.html"))`.
- **Lookup tables** — a pure function over a small input space can be replaced by array indexing. If the table fits in L1/L2 cache, lookup beats even simple computation:

```go
var hexDigit = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

func byteToHex(b byte) (byte, byte) {
    return hexDigit[b>>4], hexDigit[b&0x0f] // two lookups, no branching
}
```

## Request-level caching

**Diagnose:** the goroutine profile shows many goroutines stuck on the same external call, or `fgprof` shows one fetch function dominating wall-clock across goroutines — a stampede of simultaneous cache misses.

### singleflight

When a cache entry expires, N goroutines can all rediscover the miss and duplicate the fetch. `singleflight` lets one goroutine compute while the rest wait and share the result:

```go
import "golang.org/x/sync/singleflight"

var (
    cache sync.Map
    sf    singleflight.Group
)

func GetWeather(city string) (string, error) {
    if v, ok := cache.Load(city); ok {
        return v.(string), nil
    }
    v, err, _ := sf.Do(city, func() (any, error) {
        data, err := fetchFromAPI(city)
        if err == nil {
            cache.Store(city, data)
        }
        return data, err
    })
    return v.(string), err
}
```

The stdlib `singleflight.Group` boxes through `any`; `github.com/samber/go-singleflightx` is the generic equivalent, roughly 2-4x faster at result retrieval. See `golang-concurrency` for `sync.Map` vs `RWMutex` guidance.

### Bounded LRU caches

For eviction caches, stdlib `container/list` nodes are scattered heap objects with poor locality. Thread-safe, high-throughput options:

- `github.com/hashicorp/golang-lru` — simple, widely used.
- `github.com/elastic/go-freelru` — hashmap and ringbuffer in one contiguous allocation; claims roughly 37x over sharded implementations.

Verify current APIs in each library's docs.

## Algorithmic complexity

Before constant-factor tricks, confirm the algorithm isn't the bottleneck. A naive O(n log n) beats a micro-optimized O(n²) at scale.

| Pattern | Complexity | Fix | Fixed |
| --- | --- | --- | --- |
| `slices.Contains` inside a loop | O(n·m) | build `map[T]struct{}`, then look up | O(n+m) |
| Nested loops for matching | O(n²) | index with a map, or `slices.BinarySearch` | O(n log n)/O(n) |
| Repeated `append` growth | O(n²) copies | `make([]T, 0, n)` | O(n) |
| String `+=` concatenation | O(n²) copies | `strings.Builder` | O(n) |
| Repeated min/max/dedup scans | O(n) per query | sort once, query via binary search | amortized low |

Verify with a size sweep (100 → 1K → 10K inputs): if time grows roughly quadratically, replace the algorithm first, then tune constants.

## Work avoidance

**Diagnose:** linear scans (`slices.Contains`, `slices.Index`) or iterator chains consuming CPU in a hot path.

- **Map over slice scan** — membership tests in a loop are O(n·m). Build a `map[T]struct{}` once and look up in O(1); use `struct{}` (0 bytes), not `bool`.
- **Early returns** — return the moment the answer is known; finding the target at iteration 3 of 1000 skips 997 iterations.
- **No iterator chains** — `Filter → Map → First` builds closures and intermediate machinery. A plain loop with an early return is simpler and faster:

```go
for i := range items {
    if predicate(items[i]) {
        return items[i], true
    }
}
return zero, false
```

- **Direct loops over functional wrappers** — wrapping one call in another (e.g. `Map(items, func(p *T) T { return *p })`) blocks inlining. A direct loop inlines and is measurably faster (around 13-17% in the reported case):

```go
func FromSlicePtr(items []*T) []T {
    result := make([]T, len(items))
    for i := range items {
        result[i] = *items[i]
    }
    return result
}
```
