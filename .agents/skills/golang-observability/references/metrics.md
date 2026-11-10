# Metrics with Prometheus

`github.com/prometheus/client_golang` is the standard Go client; for current API signatures refer to the library docs. Metrics feed the dashboards in [dashboards.md](./dashboards.md) and the rules in [alerting.md](./alerting.md); `golang-troubleshooting` covers reading metrics during incidents, and `samber/cc-skills@promql-cli` runs PromQL interactively from the CLI.

## Metric types

| Type | Semantics | Example | Use for |
| --- | --- | --- | --- |
| `Counter` | monotonic total | requests, errors | event counts and rates |
| `Gauge` | current level | in-flight, queue depth | state snapshots |
| `Histogram` | bucketed observations | latency, sizes | distributions, percentiles |
| `Summary` | client-side quantiles | latency | rarely — see below |

## Histogram vs Summary

- **Histogram** keeps bucket counts server-side; quantiles are computed at query time with `histogram_quantile()`. Because the data lives in Prometheus, histograms **aggregate across instances** — the fleet-wide P99 is meaningful.
- **Summary** precomputes quantiles in the process and ships fixed values. Those values **cannot be aggregated** — at 10 replicas there is no honest fleet P99, and an orphaned pod's percentile hides behind the average.

Use Histograms. Reach for a Summary only when a single instance needs exact quantiles and nothing else will combine them.

## Naming

Names follow `<namespace>_<subsystem>_<name>_<unit>`, one unit and one quantity per metric, units in the base form:

| Measurement | Use | Not |
| --- | --- | --- |
| Time | `_seconds` | `_milliseconds`, `_minutes` |
| Size | `_bytes` | `_kilobytes`, `_megabytes` |
| Ratio | `_ratio` (0-1) | `_percent` |
| Temperature | `_celsius` | `_fahrenheit` |

Suffix conventions: `_total` is mandatory for counters; `_seconds` for durations; `_bytes` for sizes; `_info` for metadata pseudo-metrics; `_created` for counter creation time.

- **No embedded labels** in the name — `myapp_http_requests_total{method="GET"}` not `myapp_http_get_requests_total`.
- `sum()`/`avg()` across a metric's dimensions should stay meaningful; if not, split it.

```go
// good
myapp_http_requests_total
myapp_http_request_duration_seconds
myapp_http_response_size_bytes
myapp_db_connections_active

// bad
request_count            // no namespace, no unit
httpDuration             // camelCase
request_duration_ms       // derived unit
```

## Exposing

```go
mux.Handle("/metrics", promhttp.Handler())
```

## Self-documenting declarations

Comment the PromQL right above each declaration so the queries are reviewed alongside the metric and stay in sync with renames:

```go
// Dashboard: rate(myapp_http_requests_total[5m])
// Alert:     sum(rate(myapp_http_requests_total{status=~"5.."}[5m])) / sum(rate(myapp_http_requests_total[5m])) > 0.01
var httpRequestsTotal = promauto.NewCounterVec(...)
```

