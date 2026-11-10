# API Reference

The full `samber/hot` surface in signature form. Builder methods are called on the value returned by `NewHotCache`, then finalized with `Build()`.

## Constructor

```go
hot.NewHotCache[K comparable, V any](algorithm hot.EvictionAlgorithm, capacity int) *HotCacheBuilder[K, V]
```

Algorithm constants:

| Constant | Policy |
| --- | --- |
| `hot.LRU` | Least Recently Used |
| `hot.LFU` | Least Frequently Used |
| `hot.TinyLFU` | TinyLFU with frequency decay |
| `hot.WTinyLFU` | Weighted TinyLFU; recommended default |
| `hot.S3FIFO` | Segmented small-size FIFO |
| `hot.ARC` | Adaptive Replacement Cache |
| `hot.TwoQueue` | Two-Queue |
| `hot.SIEVE` | SIEVE |
| `hot.FIFO` | First In, First Out |

## Builder methods

| Method | Effect |
| --- | --- |
| `WithTTL(ttl time.Duration)` | Default expiration applied to every entry |
| `WithJitter(lambda float64, upperBound time.Duration)` | Randomize TTL within +/-lambda (capped) to spread expirations |
| `WithJanitor()` | Background goroutine that sweeps expired entries; incompatible with `WithoutLocking()` |
| `WithLoaders(loaders ...Loader[K, V])` | Miss-loader chain; each next loader receives only keys unmapped by the previous |
| `WithRevalidation(stale time.Duration, loaders ...Loader[K, V])` | Stale-while-revalidate; entries refresh in the background after TTL and hard-expire after `stale` |
| `WithRevalidationErrorPolicy(policy)` | `hot.KeepOnError` or `hot.DropOnError` on refresh failure |
| `WithMissingCache(algorithm, capacity)` | Separate cache dedicated to missing keys, with its own eviction |
| `WithMissingSharedCache()` | Store missing keys in the main cache |
| `WithSharding(shards uint64, hasher Hasher[K])` | Split into N shards to cut lock contention; prefer powers of two |
| `WithCopyOnRead(fn func(V) V)` | Clone on retrieval so callers cannot mutate cached values |
| `WithCopyOnWrite(fn func(V) V)` | Clone on store so cached values snapshot, not alias |
| `WithPrometheusMetrics(cacheName string)` | Export counters; the cache registers as a `prometheus.Collector` |
| `WithEvictionCallback(fn func(K, V))` | Synchronous callback on eviction |
| `WithoutLocking()` | Disable mutexes for single-goroutine access; incompatible with `WithJanitor()` |
| `WithWarmUp(fn func() (map[K]V, []K, error))` | Pre-populate at build time; returns values, known-missing keys, error |
| `WithWarmUpWithTimeout(timeout, fn)` | Warm-up bounded by a timeout |
| `Build()` | Finalize and return `*HotCache[K, V]` |

## Read operations

| Method | Returns | Behavior |
| --- | --- | --- |
| `Get(key)` | `(V, bool, error)` | Triggers loaders on miss; `false` with a loader error on backend failure |
| `GetWithLoaders(key, loaders...)` | `(V, bool, error)` | Per-call loader override |
| `GetMany(keys)` | `(map[K]V, []K, error)` | Found map plus the keys still missing |
| `GetManyWithLoaders(keys, loaders...)` | `(map[K]V, []K, error)` | Batch with loader override |
| `MustGet(key)` / `MustGetWithLoaders` | `(V, bool)` | Panics on loader error |
| `MustGetMany(keys)` / `MustGetManyWithLoaders` | `(map[K]V, []K)` | Batch, panics on error |
| `Peek(key)` | `(V, bool)` | No loaders, ignores expiration |
| `PeekMany(keys)` | `(map[K]V, []K)` | Batch peek |
| `Has(key)` | `bool` | Existence check without loader side effects |
| `HasMany(keys)` | `map[K]bool` | Batch existence check |
| `Keys()` | `[]K` | Keys of live values only; missing entries excluded |
| `Values()` | `[]V` | All live values |
| `All()` | `map[K]V` | Key-value snapshot |
| `Range(fn func(K, V) bool)` | — | Iterate; stop early by returning false |
| `Len()` | `int` | Total entry count |
| `Capacity()` | `(int, int)` | Main and missing-cache capacities |
| `Algorithm()` | `(string, string)` | Main and missing-cache algorithm names |

## Write operations

| Method | Behavior |
| --- | --- |
| `Set(key, value)` | Store with the default TTL |
| `SetWithTTL(key, value, ttl)` | Store with a per-entry TTL |
| `SetMany(items)` | Batch store, default TTL |
| `SetManyWithTTL(items, ttl)` | Batch store, custom TTL |
| `SetMissing(key)` | Mark the key as non-existent; requires `WithMissingCache()` or `WithMissingSharedCache()` |
| `SetMissingWithTTL(key, ttl)` | Mark missing with custom TTL |
| `SetMissingMany(keys)` | Batch mark-missing |
| `SetMissingManyWithTTL(keys, ttl)` | Batch mark-missing with custom TTL |

## Maintenance operations

| Method | Behavior |
| --- | --- |
| `Delete(key)` | Remove one key; returns true if present |
| `DeleteMany(keys)` | Batch delete; returns per-key existence map |
| `Purge()` | Clear all entries |
| `WarmUp(fn)` | Pre-populate at runtime |
| `Janitor()` | Start the background expiration sweeper |
| `StopJanitor()` | Stop the background sweeper; always pair with `WithJanitor()` |

## Loader type

```go
type Loader[K comparable, V any] func(keys []K) (found map[K]V, err error)
```

Chain semantics:

- Loaders run sequentially in registration order.
- Each loader receives only keys the earlier loaders did not produce.
- A later loader's value overwrites an earlier one for the same key.
- Any loader error stops the chain and returns it.
- Concurrent `Get` calls for one key share a single invocation via singleflight.

## Hasher type

```go
type Hasher[K any] func(key K) uint64
```

## Prometheus integration

`*HotCache` implements `prometheus.Collector`; register it after enabling metrics to expose counters for hit rate, evictions, and size:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithPrometheusMetrics("user_cache").
    Build()

prometheus.MustRegister(cache)
```