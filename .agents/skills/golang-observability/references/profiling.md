# Profiling and Continuous Profiling

Metrics say the service is slow; a profile says *which function, on which line*. On-demand pprof answers a live question; continuous profiling answers it after the fact. See `golang-troubleshooting` (pprof.md) for the on-demand workflow.

## On-demand pprof

- pprof endpoints leak runtime internals and invite DoS — **secure them** (basic auth or internal network only), see `golang-security`.
- Beyond the standard profiles, expose `/debug/pprof/goroutineleak` (generally available since Go 1.27): it reports goroutines blocked on unreachable concurrency primitives, a signal the classic goroutine profile doesn't surface directly.
- Since Go 1.27, tracebacks for `go 1.27+` modules carry goroutine labels by default; if a label could leak sensitive request data into crash logs, disable with `GODEBUG=tracebacklabels=0`.

## Continuous profiling with Pyroscope

On-demand profiling requires being present at the incident. Continuous profiling samples always-on at ~2-5% CPU, best gated behind an environment variable:

```go
func setupContinuousProfiling() {
    if os.Getenv("PROFILING_ENABLED") != "true" {
        return
    }

    _, err := pyroscope.Start(pyroscope.Config{
        ApplicationName: "my-service",
        ServerAddress:   os.Getenv("PYROSCOPE_URL"),
        ProfileTypes: []pyroscope.ProfileType{
            pyroscope.ProfileCPU,
            pyroscope.ProfileAllocObjects,
            pyroscope.ProfileInuseObjects,
            pyroscope.ProfileGoroutines,
            pyroscope.ProfileMutexDuration,
            pyroscope.ProfileBlockDuration,
        },
    })
    if err != nil {
        slog.Error("failed to start pyroscope", "error", err)
        return
    }
    slog.Info("continuous profiling enabled", "url", os.Getenv("PYROSCOPE_URL"))
}
```

## Cost of continuous profiling

The overhead is per-instance and always-on:

- **CPU** — stack sampling competes with the workload, even at 2-5%.
- **Network/storage** — profiles ship to the backend continuously; cost multiplies with replica count.
- **Profile types** — each additional type (mutex, block, goroutine) adds incremental overhead.

Mitigate: enable via env variable on a subset of replicas (e.g. 1 in 10), start with CPU + heap only, and add mutex/block/goroutine profiles while investigating a specific issue.

## When to profile

| Observation | Profile |
| --- | --- |
| CPU or memory usage high | CPU + heap |
| P99 latency spikes | CPU + mutex (contention) |
| Goroutine count climbing | goroutine (leaks) |
| Before/after an optimization | compare profiles |

## Security

Never expose pprof endpoints publicly; protect them the same way as any admin surface (`golang-security`).
