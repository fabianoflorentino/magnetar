# Investigation Session Setup

A performance deep-dive is a temporary operation: enable higher-resolution collection for hours to days while a specific issue is under investigation, then switch it back off. This is not everyday monitoring.

## Standing up the session

1. **Tighten the scrape interval** to ≤10 s on the target instance (vs the usual 15–30 s). Short investigation windows need the extra density. Revert afterward.

   ```bash
   kubectl set env deployment/my-service PPROF_ENABLED=true
   kubectl rollout restart deployment/my-service
   ```

2. **Enable pprof via environment toggle** on one instance — never recompile for an investigation.

3. **Continuous profiling (Pyroscope/Parca) on the target instance only.** Fleet-wide, it overwhelms the backend.

4. **Debug logging on the target only** — it has real throughput cost.

The governing principle: every costly feature — pprof HTTP, continuous profiling, debug level, trace collection — must be switchable by environment variable, designed in from day one.

## The Prometheus Go runtime collector

`prometheus/client_golang` ships collectors that turn Go runtime state into scrapable series — a time-series complement to point-in-time profiles.

Two nuances matter:

- The classic `go_memstats_*` series are backed by `runtime.ReadMemStats()`, which causes a short stop-the-world pause per scrape.
- On Go 1.17+ the preferred `collectors.NewGoCollector()` reads from the cheaper `runtime/metrics`.

```go
import "github.com/prometheus/client_golang/prometheus/collectors"

reg := prometheus.NewRegistry()
reg.MustRegister(collectors.NewGoCollector(
    collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
))
reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
```

The exhaustive metric list (and which series appear by default vs via `MetricsAll`) is in [`prometheus-go-metrics.md`](./prometheus-go-metrics.md). Series availability drifts with Go version — enumerate at runtime with `metrics.All()` when in doubt.

## PromQL deep dives

Raised-frequency queries for the session, with the reading for each.

**GC pressure**

| Query | Reading |
| --- | --- |
| `rate(go_gc_duration_seconds_count[5m])` | Cycles/second; sustained >2 signals an excess allocation rate |
| `rate(go_gc_duration_seconds_sum[5m]) / rate(go_gc_duration_seconds_count[5m])` | Average pause; an upward trend = heap growth or heavy pointer scanning |
| `go_gc_duration_seconds{quantile="1"}` | Worst-case pause; spikes here drive P99 tail latency |

**Memory leaks**

| Query | Reading |
| --- | --- |
| `go_memstats_alloc_bytes` | Stable under constant load; a steady climb is a leak |
| `rate(go_memstats_alloc_bytes_total[5m])` | Allocation rate; compare across deploys |
| `process_resident_memory_bytes - go_memstats_sys_bytes` | Non-Go memory (cgo, mmap); growing gap = non-Go leak |

**Goroutine leaks**

| Query | Reading |
| --- | --- |
| `go_goroutines` | Correlates with load when healthy |
| `delta(go_goroutines[1h])` | Grows without a load increase = leak |

**CPU saturation**

| Query | Reading |
| --- | --- |
| `rate(process_cpu_seconds_total[5m])` | Cores consumed |
| `rate(process_cpu_seconds_total[5m]) / <GOMAXPROCS>` | Utilization ratio; sustained >0.8 = saturated |

**Post-deploy regression**

| Query | Reading |
| --- | --- |
| `rate(go_memstats_alloc_bytes_total[5m])` | Jumps across the deploy window = new allocation pattern |
| `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))` | P99 climb after deploy; needs an app-level histogram |

**Alerting examples**

```yaml
- alert: HighGCPauseTime
  expr: rate(go_gc_duration_seconds_sum[5m]) / rate(go_gc_duration_seconds_count[5m]) > 0.01
  for: 10m
  annotations:
    summary: "Average GC pause >10ms — reduce allocations or tune GOGC"

- alert: GoroutineLeak
  expr: go_goroutines > 10000
  for: 5m

- alert: MemoryNearLimit
  expr: predict_linear(process_resident_memory_bytes[1h], 3600) > <container_limit_bytes>
  for: 15m
```

Tune to the workload — a data pipeline and an API server share no baseline.

## Host-level correlation

Runtime series alone never tell the whole story. `node_exporter` gives host CPU/memory/disk/network — high `node_cpu_seconds_total` with low `process_cpu_seconds_total` means a noisy neighbor, not your app. `process-exporter` separates per-process usage when several Go services share a host.

## Cost warnings

| Feature | Cost |
| --- | --- |
| pprof CPU profiles | CPU-bound while capturing; never chain 30 s captures back-to-back in production |
| Pyroscope continuous profiling | ~2–5% CPU per instance, always on; scales badly across hundreds of instances — enable on a slice on demand |
| Execution traces | MB/s of output; capture 5–10 s |
| Debug log level | Real allocation and I/O overhead; never leave on |

Everything above stays behind an environment variable so it can be flipped off without a rebuild.