---
name: golang-observability
description: "Golang everyday observability — the always-on signals in production. Covers structured logging with slog, Prometheus metrics, OpenTelemetry distributed tracing, continuous profiling with pprof/Pyroscope, server-side RUM event tracking, alerting, and Grafana dashboards. Apply when instrumenting Go services for production monitoring, setting up metrics or alerting, adding OpenTelemetry tracing, correlating logs with traces, migrating legacy loggers (zap/logrus/zerolog) to slog, adding observability to new features, or implementing GDPR/CCPA-compliant tracking with Customer Data Platforms (CDP). Not for temporary deep-dive performance investigation (→ See `fabianoflorentino/golang-agent-skills@golang-benchmark` and `fabianoflorentino/golang-agent-skills@golang-performance` skills)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📡"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch WebSearch AskUserQuestion
paths:
  - "**/*.go"
---

# Observability in Go

**Persona:** You are a Go observability engineer. An unobserved production system is a liability — instrument proactively, correlate signals when diagnosing, and never call a feature done until it is observable.

**Modes:**

- **Instrument** (default) — add observability to new or existing code: declare metrics, add spans, structured logging, pprof toggles. Sequential, following the guide order.
- **Review** — review a PR's instrumentation: metrics declared, spans opened/closed, log fields consistent. Sequential.
- **Audit** — sweep observability coverage. Fan out one sub-agent per signal (metrics, logging, tracing, profiling, RUM) and merge findings. Parallel.

**When to use:** any Go service work needing production signals, alerting, dashboards, or tracing. For deep-dive performance investigation use `golang-benchmark`; for optimization patterns `golang-performance`; for protecting pprof and log PII see `golang-security`.

## The five signals

| Signal | Answers | Tool | Reach |
| --- | --- | --- | --- |
| **Logs** | What happened, for a given request/deploy | `log/slog` | Discrete events, errors, audits |
| **Metrics** | How much, how fast, right now | Prometheus client | Aggregates, alerts, SLOs |
| **Traces** | Where did the time go in a request | OpenTelemetry | Flow across services |
| **Profiles** | Why is this slow / eating memory | pprof, Pyroscope | Hotspots, leaks, contention |
| **RUM** | How do users actually experience it | PostHog, Segment | Funnels, session replay |

Logs say *what*; metrics say *how much*; traces say *where in the flow*; profiles say *why deep down*; RUM says *what users feel*.

## Signal naming and design decisions

**Metrics — Counter, Gauge, or Histogram.**

- Counter = rate of change (requests, errors). Monotonic; human value in `rate()`, not the raw number.
- Gauge = snapshot of a level (queue depth, in-flight, heap).
- Histogram = latency/payload distribution; never a Summary.

Prometheus **Histograms** win over **Summaries** because the server aggregates buckets across instances, so `histogram_quantile(0.99, rate(...))` works globally. A Summary's quantiles are precomputed per-process and cannot be aggregated — a P99 on an orphaned pod hides behind the fleet average. Every HTTP endpoint must export latency + error rate.

**Cardinality is the top constraint.** Label values must be bounded: route patterns yes, full URLs and user IDs no. One unbounded label (e.g. a request ID) multiplies series per request and can burn out a small Prometheus in hours. Keep the label set enumerable and small; put the request ID in logs, not in metrics.

**Traces — sample; you cannot collect everything.** Always-on tracing across a fleet is financially untenable. Sample by route and error (`otelhttp` sampling, `picobackend`), record every error span, and keep the happy path sampled at a rate your budget survives. Propagate context everywhere — `context` is the vehicle for `trace_id`/`span_id`/deadlines across boundaries.

**Profiles — on-demand pprof for reads, continuous (Pyroscope) for the always-on signal.** Toggle pprof via env without redeploy; secure the endpoint (`golang-security`).

**RUM — server-side events, consent-checked.** Track key business events from the backend (PostHog/Segment) with identity key `user_id`, never email; check GDPR/CCPA consent before tracking. See the RUM guide for user-deletion endpoints.

## The frontmatter of a good signal setup

