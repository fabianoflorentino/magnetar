# Prometheus Go Runtime Metrics Reference

What `prometheus/client_golang` actually exposes as Prometheus series about the Go runtime.

Start with one clarification: `runtime/metrics` is *not* a Prometheus format — it is the Go runtime's own metrics API. The client library converts a subset of it into Prometheus names. By default it exposes the traditional `go_memstats_*` and `go_gc_*` series to keep cardinality low; the runtime/metrics-backed series are opt-in (see below). This document lists only what lands on the scrape endpoint.

Verify against the library's docs for the exact collector in use, and note that the underlying `runtime/metrics` key set changes with Go versions.

## Series with labels

| Metric | Label | Values |
| --- | --- | --- |
| `go_gc_duration_seconds` | `quantile` | 0, 0.25, 0.5, 0.75, 1 |
| `go_info` | `version` | e.g. `go1.21.3` |

Every other series here is label-free.

## Default Go series (always exposed)

**Allocations and heap**

| Metric | Type | Meaning |
| --- | --- | --- |
| `go_memstats_alloc_bytes` | gauge | Heap bytes currently allocated |
| `go_memstats_alloc_bytes_total` | counter | Cumulative bytes allocated |
| `go_memstats_sys_bytes` | gauge | Total bytes drawn from the OS |
| `go_memstats_heap_alloc_bytes` | gauge | Allocated heap bytes |
| `go_memstats_heap_idle_bytes` | gauge | Idle heap bytes |
| `go_memstats_heap_inuse_bytes` | gauge | Heap bytes in use |
| `go_memstats_heap_objects` | gauge | Live heap object count |
| `go_memstats_heap_released_bytes` | gauge | Heap bytes returned to the OS |
| `go_memstats_heap_sys_bytes` | gauge | Heap bytes reserved from the OS |
| `go_memstats_stack_inuse_bytes` | gauge | Stack bytes in use |
| `go_memstats_stack_sys_bytes` | gauge | Stack bytes reserved |
| `go_memstats_mspan_inuse_bytes` | gauge | mspan bytes in use |
| `go_memstats_mspan_sys_bytes` | gauge | mspan bytes reserved |
| `go_memstats_mcache_inuse_bytes` | gauge | mcache bytes in use |
| `go_memstats_mcache_sys_bytes` | gauge | mcache bytes reserved |
| `go_memstats_other_sys_bytes` | gauge | Other runtime-allocated bytes |
| `go_memstats_gc_sys_bytes` | gauge | Bytes used for GC bookkeeping |
| `go_memstats_buck_hash_sys_bytes` | gauge | Profiling-bucket hash table bytes |
| `go_memstats_mallocs_total` | counter | Total allocation events |
| `go_memstats_frees_total` | counter | Total free events |

**GC configuration and timing**

| Metric | Type | Meaning |
| --- | --- | --- |
| `go_gc_gogc_percent` | gauge | Configured GOGC target |
| `go_gc_gomemlimit_bytes` | gauge | GOMEMLIMIT soft limit |
| `go_memstats_last_gc_time_seconds` | gauge | Unix time of the last GC end |
| `go_memstats_next_gc_bytes` | gauge | Heap size that triggers the next GC |
| `go_gc_duration_seconds` | summary | Pause durations at quantiles 0/0.25/0.5/0.75/1 |
| `go_gc_duration_seconds_count` | counter | Number of GC pauses |
| `go_gc_duration_seconds_sum` | counter | Total paused time |

**Runtime state and version**

| Metric | Type | Meaning |
| --- | --- | --- |
| `go_goroutines` | gauge | Current goroutine count |
| `go_threads` | gauge | Current OS thread count |
| `go_sched_gomaxprocs_threads` | gauge | Current GOMAXPROCS |
| `go_info` | gauge | Go version string (label) |

## Opt-in runtime/metrics series

Register the runtime/metrics-backed collector to expose everything below:

```go
prometheus.NewRegistry().MustRegister(
    collectors.NewGoCollector(
        collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
    ),
)
```

**GC accounting**

| Metric | Type | Meaning |
| --- | --- | --- |
| `go_gc_cycles_automatic_gc_cycles_total` | counter | GC cycles triggered by heap growth |
| `go_gc_cycles_forced_gc_cycles_total` | counter | GC cycles forced via `runtime.GC()` |
| `go_gc_heap_allocs_bytes_total` | counter | Cumulative allocation bytes |
| `go_gc_heap_allocs_objects_total` | counter | Cumulative allocation count |
| `go_gc_heap_frees_bytes_total` | counter | Cumulative freed bytes |
| `go_gc_heap_frees_objects_total` | counter | Cumulative freed count |
| `go_gc_heap_goal_bytes` | gauge | Heap goal for the next GC |
| `go_gc_heap_live_bytes` | gauge | Currently live heap bytes |
| `go_gc_heap_objects_objects` | gauge | Total heap objects |
| `go_gc_pauses_seconds` | histogram | GC pause duration distribution |

**CPU time by class**

