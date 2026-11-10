# Code Review Red Flags

Below are patterns worth stopping a review for. The fix for each is tracked in [common-go-bugs.md](./common-go-bugs.md).

| Pattern | Why it is a problem |
| --- | --- |
| `result, _ := doSomething()` | swallowed error — surfaces later as a mystery bug |
| `go func() { ... }()` with no way to stop | uncancellable goroutine, leaks under load |
| channel opened but never closed | sender side can wedge receivers forever |
| `time.After` inside a loop | fresh timer per iteration; reuse a timer or ticker |
| global map without a mutex | data race — and concurrent map access is fatal |
| `defer` inside a loop | deferred calls accumulate until the function returns |
| `json.Marshal` in a hot path | expensive, GC pressure |
| `for range` receive without `ok` | hides a closed channel |
| `var err *MyError; return err` | typed-nil interface trap |
| `http.Get` / default client | no timeouts anywhere in the stack |
| `fmt.Errorf("...: %v", err)` | loses the error chain; use `%w` |
| `:=` shadowing an outer `err` | outer `err` stays nil while the inner one is handled |
| value receiver on a struct with `sync` fields | copies the mutex on every call |
| `wg.Add(1)` inside the goroutine | races `wg.Wait()` — Wait may return early |
| `http.Error(...)` without `return` | handler continues after writing the error |
| `iota` enum starting at 0 | zero value indistinguishable from a real state |
| `strings.Trim(s, "prefix")` | strips a character set, not a substring |
| `log.Fatal` where defers are pending | `os.Exit` skips all deferred cleanup |
| `t1 == t2` on `time.Time` | monotonic clock makes equal instants unequal |
| `db.Query` rows never closed | database connection leak |
| `ch <- v` after `close(ch)` | panics; only the sender closes |
| `select { ... default: }` in a loop | busy-wait burning CPU |
| `int8(bigInt64)` | silent truncation with no overflow check |
| `filepath.Join(base, userInput)` | does not confine `..` traversal to the base |
| `regexp.MustCompile` per call | recompiles every invocation — hoist to package level |
| `fallthrough` | runs the next case body unconditionally |