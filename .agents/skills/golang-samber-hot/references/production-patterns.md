# Production Patterns

Patterns that move a cache from demo to production: background revalidation, contention reduction, negative caching, multi-tier loaders, value safety, monitoring, and startup/shutdown hygiene.

## Stale-while-revalidate

Serve stale data instantly while a loader refreshes it in the background. Two clocks govern an entry:

1. **TTL** — after it lapses, the entry is *stale* and an async refresh is triggered.
2. **Stale duration** — after TTL + stale, the entry is hard-expired and removed.

```go
cache := hot.NewHotCache[string, *Config](hot.WTinyLFU, 1_000).
    WithTTL(5 * time.Minute).                       // stale after 5 min
    WithRevalidation(1 * time.Minute, refreshLoader). // hard-expire after 6 min
    WithRevalidationErrorPolicy(hot.KeepOnError).   // keep stale value on failure
    WithJitter(0.1, 30*time.Second).                // spread expirations
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

Timeline for an entry set at T=0: fresh until T+5min, stale-but-served from T+5min to T+6min while a refresh runs, expired after. On `KeepOnError` a failed refresh leaves the stale value in place until hard expiry; `DropOnError` evicts it immediately. Prefer `KeepOnError` when stale data beats no data (config, catalogs); `DropOnError` when correctness outranks availability.

## Sharding

Split the cache into N independent segments so hot keys rarely contend on the same lock:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 100_000).
    WithTTL(5 * time.Minute).
    WithSharding(16, func(key string) uint64 {
        h := fnv.New64a()
        h.Write([]byte(key))
        return h.Sum64()
    }).
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

Sizing guidance:

- Use powers of two (4, 8, 16, 32) for even key distribution.
- Target roughly the core count for high-contention workloads.
- Each shard holds `capacity / shards` entries.
- More than ~64 shards adds overhead without payoff.

## Missing-key caching

Negative caching stops the loader from being hammered by keys that do not exist upstream.

**Dedicated missing cache** — an independent policy and capacity give fine-grained control:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 100_000).
    WithTTL(1 * time.Hour).
    WithMissingCache(hot.LFU, 10_000).
    WithLoaders(userLoader).
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

**Shared missing cache** — `WithMissingSharedCache()` stores misses in the main cache; simpler, but burns main capacity.

**Manual marking** — `SetMissing(key)` and `SetMissingWithTTL(key, ttl)` mark individual keys; `SetMissingMany` batches. Remember that `Keys()`, `Values()`, and `All()` return only real values and exclude miss entries.

## Loader chains

Sequential loaders express L1/L2 patterns — a fast tier first, a slow tier for the leftovers:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithLoaders(redisLoader, dbLoader).
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

`redisLoader` sees every missing key; `dbLoader` sees only the keys Redis did not return. A key produced by both resolves to the later value, and any error aborts the chain, discarding earlier partial results.

## Copy-on-read / copy-on-write

Mutable values (pointers, slices, maps) need copying or one careless caller corrupts shared state:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithCopyOnRead(func(u *User) *User {
        copy := *u
        return &copy
    }).
    WithCopyOnWrite(func(u *User) *User {
        copy := *u
        return &copy
    }).
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

`WithCopyOnRead` clones at retrieval, keeping callers' mutations out of the cache; `WithCopyOnWrite` clones at store, keeping external mutation from the snapshot. Use both when callers read and write concurrently, one when the mutation direction is known.

## Prometheus monitoring

Metrics setup:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithPrometheusMetrics("user_cache").
    WithJanitor().
    Build()
defer cache.StopJanitor()

prometheus.MustRegister(cache)
```

Useful PromQL:

```promql
rate(hot_cache_hit_count{cache="user_cache"}[5m]) /
rate(hot_cache_get_count{cache="user_cache"}[5m])

rate(hot_cache_eviction_count{cache="user_cache"}[5m])

hot_cache_len{cache="user_cache"} / hot_cache_capacity{cache="user_cache"}
```

Alert on: hit rate under ~70% sustained (undersized), eviction spikes (working set exceeds capacity), and size near capacity (raise capacity or shorten TTLs).

## Warm-up on startup

Pre-populate before traffic arrives so the first requests do not pay a cold miss:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(1 * time.Hour).
    WithWarmUp(func() (map[string]*User, []string, error) {
        users, err := db.GetFrequentUsers(ctx)
        if err != nil {
            return nil, nil, err
        }
        return users, []string{"deleted-user-1"}, nil
    }).
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

The callback returns values, keys known to be missing, and an error. `WithWarmUpWithTimeout(30*time.Second, fn)` bounds startup time.

## Graceful shutdown

Stop the janitor in teardown or its goroutine keeps sweeping into shutdown:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithJanitor().
    Build()
defer cache.StopJanitor()
```

In orchestrated shutdown, call `cache.StopJanitor()` in the drain phase alongside the other resource cleanup.