# Operators Guide

150+ operators grouped by role. All are generic `func(...) Observable[R]` pipeline stages intended for `Pipe` composition.

## Building pipelines

`Pipe1` … `Pipe25` check the chain at compile time; the bare `Pipe` accepts `any` and drops that safety. Curried operator composition reuses a transform across pipelines. `Collect` blocks and returns everything as a slice.

```go
result := ro.Pipe2(source, op1, op2)

transform := ro.PipeOp2(op1, op2) // func(Observable[A]) Observable[C]
result := transform(source)

values, err := ro.Collect(observable)
```

## Creation

| Operator | Emits |
| --- | --- |
| `Just` / `Of` | Given values, then completes |
| `FromSlice` | The slice's elements |
| `FromChannel` | Whatever a Go channel delivers |
| `Range` / `RangeWithStep` | A bounded integer sequence |
| `RangeWithInterval` / `RangeWithStepAndInterval` | Integers spaced by a duration |
| `Interval` | Sequential integers on a timer (infinite) |
| `IntervalWithInitial` | Interval with an initial delay |
| `Timer` | A single value after a delay |
| `Repeat` / `RepeatWithInterval` | A value repeated n times, optionally spaced |
| `Defer` | The result of a factory, per subscription |
| `Future` | One value from an async `(T, error)` function |
| `Start` | One value from a callback |
| `Empty` / `Never` / `Throw` | Nothing / nothing ever / an immediate error |

Direct observables give full control:

```go
obs := ro.NewObservable[int](func(ctx context.Context, observer ro.Observer[int]) error {
    observer.Next(1)
    observer.Next(2)
    observer.Complete()
    return nil
})
```

## Transformation

| Operator | Effect |
| --- | --- |
| `Map` / `MapI` | Value transform, with and without an index |
| `MapErr` | Transforming that can fail; errors propagate |
| `MapWithContext` | Map with context access |
| `MapTo` | Replace everything with a constant |
| `FlatMap` / `MergeMap` | Map each value to an observable, flattening results |
| `Scan` | Running accumulation, emitting every intermediate |
| `Reduce` | Accumulation emitting only the final value |
| `GroupBy` | Split into keyed observables |
| `Cast` | Type-cast values |
| `Flatten` | `Observable[[]T]` → `Observable[T]` |
| `Materialize` / `Dematerialize` | Wrap in / unwrap from `Notification[T]` |
| `Timestamp` / `TimeInterval` | Attach emission time / time since the last value |

```go
paid := ro.Pipe2(
    userIDs,
    ro.FlatMap(func(id int) ro.Observable[Order] { return fetchOrders(id) }),
    ro.Filter(func(o Order) bool { return o.Status == "paid" }),
)

// Scan: running total
ro.Pipe1(ro.Just(1, 2, 3, 4, 5), ro.Scan(func(acc, x int) int { return acc + x }, 0))
// emits 1, 3, 6, 10, 15
```

## Filtering

| Operator | Keeps |
| --- | --- |
| `Filter` / `FilterI` | Values matching the predicate, with optional index |
| `FilterWithContext` | Values matching with context access |
| `Distinct` / `DistinctBy` | First occurrence, by value or key |
| `Take` / `TakeLast` | First / last n values |
| `TakeWhile` | Values while the predicate holds |
| `TakeUntil` | Values up to a signal emission |
| `Skip` / `SkipLast` | Everything except the first / last n |
| `SkipWhile` / `SkipUntil` | Values after the predicate / signal |
| `Head` / `Tail` | First item / rest |
| `ElementAt` / `ElementAtOrDefault` | Value at an index, with fallback |
| `Find` / `First` / `Last` | First / last matching value |
| `Contains` | A boolean: did anything match |

Every filter has `I`, `WithContext`, and `IWithContext` variants.

## Combining

| Operator | Combines |
| --- | --- |
| `Merge` / `MergeWith` / `MergeAll` | Interleaves all sources |
| `Concat` / `ConcatWith` / `ConcatAll` | Completes sources sequentially |
| `Zip2` … `Zip6` / `ZipWith` | Pairs values across sources into `lo.Tuple` |
| `CombineLatest2` … `CombineLatest5` / `CombineLatestWith` | Re-emits the latest tuple when any source fires |
| `Race` / `Amb` | Winner takes all at first emission |
| `StartWith` / `EndWith` | Prepend / append values |

