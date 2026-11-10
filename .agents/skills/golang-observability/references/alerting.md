# Alerting

Golden signals and burn-rate patterns come from [metrics.md](./metrics.md); this file is about what to alert on and how not to write the rules.

## The four golden signals

Cover all four (Google SRE provenance):

| Signal | Measures | Example | Triggers |
| --- | --- | --- | --- |
| Latency | time to serve a request | request duration histogram | P99 > 2s for 5m |
| Traffic | demand on the system | requests counter | zero requests for 10m |
| Errors | rate of failed requests | 5xx counter | error ratio > 1% for 5m |
| Saturation | how full the system is | active / max connections | pool > 90% for 5m |

## Start from awesome-prometheus-alerts

[awesome-prometheus-alerts](https://samber.github.io/awesome-prometheus-alerts/) curates ~500 ready-to-use Prometheus rules — treat them as templates, not gospel:

| Category | Rules |
| --- | --: |
| Basic resource monitoring | ~107 |
| Databases and brokers | ~233 |
| Reverse proxies / load balancers | ~45 |
| Runtimes | ~4 |
| Orchestrators | ~74 |
| Network, security, storage | ~40 |

Workflow when adding an infrastructure dependency (database, cache, broker, proxy):

1. Find the technology's section and copy the relevant rule.
2. Adapt thresholds (`> 0.01`, `> 100`, ...) and `for:` durations to your SLO and traffic shape.
3. Make sure the exporter is actually deployed (e.g. `postgres_exporter`, `redis_exporter`) — the rules depend on its metrics.
4. Land the rules in `prometheus/rules/` and reference the file from `rule_files`.

Example for PostgreSQL:

```yaml
groups:
  - name: postgresql
    rules:
      - alert: PostgresqlDown
        expr: pg_up == 0
        for: 1m
        labels:
          severity: critical
      - alert: PostgresqlTooManyConnections
        expr: sum by (instance) (pg_stat_activity_count{datname!~"template.*|postgres"}) > pg_settings_max_connections * 0.8
        for: 2m
        labels:
          severity: warning
```

## Go runtime alerts

The Prometheus Go client exports runtime metrics for free. These four catch leaks and GC pressure before users feel them:

```yaml
groups:
  - name: go-runtime
    rules:
      # goroutine leak — steady growth under flat load
      - alert: GoroutineLeak
        expr: go_goroutines > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High goroutine count ({{ $labels.instance }})"

      # worst-case GC pause destroys tail latency
      - alert: HighGCDuration
        expr: go_gc_duration_seconds{quantile="1"} > 0.1
        for: 5m
        labels:
          severity: warning

      # heap swallowing most of the RSS — possible leak
      - alert: HighMemoryUsage
        expr: go_memstats_alloc_bytes / go_memstats_sys_bytes > 0.9
        for: 5m
        labels:
          severity: critical

      # blocking syscalls / cgo spinning up threads
      - alert: HighThreadCount
        expr: go_threads > 500
        for: 5m
        labels:
          severity: warning
```

## Severity and `for:`

| Severity | Action | `for:` | Example |
| --- | --- | --- | --- |
| `critical` | page the on-call | 2-5m | service down, error >5%, data-loss risk |
| `warning` | open a ticket | 10-30m | P99 high, pool >80%, goroutine creep |

`for:` is the seasoning: too short and you page on blips, too long and incidents go unnoticed. Only binary aliveness checks (`pg_up == 0`) may use `for: 0m`.

## Common mistakes

```yaml
# Bad — irate flutters on a single scrape; one slow request pages
expr: irate(http_requests_total{status=~"5.."}[5m]) > 0.01

# Good — rate smooths the window and for: confirms it's sustained
expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.01
for: 5m
```

```yaml
# Bad — percentile without for: fires on one scrape
expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 2

# Good — sustained for 5 minutes
expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 2
for: 5m
```

```yaml
# Bad — raw gauge without trend analysis, flaps constantly
expr: myapp_queue_messages_pending > 1000

# Good — trend-aware threshold
expr: myapp_queue_messages_pending > 1000
for: 10m
```
