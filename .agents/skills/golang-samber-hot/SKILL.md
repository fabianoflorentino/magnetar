---
name: golang-samber-hot
description: "In-memory caching in Golang using samber/hot — eviction algorithms (LRU, LFU, TinyLFU, W-TinyLFU, S3FIFO, ARC, TwoQueue, SIEVE, FIFO), TTL, cache loaders, sharding, stale-while-revalidate, missing key caching, and Prometheus metrics. Apply when using or adopting samber/hot, when the codebase imports github.com/samber/hot, or when the project repeatedly loads the same medium-to-low cardinality resources at high frequency and needs to reduce latency or backend pressure."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔥"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who treats caching as a sizing problem, not a flag to flip. You pick an eviction policy from the workload's access shape, size the cache from a memory budget, and design for expiration jitter, loader failure, and observability.

**Modes:**

- **Build** — adding an in-memory cache or tuning an existing one.
- **Review** — checking capacity math, eviction choice, and missing janitor/provider wiring.
- **Debug** — chasing stale reads, thundering-herd load spikes, or ballooning memory.

**When to use:** any task centered on samber/hot. The skill depends on `golang-performance` (is caching even the right fix?) and `golang-observability` (metrics wiring). For distributed caching, `golang-database` covers the Redis/Memcached side of the decision.

## Choosing an eviction policy

| Policy | Constant | Matches when | Weak spot |
| --- | --- | --- | --- |
| W-TinyLFU | `hot.WTinyLFU` | default; mixed recency+frequency workloads | debugging complexity |
| LRU | `hot.LRU` | recency-dominated (sessions, recent queries) | scan pollution evicts hot keys |
| LFU | `hot.LFU` | frequency-dominated (top products, DNS) | stale hits never leave |
| TinyLFU | `hot.TinyLFU` | read-heavy, frequency-biased | write-heavy overhead |
| S3FIFO | `hot.S3FIFO` | high throughput, scan-resistant | small caches |
| ARC | `hot.ARC` | self-tuning on unknown patterns | ~2× tracking memory |
| TwoQueue | `hot.TwoQueue` | mixed hot/cold split | tuning overhead |
| SIEVE | `hot.SIEVE` | simple scan-resistant LRU successor | skewed access |
| FIFO | `hot.FIFO` | predictable eviction order | hit rate |

Start with `hot.WTinyLFU` and switch only when the miss rate misses your SLO. [Algorithm guide](references/algorithm-guide.md) has the benchmarks and decision tree.

## Building a cache

```go
cache := hot.NewHotCache[string, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithJanitor().
    Build()
defer cache.StopJanitor()

cache.Set("user:123", user)
u, found, err := cache.Get("user:123")
```

The janitor sweeps expired entries; without it they linger until the policy evicts them. `WithJanitor()` and `StopJanitor()` are a matched pair — always both.

## Read-through loaders

Loaders fetch on miss and de-duplicate concurrent requests for the same key with singleflight, so one hot key triggers one backend query, not a stampede:

```go
cache := hot.NewHotCache[int, *User](hot.WTinyLFU, 10_000).
    WithTTL(5 * time.Minute).
    WithLoaders(func(ids []int) (map[int]*User, error) {
        return db.UsersByIDs(ctx, ids) // batch
    }).
    WithJanitor().
    Build()

u, found, err := cache.Get(123) // miss triggers the loader
```

Check `err`, not just `found`: on loader failure `Get` returns `(zero, false, err)` and pretending otherwise hides backend outages.

## Sizing from a memory budget

1. Estimate the per-entry footprint: value struct + heap-held fields (slices, maps, strings) + key, plus ~100 bytes of bookkeeping (pointers, expiries, policy metadata).
2. **Ask the developer** what memory the cache may consume in production (e.g. 256 MB) — depends on process total and neighbors. Use the question tool.
3. Compute `capacity = budget / per-entry`, rounding down.

```
*User ~500B + key ~50B + bookkeeping ~100B ≈ 650B/entry
256 MB → 256_000_000 / 650 ≈ 393,000 entries
```

If entry size is unknown, ask the developer to size it with a unit test (allocate N entries, read `runtime.ReadMemStats`). Guessing capacity invites OOM or wastes RAM.

## Common mistakes

| Mistake | Consequence | Fix |
| --- | --- | --- |
| No janitor | expired entries linger until eviction | `WithJanitor()` + `defer StopJanitor()` |
| `SetMissing` without missing-cache config | runtime panic | `WithMissingCache(...)` or shared missing cache |
| `WithoutLocking()` with janitor | panic; mutually exclusive | lock-free only single-goroutine, no background chores |
| Cache sized to whole dataset | a map with overhead | 10–20% of working set; watch hit rate |
| Ignoring loader `err` | backend failures masquerade as misses | check `err` on every `Get` |
| No TTL | unbounded stale data | always set a TTL |

## Best practices

1. Always set TTL; refresh signals are what keep caches honest.
2. Spread expiry with `WithJitter(lambda, upperBound)` — batch-created keys expire together and thundering-herd the loader.
3. Export `WithPrometheusMetrics(name)`; sustained hit rate under ~80% means the policy or capacity is wrong.
4. `WithCopyOnRead`/`WithCopyOnWrite` for mutable values — shared cached objects get corrupted by one careless caller.

Advanced material (revalidation, sharding, missing cache, monitoring) is in [production patterns](references/production-patterns.md); the full API in [API reference](references/api-reference.md). Library bugs: [samber/hot issues](https://github.com/samber/hot/issues).

## Cross-references

- `golang-performance` — when in-memory cache beats Redis/CDN, and profiling before caching.
- `golang-observability` — Prometheus metrics and fire the dashboards for this cache.
- `golang-database` — batch query patterns that loaders wrap.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.
