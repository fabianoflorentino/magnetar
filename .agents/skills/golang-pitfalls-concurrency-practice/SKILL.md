---
name: golang-pitfalls-concurrency-practice
description: "Golang concurrency in practice — propagating an inappropriate context, starting a goroutine without knowing when to stop it, per-iteration loop variables, expecting deterministic select behavior, not using chan struct notification channels, not using nil channels, channel buffer size, string formatting side effects under lock, data races with append, mutex boundaries with slices and maps, misusing sync.WaitGroup, forgetting sync.Cond, not using errgroup, and copying sync types. Distilled from mistakes #61-74 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang concurrent and async code."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🚦"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Concurrency Practice

Source material: mistakes #61-74 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when writing practical Go concurrency code.

## 61. Propagating an inappropriate context (#61)

- An HTTP request context is canceled when: the client's connection closes, the request is canceled (HTTP/2), or **the response is written**.
- If you spawn an async goroutine (e.g., Kafka publish) using the request context, the context may be canceled right after the response is written, silently dropping the work.
- Propagate with care; from Go 1.21 you can use `context.WithoutCancel(parent)` for work that must outlive the request.

## 62. Starting a goroutine without knowing when to stop it (#62)

- Every goroutine is a resource; always have a plan to stop it — otherwise goroutine/resource leaks.
- Signaling alone (a canceled context) isn't enough; the parent may exit before the goroutine cleans up.
- If a goroutine's lifetime is bound to the app's, wait for it to finish before returning: expose a `close()` (use `defer w.close()`) or wait for completion.

## 63. Goroutines and loop variables (#63)

- Not relevant anymore from Go 1.22 — loop variables are per-iteration now. No `:= v` copy needed on Go 1.22+.

## 64. Expecting deterministic `select` behavior (#64)

- With several ready cases, `select` chooses **uniformly at random** (not source order) to prevent starvation.
- Receiving from `messageCh` and `disconnectCh` may exit before consuming all messages.
- For deterministic draining with a single producer, use unbuffered channels or a single channel.

## 65. Not using notification channels (#65)

- For signals without data, use `chan struct{}` (zero-size), not `chan bool` — `true`/`false` semantics are ambiguous.
- `chan struct{}` signals "event happened" without conveying a value.

## 66. Not using nil channels (#66)

- Sending to / receiving from a nil channel blocks forever — this can be used deliberately.
- Nil channels drop cases from `select`: to merge two channels, set a channel to `nil` when it's closed so its case is disabled.

```go
case v, open := <-ch1:
    if !open {
        ch1 = nil
        break
    }
    out <- v
```

## 67. Being puzzled about channel size (#67)

- **Unbuffered** channels (`make(chan T)` or `make(chan T, 0)`) give strong synchronization guarantees — sender blocks until a receiver takes the value.
- **Buffered** channels give no strong synchronization; a sender can continue once the buffer has room.
- Default buffered size is **1** unless there's a good reason: worker pooling (tie size to goroutine count) or rate limiting (size = limit).
- Queues rarely run at a balanced middle; they oscillate full/empty (Martin Thompson).

## 68. Forgetting side effects with string formatting (#68)

- `fmt.Sprintf`/`%v` may call methods (`String()`) under the hood — including a method that takes a lock.
- Formatting a receiver while holding its mutex can deadlock (e.g., `fmt.Errorf("... %v", c)` calling `c.String()` while `UpdateAge` holds the lock).
- Fixes: restrict the lock scope, format fields directly, or avoid calling `String` under lock.

## 69. Creating data races with `append` (#69)

- `append` on a shared slice is not always safe.
  - `make([]int, 1)` (len == cap) → each goroutine's `append` allocates a new backing array: **no race**.
  - `make([]int, 0, 1)` (len < cap) → goroutines write the same backing-array index: **data race**.
- Concurrent code should not `append` on a shared slice; copy first.

## 70. Using mutexes inaccurately with slices and maps (#70)

- A map/slice variable holds a header/pointer to shared data — assigning `balances := c.balances` under `RLock` then iterating **outside** the lock is still a data race, because both reference the same buckets.
- Options: hold the lock across the whole iteration, or deep-copy inside the lock (`maps.Clone`) then iterate the copy.
- Get the critical-section boundaries right.

## 71. Misusing `sync.WaitGroup` (#71)

- Call `wg.Add(n)` **before** spawning goroutines, in the parent goroutine — never inside the worker goroutines.
- Calling `Add` inside the goroutine races with `Wait()` and produces nondeterministic results.
- Good:

  ```go
  wg.Add(3)
  for i := 0; i < 3; i++ {
      go func() { atomic.AddUint64(&v, 1); wg.Done() }()
  }
  wg.Wait()
  ```

## 72. Forgetting about `sync.Cond` (#72)

- `sync.Cond` broadcasts repeated notifications to multiple waiting goroutines, where a channel would only reach one.
- Use it when many goroutines must be woken repeatedly.

## 73. Not using `errgroup` (#73)

- `golang.org/x/sync/errgroup` synchronizes a group of goroutines while propagating the first error and offering context-derivation helpers (`WithContext`).
- Prefer it over hand-rolled WaitGroup + error plumbing when goroutines return errors.

## 74. Copying a `sync` type (#74)

- Never copy types from `sync` package (`sync.Mutex`, `sync.WaitGroup`, `sync.Cond`, `sync.Pool`, etc.) — copying after first use leads to undefined behavior.
- Pass them by pointer; this is also why a receiver holding a `sync.Mutex` must use a pointer receiver.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-concurrency` for worker pools, fan-out/fan-in, and pipelines
- → See `fabianoflorentino/golang-agent-skills@golang-context` for context cancellation and propagation best practices
- → See `fabianoflorentino/golang-agent-skills@golang-safety` for nil-channel and append-aliasing safety
- → See `fabianoflorentino/golang-agent-skills@golang-testing` for race detection and goroutine-leak testing
