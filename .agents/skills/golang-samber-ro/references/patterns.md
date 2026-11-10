# Reactive Patterns

Production recipes with `samber/ro`. Each pattern is a small, composable pipeline — adjust the operators to the system at hand.

## Pattern 1: remote call with retry, timeout, fallback

One pipeline wraps all three concerns a manual loop would spread across ten lines:

```go
result := ro.Pipe3(
    fetchUser(userID), // Observable[User]
    ro.Timeout[User](5 * time.Second),
    ro.RetryWithConfig[User](ro.RetryConfig{
        Max:               3,
        Delay:             500 * time.Millisecond,
        BackoffMultiplier: 2.0,
        MaxDelay:          5 * time.Second,
    }),
    ro.Catch[User](func(err error) ro.Observable[User] {
        log.Printf("remote call failed after retries: %v, using cache", err)
        return getCachedUser(userID)
    }),
)

user, err := ro.Collect(result)
```

Order is deliberate: timeout caps the attempt, retry handles transience, catch stands in for total failure.

## Pattern 2: one hot stream, many consumers

Wrap a connection-backed source once, `Share` it, and let each consumer subscribe independently:

```go
eventStream := ro.NewObservable[TickerEvent](func(ctx context.Context, obs ro.Observer[TickerEvent]) error {
    for {
        event, err := streamSource.Read(ctx)
        if err != nil {
            return err
        }
        obs.Next(event)
    }
})

shared := ro.Pipe1(eventStream, ro.Share[TickerEvent]())

shared.Subscribe(ro.OnNext(func(e TickerEvent) { updateDashboard(e) }))
shared.Subscribe(ro.OnNext(func(e TickerEvent) { metrics.RecordTick(e.Symbol, e.Price) }))

ro.Pipe1(shared, ro.Filter(func(e TickerEvent) bool {
    return e.Price > alertThreshold
})).Subscribe(ro.OnNext(func(e TickerEvent) { sendAlert(e) }))
```

Without `Share`, every subscriber would open its own connection.

## Pattern 3: fan-in and batching

Merge independent sources, deduplicate, then batch for processing:

```go
combined := ro.Pipe2(
    ro.Merge(apiStream, pushStream, cronScheduleStream),
    ro.Distinct[Event](),
    ro.BufferWithTimeOrCount[Event](100, 5 * time.Second),
    ro.Map(func(batch []Event) ProcessResult { return processBatch(batch) }),
)
```

Pick the combiner by intent:

| Operator | Behavior | Fit |
| --- | --- | --- |
| `Merge` | Interleave as sources emit | Independent streams, order irrelevant |
| `Concat` | Finish one source before the next | Ordered processing, fallback chains |
| `Zip` | Pair one value from each source | Correlated data |
| `CombineLatest` | Re-emit the latest tuple on any change | Dependent state |

## Pattern 4: combining dependent async data

Fetch independent pieces in parallel and merge once both resolve:

```go
profile := ro.Pipe1(
    ro.CombineLatest2(fetchUser(userID), fetchOrders(userID)),
    ro.Map(func(pair lo.Tuple2[User, []Order]) UserProfile {
        return UserProfile{User: pair.A, Orders: pair.B}
    }),
)
```

When exactly one value from each source suffices, `Zip2` waits for both instead.

## Pattern 5: running aggregation

`Scan` maintains live state for dashboards; `Reduce` is the batch-only counterpart:

```go
statsStream := ro.Pipe2(
    metricsStream,
    ro.Scan(func(acc Stats, v float64) Stats {
        acc.Count++
        acc.Sum += v
        acc.Avg = acc.Sum / float64(acc.Count)
        if v > acc.Max {
            acc.Max = v
        }
        return acc
    }, Stats{}),
    ro.SampleTime[Stats](5 * time.Second),
)
```

Every input updates `Stats`; sampling throttles how often the dashboard sees it.

## Pattern 6: layered error recovery

Stack strategies so each failure lands at the right depth:

```go
resilient := ro.Pipe3(
    primaryDataSource,
    ro.RetryWithConfig[Data](ro.RetryConfig{Max: 2, Delay: time.Second}),
    ro.Catch[Data](func(err error) ro.Observable[Data] {
        log.Warn("primary failed, trying secondary", "err", err)
        return secondaryDataSource
    }),
    ro.OnErrorReturn[Data](cachedDefault),
)
```

Retry transient errors first, fall back on persistent ones, then surrender to a default.

## Pattern 7: debounced file watcher

Reactive config reload with throttling:

```go
import rofsnotify "github.com/samber/ro/plugins/fsnotify"

watcher := ro.Pipe3(
    rofsnotify.Watch("/etc/app/config/"),
    ro.Filter(func(e fsnotify.Event) bool {
        return e.Op&fsnotify.Write != 0
    }),
    ro.ThrottleTime[fsnotify.Event](2 * time.Second),
    ro.Map(func(e fsnotify.Event) Config { return reloadConfig(e.Name) }),
)

watcher.Subscribe(ro.NewObserver(
    func(cfg Config) { applyConfig(cfg) },
    func(err error) { log.Error("config watch failed", "err", err) },
    func() { log.Info("config watcher stopped") },
))
```

## Pattern 8: graceful shutdown

Terminate infinite streams with a signal observable or context cancellation:

```go
import rosignal "github.com/samber/ro/plugins/signal"

shutdown := rosignal.Notify(syscall.SIGTERM, syscall.SIGINT)
sub := ro.Pipe1(
    workStream,
    ro.TakeUntil[Work, os.Signal](shutdown),
).Subscribe(worker)
sub.Wait()

// Context alternative
ctx, cancel := context.WithCancel(context.Background())
ro.Pipe2(
    workStream,
    ro.ContextReset[Work](ctx),
    ro.ThrowOnContextCancel[Work](),
).Subscribe(worker)
```

## Pattern 9: observable production pipeline

Observability at every stage, errors counted and retried:

```go
import roslog "github.com/samber/ro/plugins/observability/slog"

pipeline := ro.Pipe5(
    eventSource,
    ro.TapOnSubscribe[Event](func() { slog.Info("pipeline started") }),
    ro.Filter(func(e Event) bool { return e.Valid() }),
    roslog.TapOnNext[Event](logger, slog.LevelDebug),
    ro.Map(enrichEvent),
    ro.BufferWithTimeOrCount[EnrichedEvent](50, 10 * time.Second),
    ro.MapErr(func(batch []EnrichedEvent) (Result, error) {
        return persistBatch(batch)
    }),
    ro.TapOnError[Result](func(err error) {
        slog.Error("pipeline error", "err", err)
        metrics.IncrCounter("pipeline.errors", 1)
    }),
    ro.RetryWithConfig[Result](ro.RetryConfig{Max: 3, Delay: time.Second}),
)
```