| Metric | Meaning |
| --- | --- |
| `go_cpu_classes_gc_mark_assist_cpu_seconds_total` | GC mark-assist time |
| `go_cpu_classes_gc_mark_dedicated_cpu_seconds_total` | Dedicated mark workers |
| `go_cpu_classes_gc_mark_idle_cpu_seconds_total` | Idle mark workers |
| `go_cpu_classes_gc_pause_cpu_seconds_total` | GC pause CPU time |
| `go_cpu_classes_gc_total_cpu_seconds_total` | Total GC CPU time |
| `go_cpu_classes_idle_cpu_seconds_total` | Idle CPU time |
| `go_cpu_classes_scavenge_assist_cpu_seconds_total` | Scavenger assist |
| `go_cpu_classes_scavenge_background_cpu_seconds_total` | Background scavenger |
| `go_cpu_classes_scavenge_total_cpu_seconds_total` | Total scavenger CPU |
| `go_cpu_classes_total_cpu_seconds_total` | Total CPU, all classes |
| `go_cpu_classes_user_cpu_seconds_total` | User-mode CPU time |

All counters with `_cpu_seconds_total` suffix.

**Memory classes**

| Metric | Meaning |
| --- | --- |
| `go_memory_classes_heap_free_bytes` | Free heap |
| `go_memory_classes_heap_objects_bytes` | Live objects |
| `go_memory_classes_heap_released_bytes` | Released to OS |
| `go_memory_classes_heap_stacks_bytes` | Stack memory |
| `go_memory_classes_heap_unused_bytes` | Unused heap |
| `go_memory_classes_metadata_mcache_free_bytes` | Free mcache |
| `go_memory_classes_metadata_mcache_inuse_bytes` | In-use mcache |
| `go_memory_classes_metadata_mspan_free_bytes` | Free mspan |
| `go_memory_classes_metadata_mspan_inuse_bytes` | In-use mspan |
| `go_memory_classes_other_bytes` | Other runtime memory |
| `go_memory_classes_total_bytes` | Total memory in the heap arena |

All gauges.

**Scheduler**

| Metric | Type | Meaning |
| --- | --- | --- |
| `go_sched_goroutines_running_goroutines` | gauge | Currently executing |
| `go_sched_goroutines_runnable_goroutines` | gauge | Runnable, waiting for a P |
| `go_sched_goroutines_goroutines` | gauge | Total goroutines |
| `go_sched_goroutines_created_goroutines_total` | counter | Goroutines ever created |
| `go_sched_goroutines_waiting_goroutines` | gauge | Waiting, not runnable |
| `go_sched_latencies_seconds` | histogram | Scheduling-latency distribution |
| `go_sched_pauses_stopping_gc_seconds` | histogram | STW for GC stop |
| `go_sched_pauses_stopping_other_seconds` | histogram | STW for other stops |
| `go_sched_pauses_total_gc_seconds` | histogram | Total GC pauses |
| `go_sched_pauses_total_other_seconds` | histogram | Total other pauses |
| `go_sched_threads_total_threads` | counter | OS threads ever created |
| `go_sync_mutex_wait_total_seconds_total` | counter | Time spent waiting on mutexes |

**cgo**

| Metric | Type | Meaning |
| --- | --- | --- |
| `go_cgo_go_to_c_calls_total` | counter | Calls from Go into C |

## Process series

Not Go-specific; provided by the standard process collector.

| Metric | Type | Meaning |
| --- | --- | --- |
| `process_cpu_seconds_total` | counter | Total CPU time, user + system |
| `process_resident_memory_bytes` | gauge | RSS |
| `process_virtual_memory_bytes` | gauge | Virtual memory |
| `process_virtual_memory_max_bytes` | gauge | Virtual-memory ceiling |
| `process_open_fds` | gauge | Open file descriptors |
| `process_max_fds` | gauge | FD limit |
| `process_start_time_seconds` | gauge | Process start (Unix time) |
| `process_page_faults_total` | counter | All page faults |
| `process_page_faults_minor_total` | counter | Minor faults |
| `process_page_faults_major_total` | counter | Major faults |

## Common PromQL

**Memory leaks**

```promql
go_memstats_alloc_bytes
go_gc_heap_live_bytes
rate(go_memstats_alloc_bytes_total[5m])
```

**GC pressure**

```promql
go_gc_duration_seconds{quantile="1"}
rate(go_gc_duration_seconds_sum[5m]) / rate(go_gc_duration_seconds_count[5m])
rate(go_gc_duration_seconds_count[5m])
```

**Goroutine leaks**

```promql
go_goroutines
delta(go_goroutines[1h])
```

**CPU usage**

```promql
rate(process_cpu_seconds_total[5m])
rate(process_cpu_seconds_total[5m]) / <GOMAXPROCS>
```

**File-descriptor leaks**

```promql
delta(process_open_fds[1h])
process_open_fds / process_max_fds
```

For executing these against a live Prometheus from the CLI, see the `samber/cc-skills@promql-cli` skill.

## References

- [prometheus/client_golang collectors](https://github.com/prometheus/client_golang/tree/main/prometheus/collectors)
- [Go runtime/metrics package](https://pkg.go.dev/runtime/metrics)