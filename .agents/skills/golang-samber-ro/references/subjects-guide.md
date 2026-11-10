# Subjects Guide

A subject is both observable and observer: you push to it with `Send`/`Error`/`Complete`, and subscribers receive what it emits. Subjects are natively **hot** — one shared execution, and late subscribers miss earlier emissions unless replay is configured.

## Subjects vs cold observables

| Need | Approach |
| --- | --- |
| Transform a known source (slice, channel, HTTP) | Cold observable (default) |
| Decoupled producers and consumers | Subject |
| One expensive stream, many consumers | Cold observable + `Share()` / `ShareReplay()` |
| Imperative code pushes values into a pipeline | Subject |
| Bridge a callback API into reactive code | Subject (receive callbacks, emit to pipeline) |

## The five subject types

**PublishSubject** — standard multicast. Subscribers see only what is emitted after they subscribe; no history. Fit: UI events, log streams, notifications where late joiners need nothing.

```go
subject := ro.NewPublishSubject[string]()
subject.Subscribe(ro.OnNext(func(s string) { fmt.Println("sub1:", s) }))
subject.Send("hello") // sub1 sees this
subject.Send("world") // every current subscriber sees this
```

**BehaviorSubject** — replays the most recent value (or its initial value) to every new subscriber immediately. Fit: current-state readers — config values, connection status, latest price.

```go
subject := ro.NewBehaviorSubject[int](0) // initial value 0
subject.Subscribe(ro.OnNext(func(v int) { fmt.Println("sub1:", v) })) // immediately 0
subject.Send(42)
```

**ReplaySubject** — buffers the last N values and replays them to late subscribers. Fit: recent-history consumers — chat, last N ticks.

```go
subject := ro.NewReplaySubject[string](3) // buffer size 3
subject.Send("a")
subject.Send("b")
subject.Send("c")
subject.Send("d") // "a" evicted
```

**AsyncSubject** — emits only the last value, and only at completion; an error means nothing is delivered. Fit: final-result consumers — a computation's answer, the last item of a batch.

```go
subject := ro.NewAsyncSubject[int]()
subject.Send(1)
subject.Send(2)
subject.Subscribe(ro.NewObserver(
    func(v int) { fmt.Println(v) }, // receives 3 only
    func(err error) {},
    func() { fmt.Println("done") },
))
subject.Send(3)
subject.Complete()
```

**UnicastSubject** — exactly one subscriber, with an internal buffer that fills until it connects. Fit: job queues and single-consumer request pipelines.

```go
subject := ro.NewUnicastSubject[int](100) // buffer size
subject.Send(1) // buffered
subject.Send(2) // buffered
// a second Subscribe panics or errors
```

## Cold to hot conversion

A cold observable re-executes per subscriber — an HTTP source would fire a request for each one. Convert instead:

```go
cold := httpPlugin.Get[Data](url)
hot := ro.Pipe1(cold, ro.Share[Data]())

hot.Subscribe(uiObserver)
hot.Subscribe(metricsObserver) // shares the same execution
```

`Share` reference-counts: the source subscribes at the first subscriber and tears down at the last. `ShareReplay(n)` additionally replays the last n values. `Connectable` defers the shared start until an explicit `Connect(ctx)`:

```go
connectable := ro.Connectable[Data](cold)
connectable.Subscribe(observer1)
connectable.Subscribe(observer2)
sub, err := connectable.Connect(ctx)
```

## Decision table

| Subject | Replays to newcomers | Subscribers | Fit |
| --- | --- | --- | --- |
| `PublishSubject` | Nothing | Many | Event bus, notifications |
| `BehaviorSubject` | Last value (+ initial) | Many | Current state, config |
| `ReplaySubject` | Last N values | Many | Recent history |
| `AsyncSubject` | Last value, at complete | Many | Final result |
| `UnicastSubject` | Buffered pre-subscribe | Exactly one | Single-consumer queue |

## Common mistakes

| Mistake | Why it fails | Fix |
| --- | --- | --- |
| `Send` after `Complete` | Subjects are terminal; values drop silently | Do not reuse a completed subject |
| PublishSubject where history matters | Late subscribers miss everything | BehaviorSubject (last 1) or ReplaySubject (last N) |
| Unbounded ReplaySubject | Memory grows without limit | Always set an explicit buffer size |
| Second subscriber on UnicastSubject | Panic or undefined behavior | Multicast types for many consumers |
| Never calling `Complete`/`Error` | Subscribers wait forever; goroutine leak | Terminate the subject when the source ends |