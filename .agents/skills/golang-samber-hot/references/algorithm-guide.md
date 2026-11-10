# Algorithm Selection Guide

The eviction policy drives the hit rate long before TTL does. Choose from the workload's access shape, then verify with hit-ratio metrics.

## Decision tree

```
Unknown access pattern? ──────────────► W-TinyLFU (adapts automatically)
Known pattern:
  Recency dominates (sessions, recent queries) ──► LRU
  Frequency dominates (popular products, DNS, static config):
    popularity stable ───────────────────────────► LFU
    popularity shifts over time ─────────────────► TinyLFU
  Very large cache + high write throughput ──────► S3FIFO
  Shifting pattern, no config appetite ──────────► ARC or W-TinyLFU
  Scan resistance with minimal overhead ─────────► SIEVE
  Everything else ───────────────────────────────► W-TinyLFU (safe default)
```

## Policy deep dives

**LRU** (`hot.LRU`) evicts the value that went longest without an access. A linked list plus hash map keeps every operation cheap and behavior predictable. The failure mode is scan pollution: one sequential pass over cold keys evicts the hot set. Fit: short-lived sessions, recent-search results, expiring tokens.

**LFU** (`hot.LFU`) evicts the least-frequently-accessed key, so real popularity survives regardless of timing. Counts never decay, though — something hot yesterday walls off everything new today. Fit: stable rankings such as DNS records, country-code lookups, static configuration.

**TinyLFU** (`hot.TinyLFU`) tracks frequency through a compact Count-Min Sketch instead of per-key counters, and applied decay lets old hits fade so popularity shifts are absorbed. Frequency memory is cheap and admission filtering is decent; the sketch can admit rare false positives and the filter taxes write-heavy loads. Fit: read-heavy caches with frequency bias — response caching, content metadata.

**W-TinyLFU** (`hot.WTinyLFU`) puts a small recency window in front of TinyLFU's admission filter. New keys enter the window and are promoted to the main cache only if they prove useful, balancing recency and frequency without configuration. It is the best general-purpose hit rate and the safest default; the internals are harder to reason about mid-incident. Fit: mixed or unknown access patterns.

**S3FIFO** (`hot.S3FIFO`) arranges three small/main/ghost FIFO segments. Keys graduate from small to main only on a second access, and the ghost queue remembers recently-evicted keys for scan resistance. FIFO operations are nearly free so throughput is excellent, but the segmented design needs capacity to work — it struggles on small caches. Fit: large (100k+) high-throughput caches with CDN-like access.

**ARC** (`hot.ARC`) maintains recent, recent-ghost, frequent, and frequent-ghost lists, adapting the recency/frequency split to whichever ghost list is hit more. No manual tuning, at roughly 2x tracking memory. Fit: workloads that oscillate between recency and frequency, such as mixed database-query caches.

**TwoQueue** (`hot.TwoQueue`) splits items into a hot and a cold queue with independent eviction; a second access graduates a key to hot. One-hit wonders die in the cold queue while favorites survive. The hot/cold ratio needs tuning and is wasted on uniform access. Fit: clear 80/20 skews.

**SIEVE** (`hot.SIEVE`) keeps a single visited bit per entry and a circular hand pointer. Modern scan-resistant replacement for LRU with near-zero per-item overhead, though less refined than W-TinyLFU or ARC on skewed access. Fit: scan resistance with minimal complexity and cost.

**FIFO** (`hot.FIFO`) evicts the oldest insertion with no tracking at all. Predictable and free per access, but blind to usefulness — any other policy wins on non-uniform access. Fit: TTL-driven caches where keys share a lifetime, such as log buffers or time-series windows.

## Comparison matrix

| Algorithm | Scan resistance | Frequency awareness | Memory overhead | Throughput | Tuning |
| --- | --- | --- | --- | --- | --- |
| LRU | None | None | Low | High | None |
| LFU | None | High (no decay) | Medium | Medium | None |
| TinyLFU | Medium | High (with decay) | Low | Medium | None |
| W-TinyLFU | High | High (with decay) | Low | Medium | None |
| S3FIFO | High | Low | Medium | Very High | None |
| ARC | High | Medium | High (~2x) | Medium | Self-tuning |
| TwoQueue | Medium | Medium | Medium | Medium | Low |
| SIEVE | Medium | None | Very Low | High | None |
| FIFO | None | None | Very Low | Very High | None |

## Measuring hit rate

`WithPrometheusMetrics(name)` exports counters and the cache registers as a `prometheus.Collector`:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithPrometheusMetrics("user_cache").
    WithJanitor().
    Build()
defer cache.StopJanitor()

prometheus.MustRegister(cache)
```

Drive decisions with PromQL:

```promql
rate(hot_cache_hit_count{cache="user_cache"}[5m]) /
rate(hot_cache_get_count{cache="user_cache"}[5m])

rate(hot_cache_eviction_count{cache="user_cache"}[5m])
```

A sustained hit rate below ~80% means undersized capacity or the wrong policy. Increase capacity first, then switch algorithms; rising evictions point at a working set larger than the cache.

## Switching algorithms

The algorithm is the first argument to `NewHotCache`, so the switch is one line and the rest of the builder chain is untouched:

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithJanitor().
    Build()
```

Benchmark hit rate before and after the change; only a measurable gain justifies moving off W-TinyLFU.