First decide *who consumes what* — these four lines drive every choice downstream:

1. **Latency histograms per endpoint**, buckets tuned to the SLO; and error counters.
2. **`rate()`-based alerting** with a `for:` duration (dashboards with `irate` flap).
3. **One `trace_id` on every log line** and every exemplar.
4. **Profiles on demand** and dashboards covering the golden signals.

## Correlating signals

Signals are strongest connected:

```go
import "go.opentelemetry.io/contrib/bridges/otelslog"
logger := otelslog.NewHandler("my-service")   // injects trace_id/span_id
slog.SetDefault(slog.New(logger))
slog.InfoContext(ctx, "order created", "order_id", id)
// {"trace_id":"abc123","span_id":"def456","msg":"order created",...}
```

```go
obs := histogram.WithLabelValues("POST", "/orders")
if eo, ok := obs.(prometheus.ExemplarObserver); ok {
    eo.ObserveWithExemplar(duration, prometheus.Labels{"trace_id": traceID})
} else {
    obs.Observe(duration)
}
```

## Migrating legacy loggers

`log/slog` is stdlib since 1.21 and the ecosystem consolidated on it; ship with it, not with a third-party logger. Migration path:

1. `slog.SetDefault(slog.New(...))` next to the old logger.
2. Route slog through bridge handlers (`samber/slog-zap`, `sbank/slog-logrus`, `samber/slog-zerolog`) so output lands in the existing pipeline.
3. Replace call sites `zap.L().Info(...)` → `slog.Info(...)` incrementally.
4. Delete the bridge and the dependency once the last call is migrated.

For multi-handler fan-out, prefer stdlib `slog.NewMultiHandler` (Go 1.26+) before third-party composition.

## Definition of done

- [ ] **Metrics declared** — counter for ops/errors, histogram per endpoint, gauge for saturation; PromQL and alert hints as comments above each `var`.
- [ ] **Logs proper** — structured key/values, `*Context` variants, no PII, log-or-return (never both, `golang-error-handling`).
- [ ] **Spans on every meaningful op** — service methods, DB queries, external calls; `span.RecordError`.
- [ ] **Dashboards and alerts exist** — the PromQL in the comments is wired into Grafana + Prometheus alerting (golden signals; `awesome-prometheus-alerts` for thousands of ready-to-use rules).
- [ ] **RUM events tracked** server-side, `user_id` as identity, consent checked.

## Common mistakes

```go
if err != nil {
    slog.Error("query failed", "error", err)  // log AND return -> duplicate
    return fmt.Errorf("query: %w", err)
}
```

```go
httpRequests.WithLabelValues(r.Method, r.URL.Path, userID).Inc() // high-cardinality label
```

```go
db.Query("SELECT ...")          // no ctx -> trace chain breaks
db.QueryContext(ctx, "SELECT ...")
```

```go
prometheus.NewSummary(...)      // can't aggregate across instances
prometheus.NewHistogram(...)
```

## Cross-references

- `golang-error-handling` — the single-handling rule that underpins clean logs.
- `golang-troubleshooting` — using these signals to diagnose production incidents.
- `golang-security` — protecting pprof endpoints, PII in logs.
- `golang-context` — propagating trace context across boundaries.
- `golang-benchmark` — pprof deep dives; `golang-performance` — the fixes that follow.
- `golang-rest` — request HTTP middleware conventions.

## Reference files

- [`references/logging.md`](./references/logging.md) — slog levels, context, middleware, migration.
- [`references/metrics.md`](./references/metrics.md) — the four types, naming, PromQL conventions, high-cardinality.
- [`references/tracing.md`](./references/tracing.md) — OpenTelemetry spans, otelhttp, sampling, cost.
- [`references/profiling.md`](./references/profiling.md) — pprof on demand vs Pyroscope continuous.
- [`references/rum.md`](./references/rum.md) — event tracking, CDP, GDPR/CCPA consent.
- [`references/alerting.md`](./references/alerting.md) — golden signals, Go runtime alerts, common breakages.
- [`references/dashboards.md`](./references/dashboards.md) — prebuilt Go dashboards and customization.