For infrastructure and dependency alerting, [awesome-prometheus-alerts](https://samber.github.io/awesome-prometheus-alerts/) curates ~500 ready rules by technology — see [alerting.md](./alerting.md).

Never use `irate` in alerts; `rate` over a window keeps them stable.

## Counters

```go
// Dashboard: rate(myapp_http_requests_total[5m])
// Dashboard: sum by (status) (rate(myapp_http_requests_total[5m]))
// Alert:     sum(rate(myapp_http_requests_total{status=~"5.."}[5m])) / sum(rate(myapp_http_requests_total[5m])) > 0.01 (for: 5m)
var httpRequestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "myapp",
        Subsystem: "http",
        Name:      "requests_total",
        Help:      "Total number of HTTP requests.",
    },
    []string{"method", "path", "status"},
)

// Alert:     rate(myapp_errors_total{type="database"}[5m]) > 0.5
var errorsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "myapp",
        Name:      "errors_total",
        Help:      "Total number of errors by type.",
    },
    []string{"type"}, // "database", "external_api", "validation"
)
```

Workhorse queries:

```promql
# requests/s, distribution by status class, busiest endpoints
rate(myapp_http_requests_total[5m])
sum by (status) (rate(myapp_http_requests_total[5m]))
topk(5, sum by (path) (rate(myapp_http_requests_total[5m])))

# error ratio — the availability SLI
sum(rate(myapp_http_requests_total{status=~"5.."}[5m]))
  / sum(rate(myapp_http_requests_total[5m]))

# business check — no orders created in 30 minutes
rate(myapp_orders_created_total[30m]) == 0
```

## Gauges

```go
// Dashboard: myapp_http_in_flight_requests
// Alert:     myapp_http_in_flight_requests > 500 (for: 5m)
var httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
    Namespace: "myapp",
    Subsystem: "http",
    Name:      "in_flight_requests",
    Help:      "HTTP requests currently being processed.",
})

// usage in middleware
httpInFlight.Inc()
defer httpInFlight.Dec()
```

```go
// Dashboard: myapp_db_connections_active / myapp_db_connections_max
// Alert:     myapp_db_connections_active{pool="write"} / myapp_db_connections_max{pool="write"} > 0.9 (for: 5m)
var dbConnectionsActive = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Namespace: "myapp",
        Subsystem: "db",
        Name:      "connections_active",
        Help:      "Number of active database connections.",
    },
    []string{"pool"}, // "read", "write"
)
```

Gauge queries:

```promql
# current level, and saturation as a ratio
myapp_http_in_flight_requests
myapp_db_connections_active{pool="write"} / myapp_db_connections_max{pool="write"}

# queue growth rate, and whether the pool/queue saturates soon
deriv(myapp_queue_messages_pending[5m])
predict_linear(myapp_db_connections_active[15m], 600) > myapp_db_connections_max
predict_linear(myapp_queue_messages_pending[30m], 3600) > 10000
```

## Histograms

```go
// Dashboard: histogram_quantile(0.99, sum(rate(myapp_http_request_duration_seconds_bucket[5m])) by (le))
// Alert:     histogram_quantile(0.99, sum(rate(myapp_http_request_duration_seconds_bucket[5m])) by (le)) > 2 (for: 5m)
var httpDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Namespace: "myapp",
        Subsystem: "http",
        Name:      "request_duration_seconds",
        Help:      "HTTP request duration in seconds.",
        Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
    },
    []string{"method", "path", "status"},
)
```

Observe in middleware with `WithLabelValues(...).Observe(time.Since(start).Seconds())`. `prometheus.DefBuckets` (`.005` → `10`) is a sane default; a slow external call needs a wider tail (e.g. up to 30s).

```promql
# p50 / p90 / p99 / p99.9, plus p99 split by endpoint
histogram_quantile(0.50, sum(rate(myapp_http_request_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.99, sum(rate(myapp_http_request_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.99, sum(rate(myapp_http_request_duration_seconds_bucket[5m])) by (le, path))

# mean latency, and Apdex-style share under 300ms
sum(rate(myapp_http_request_duration_seconds_sum[5m]))
  / sum(rate(myapp_http_request_duration_seconds_count[5m]))
sum(rate(myapp_http_request_duration_seconds_bucket{le="0.3"}[5m]))
  / sum(rate(myapp_http_request_duration_seconds_count[5m]))

# request throughput straight from the histogram
sum(rate(myapp_http_request_duration_seconds_count[5m]))
```

## Summary — sparingly

Client-side quantiles that cannot be aggregated — single-process diagnostics only:

```go
var jobDuration = promauto.NewSummary(prometheus.SummaryOpts{
    Namespace:  "myapp",
    Subsystem:  "jobs",
    Name:       "processing_seconds",
    Help:       "Job processing duration in seconds.",
    Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
    MaxAge:     10 * time.Minute,
})
```

## Multi-window burn-rate alerting

Threshold alerts ("error > 1%") fire too late on fast incidents and too early on slow ones. Burn-rate window pairs scale urgency to how fast the error budget burns. For a 99.9% SLO (0.1% budget over 30 days):

| Windows | Burn rate | Error rate | Severity | Meaning |
| --- | --- | --- | --- | --- |
| 5m + 1h | 14.4x | > 1.44% | page | budget gone in ~2 days |
| 30m + 6h | 6x | > 0.6% | page | budget gone in ~5 days |
| 2h + 24h | 1x | > 0.1% | ticket | on track to exhaust budget |

```promql
# fast burn — page (for: 2m); both windows must agree so blips don't page
(
  (1 - sum(rate(myapp_http_requests_total{status=~"2.."}[5m])) / sum(rate(myapp_http_requests_total[5m]))) > 0.0144
  and
  (1 - sum(rate(myapp_http_requests_total{status=~"2.."}[1h])) / sum(rate(myapp_http_requests_total[1h]))) > 0.0144
)
```

The short window catches the incident fast; the long window proves it is sustained.

## High cardinality

Every unique label combination is a separate time series. Unbounded labels — user IDs, full URLs, request IDs — multiply series per request and can burn out a small Prometheus in hours.

```go
// never: a series per path, per user, or per request
httpRequestsTotal.WithLabelValues(r.URL.Path)       // /users/alice, /users/bob, ...
httpRequestsTotal.WithLabelValues(userID)
httpRequestsTotal.WithLabelValues(r.Header.Get("X-Request-ID"))

// yes: bounded, normalized labels
httpRequestsTotal.WithLabelValues(routePattern)     // /users/{id}
httpRequestsTotal.WithLabelValues(r.Method)         // 4 values
httpRequestsTotal.WithLabelValues(statusClass)      // "2xx", "4xx", "5xx"
```

Controls: route templates not paths; status classes not codes; never user/request/session IDs or emails. High-cardinality context belongs in trace attributes instead. Rule of thumb: if a label can exceed ~100 values, split it or drop it.
