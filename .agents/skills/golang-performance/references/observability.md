# Production Observability for Performance

Local profiling finds one-off bottlenecks; production needs always-on signals for historical trends and regression detection.

## Prometheus metrics

Expose `/metrics` via `github.com/prometheus/client_golang` with `promhttp.Handler()`. The default collectors automatically export Go runtime metrics: `go_goroutines`, `go_memstats_*`, `go_gc_duration_seconds`, `process_cpu_seconds_total`.

See `golang-benchmark` (investigation-session.md) for the full runtime metrics table, session setup, and profiling cost warnings.

### Diagnosing specific problems

**GC pressure:**

| PromQL | Tells you |
| --- | --- |
| `rate(go_gc_duration_seconds_count[5m])` | cycles/s; >2 sustained = excessive allocation rate |
| `rate(go_gc_duration_seconds_sum[5m]) / rate(go_gc_duration_seconds_count[5m])` | mean pause trend |
| `go_gc_duration_seconds{quantile="1"}` | worst pause — tail-latency spikes |

**Memory leaks:**

| PromQL | Tells you |
| --- | --- |
| `go_memstats_alloc_bytes` | should plateau under constant load; creeping up = leak |
| `rate(go_memstats_alloc_bytes_total[5m])` | allocation rate driving GC frequency |
| `process_resident_memory_bytes - go_memstats_sys_bytes` | non-Go memory gap (cgo, mmap); growing gap = C-side leak |

**Goroutine leaks:**

| PromQL | Tells you |
| --- | --- |
| `go_goroutines` | should track load; climbing on its own = leak |
| `delta(go_goroutines[1h])` | positive trend without more traffic |

**CPU saturation:**

| PromQL | Tells you |
| --- | --- |
| `rate(process_cpu_seconds_total[5m])` | cores consumed vs GOMAXPROCS |
| `rate(process_cpu_seconds_total[5m]) / <GOMAXPROCS>` | utilization; >0.8 sustained = saturated |

**Post-deploy regression:**

| PromQL | Tells you |
| --- | --- |
| `rate(go_memstats_alloc_bytes_total[5m])` | jumps after deploy = new allocation pattern shipped |
| `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))` | latency regression (needs an app-level histogram) |

Example rules live in [../assets/prometheus-alerts.yml](../assets/prometheus-alerts.yml) — adjust thresholds to your workload; a data pipeline has different baselines than a lightweight API. `samber/cc-skills@promql-cli` helps you test these expressions interactively.

Ready-made dashboards for these metrics are covered by `golang-observability`.

## Continuous profiling

Always-on profiling samples production processes at low overhead and stores the results for historical comparison: spot regressions across deploys, diff flamegraphs over time, and feed PGO (see [runtime.md](./runtime.md#profile-guided-optimization-pgo)).

| Tool | Model | Overhead | Best for |
| --- | --- | --- | --- |
| Grafana Pyroscope | push SDK or pull via Alloy | ~2-5% | Grafana stack, history comparison |
| Parca (Polar Signals) | eBPF pull | <1% | fleet-wide, no code changes |
| Datadog Continuous Profiler | push agent | ~1-2% | existing Datadog shops |
| Google Cloud Profiler | push agent | ~1-2% | GCP-hosted Go |

**Pyroscope push mode:**

```go
pyroscope.Start(pyroscope.Config{
    ApplicationName: "myapp",
    ServerAddress:   "http://pyroscope:4040",
    ProfileTypes: []pyroscope.ProfileType{
        pyroscope.ProfileCPU,
        pyroscope.ProfileAllocObjects,
        pyroscope.ProfileAllocSpace,
        pyroscope.ProfileInuseObjects,
        pyroscope.ProfileInuseSpace,
        pyroscope.ProfileGoroutines,
    },
})
```

**Pyroscope pull mode:** Grafana Alloy scrapes your `/debug/pprof/*` endpoints periodically — no code changes required.

Verify current API signatures in the library docs.

## Development-time visualization

| Tool | What it does |
| --- | --- |
| `github.com/arl/statsviz` | real-time browser view of heap, GC pauses, goroutines, and scheduler at `/debug/statsviz` via `statsviz.Register(mux)` |
| stdlib `expvar` | JSON metrics at `/debug/vars`; scrapable by Netdata, Telegraf, or custom dashboards |
