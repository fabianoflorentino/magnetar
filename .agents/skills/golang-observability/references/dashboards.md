# Grafana Dashboards for Go Services

These community dashboards visualise what the Prometheus Go client already exports (`go_goroutines`, `go_memstats_*`, `go_gc_duration_seconds`, `process_*`) — nothing to instrument yourself:

| Dashboard | ID | Shows |
| --- | --: | --- |
| Go Host & Runtime Metrics | 21221 | host + Go runtime in one view |
| Go Processes | 6671 | multi-process comparison across services |
| Go Metrics | 10826 | focused memory and GC deep dive |

## Install

Grafana → Dashboards → New → Import, enter the dashboard ID (e.g. `21221`), then select the Prometheus data source. With default collectors everything works out of the box.

## Which one when

- **21221** — the everyday default for a single service: goroutines, heap, GC, and threads beside host load.
- **6671** — comparing replicas or services side by side, useful during deploys to spot regressions across instances.
- **10826** — memory and GC investigation on one service while performance work is in progress.
