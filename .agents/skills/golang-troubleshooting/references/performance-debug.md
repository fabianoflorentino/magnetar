# Performance Troubleshooting

Capture before you conclude — profiling commands live in [pprof.md](./pprof.md).

## CPU-bound work

Take a 30-second CPU profile and drill down with `top`, `web`, or `list funcName`. Recurring CPU hogs:

1. JSON marshal/unmarshal in the hot path — preallocate buffers or use a faster encoder.
2. Reflection in critical code paths.
3. Repeated allocation churn — `sync.Pool` where objects are short-lived.
4. Hidden O(n²) inside nested loops — the quadratic term dominates.
5. Excessive syscalls — batch the operations.

## Memory growth

Take heap snapshots and diff them over time:

```bash
go tool pprof -base heap1.prof heap2.prof
```

Pair with escape analysis (see [diagnostic-tools.md](./diagnostic-tools.md)) to find values that unexpectedly escape to the heap. Common leak shapes:

1. Unbounded caches with no eviction policy.
2. Slices growing inside loops that are never reset.
3. Package-level maps that only ever get appended to.
4. String concatenation in loops — `strings.Builder`.
5. Large structs passed by value.

## Lock contention

**Symptoms:** high CPU with low throughput, latency rising with load, extra cores buying nothing.

Enable the extra profiles, then read them:

```go
runtime.SetMutexProfileFraction(1)
runtime.SetBlockProfileRate(1)
```

Validate a mutex/block profile before tuning. Remedies, in increasing scope:

1. **Shrink the critical section** — hold the lock for minimal work.
2. **Shard** — several locks, each guarding its own slice of data.
3. **`sync.Map`** — read-heavy workloads with a stable key set.
4. **`atomic`** — plain counters and flags.
5. **`RWMutex`** — when reads vastly outnumber writes.