```go
ro.Zip2(userStream, settingsStream)   // lo.Tuple2[User, Settings]
ro.CombineLatest2(priceStream, quantityStream)
```

## Aggregation

`Count`, `Sum`, `Average`, `Max`, `Min` over the stream; numeric helpers `Abs`, `Ceil`, `Floor`, `Round`, `Trunc`, `CeilWithPrecision`, `FloorWithPrecision`; and `Clamp` to constrain values. `Average`, `Abs`, and the rounding set return `float64`.

## Error handling

| Operator | Recovery |
| --- | --- |
| `Catch` | Switch to a recovery observable |
| `OnErrorReturn` | Replace the error with a value |
| `OnErrorResumeNextWith` | Continue with fallback observables |
| `Retry` | Retry forever |
| `RetryWithConfig` | Bounded retries with backoff |
| `ThrowIfEmpty` | Error on an empty stream |

```go
ro.RetryWithConfig[Response](ro.RetryConfig{
    Max:               3,
    Delay:             time.Second,
    BackoffMultiplier: 2.0,
    MaxDelay:          10 * time.Second,
})
```

## Timing and buffering

`Delay` / `DelayEach` shift the stream; `Timeout` errors on silence; `ThrottleTime` / `ThrottleWhen` suppress values within a window; `SampleTime` / `SampleWhen` emit the latest at intervals; `BufferWithCount` / `BufferWithTime` / `BufferWithTimeOrCount` / `BufferWhen` collect batches; `WindowWhen` exposes nested observables; `Pairwise` emits consecutive pairs.

## Side effects

`Tap` (alias `Do`) observes events without altering the stream; granular variants target one event each — `TapOnNext`, `TapOnError`, `TapOnComplete`, `TapOnSubscribe`, `TapOnFinalize` (with `Do*` aliases and `WithContext` forms).

## Sharing and connectables

`Share` turns a cold observable hot with reference counting; `ShareReplay` also replays the last n values to newcomers; `ShareReplayWithConfig` / `ShareWithConfig` control lifecycle via `ShareConfig` reset flags; `Serialize` queues emissions for ordered delivery.

```go
ro.ShareConfig[T]{
    ResetOnComplete:       true,
    ResetOnError:          true,
    ResetOnReferenceCount: true,
}
```

## Context

`ContextReset` swaps the pipeline context; `ContextWithValue` / `ContextWithTimeout` / `ContextWithDeadline` derive from it; `ContextMap` transforms it; `ThrowOnContextCancel` converts cancellation into an error.

## Conditional

`All` reports whether every value matched; `DefaultIfEmpty` supplies a value for empty streams; `ThrowIfEmpty` errors instead; `Iif` picks between two observables; `While` / `DoWhile` repeat while a condition holds; `SequenceEqual` compares two streams.

## Terminal

`Collect` / `CollectWithContext` block and return `[]T`; `ToSlice` emits one `[]T` on completion; `ToChannel` hands the stream to a `<-chan Notification[T]`; `ToMap` folds into a `map[K]V`.

## Observers and subscriptions

```go
observer := ro.NewObserver[T](onNext, onError, onComplete)   // full observer
observer := ro.NewObserverWithContext[T](...)               // context-aware
observer := ro.OnNext[T](func(v T) { ... })                 // value-only shortcut
observer := ro.PrintObserver[T]()                            // debug: log all events
observer := ro.NoopObserver[T]()                             // discard everything

sub := observable.Subscribe(observer)
sub.Wait()         // block until complete or error
sub.Unsubscribe()  // cancel and clean up
sub.IsActive()     // still running?
sub.GetError()     // terminal error
```

## Scheduling

`SubscribeOn(bufferSize)` runs the subscription on an async scheduler; `ObserveOn(bufferSize)` delivers notifications there.

## Concurrency modes

`safe` (default) synchronizes emissions; `unsafe` skips synchronization entirely and demands single-goroutine use; `eventually safe` allows a brief unsynchronized startup before locking in.

```go
ro.NewObservable[T](fn)              // safe
ro.NewSafeObservable[T](fn)          // safe, explicit
ro.NewUnsafeObservable[T](fn)        // no sync
ro.NewEventuallySafeObservable[T](fn) // syncs after startup
```