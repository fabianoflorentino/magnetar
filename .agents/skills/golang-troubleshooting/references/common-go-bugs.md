# Common Go Bugs

→ See the `fabianoflorentino/golang-agent-skills@golang-safety` skill for the detailed nil, slice, and map safety patterns behind the fixes below.

## Nil and interface traps

### Nil pointer dereference

The most common Go panic, and the one stack traces pinpoint best. Guard pointers from external sources before touching them:

```go
// Uninitialized struct field
type Server struct {
    logger *log.Logger // nil unless the constructor sets it
}

// Unchecked error return — on error, val may be nil
val, err := doSomething()
val.Method() // panics when doSomething returned (nil, err)

// Map lookup yields the zero value (nil for pointers)
cfg := m["missing"] // nil
cfg.Timeout         // panics

// Type assertion without the comma-ok form
n := i.(int)       // panics
n, ok := i.(int)   // ok == false, no panic
```

### Typed-nil interface values

An interface holding a typed nil pointer is **not** nil. Comparing it to `nil` succeeds, and calling methods on it panics:

```go
type MyError struct{ msg string }

func (e *MyError) Error() string { return e.msg }

func doWork() error {
    var err *MyError // typed nil pointer
    return err       // non-nil interface wrapping a nil pointer
}

if err := doWork(); err != nil {
    fmt.Println(err) // panics inside Error()
}
```

Return explicit `nil` for success paths; never return a typed-nil variable through an interface.

### `recover()` is per-goroutine

A deferred `recover()` only catches panics in its **own** goroutine. A panic in a spawned goroutine escapes and kills the process — no parent can catch it:

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("recovered: %v", r)
        }
    }()
    panic("crash!") // contained here
}()
```

Every goroutine boundary must install its own recovery.

### Copying `sync` types

`sync.Mutex`, `sync.RWMutex`, `sync.WaitGroup`, `sync.Once`, `sync.Cond`, `sync.Map`, and `sync.Pool` must never be copied after use — a value receiver silently copies the lock and synchronization stops:

```go
type Counter struct {
    mu    sync.Mutex
    count int
}

// BAD: copies the mutex on every call
func (c Counter) Increment() {
    c.mu.Lock()
    c.count++
    c.mu.Unlock()
}

// GOOD: pointer receiver
func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

`go vet` flags mutex copies.

## Scoping and control flow

### Variable shadowing with `:=`

Inside a block, `:=` declares a **new** variable instead of assigning the outer one. Shadowing `err` silently breaks error handling:

```go
func doWork() error {
    var err error
    if condition {
        result, err := someFunc() // new err, outer err untouched
        if err != nil {
            return err
        }
        process(result)
    }
    return err // always nil
}
```

Fix: declare the result up front and assign (`result, err = someFunc()`). Detect with the `golang.org/x/tools/go/analysis/passes/shadow` analyzer — not part of standard `go vet`.

### `break` in `select`/`switch` inside a loop

A bare `break` exits the `select`/`switch`, not the surrounding `for`:

```go
loop:
for {
    select {
    case msg := <-ch:
        if msg == "quit" {
            break loop // exit the for loop
        }
        process(msg)
    }
}
```

### `fallthrough` runs unconditionally

`fallthrough` jumps into the **next case body** without evaluating its condition. In practice `case a, b:` is the clearer spelling of what most code means:

```go
switch x := 5; {
case x > 10:
    fmt.Println(">10")
    fallthrough
case x > 0:
    fmt.Println(">0") // also last case's body runs
}
```

### Zero-value enums are ambiguous

`iota` starts at 0, which collides with the zero value of every uninitialized variable, struct field, and missing JSON field. Reserve 0 for "unset":

```go
type Status int

const (
    StatusUnknown Status = iota // 0 — explicit unset sentinel
    StatusActive                // 1
    StatusInactive              // 2
)
```

### `os.Exit` skips deferred cleanup

`os.Exit` terminates immediately — defers never run, and `log.Fatal` is `os.Exit(1)` under the hood. Route `main` through a `run() error` so defers complete before the process ends:

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func run() error {
    f, _ := os.Create("data.tmp")
    defer f.Close()
    return process()
}
```

## Slices and maps

### Nil map writes panic; nil reads are fine

Reads return the zero value; writes crash. `make(map[string]int)` before writing.

### Append can overwrite a shared backing array

Slicing keeps a shared backing array, so an append to the sub-slice can clobber neighbors:

```go
a := []int{1, 2, 3}
b := a[:2]
b = append(b, 99) // overwrites a[2]!
```

Fix with a full slice expression to cap capacity: `b := a[:2:2]`. See `golang-safety` for the full aliasing rules.

### Loop variable capture in goroutines

Before Go 1.22, `for _, v := range items { go func() { process(v) }() }` shares one `v` across iterations — every goroutine sees roughly the last element. Pass it as an argument, or rely on per-iteration variables in Go 1.22+.

### Concurrent map access is fatal

A concurrent map read/write is a **runtime fatal** — `recover()` cannot catch it and the process dies. The race detector surfaces it; tests only sometimes do (timing). Guard with `sync.RWMutex`, use `sync.Map` for read-heavy stable key sets, or `go test -race ./...` in CI.

### Map iteration order is random

The runtime deliberately randomizes map order. Never rely on it for output, serialization, logging, or test comparisons — sort keys first when order matters:

```go
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
```

### `WaitGroup.Add` inside the goroutine

Calling `wg.Add(1)` inside the spawned goroutine races with `wg.Wait()` — Wait can return before any Add lands. Add before launching:

```go
for i := 0; i < n; i++ {
    wg.Add(1) // before the goroutine starts
    go func() {
        defer wg.Done()
        doWork()
    }()
}
wg.Wait()
```

## Defer and errors

### `defer` evaluates arguments immediately

The arguments (and the receiver) are evaluated at the `defer` statement, not at return. `defer fmt.Println(x)` prints the value of `x` at defer time.

### `defer` inside a loop piles up

Deferred calls run at function return, so a loop that defers leaks every pending call. Wrap the loop body in a closure:

```go
for _, f := range files {
    func() {
        file, _ := os.Open(f)
        defer file.Close()
        // use file
    }()
}
```

### Named returns + defer

A deferred closure can inspect and amend the named result — the standard pattern for turning a close error into the function's error only when nothing more important already failed:

```go
func readFile() (err error) {
    f, err := os.Open("file.txt")
    if err != nil {
        return err
    }
    defer func() {
        if closeErr := f.Close(); err == nil {
            err = closeErr
        }
    }()
    // ...
    return nil
}
```

### Swallowed errors

`result, _ := doSomething()` and bare `json.Unmarshal(data, &config)` are the most common source of "mysterious" bugs. Handle or propagate with `%w`; find the leftovers with `go vet ./...` and `go tool errcheck ./...`.

### Wrapping and matching

```go
return fmt.Errorf("reading config from %s: %v", path, err) // BAD — chain lost
return fmt.Errorf("reading config from %s: %w", path, err) // GOOD

if err == sql.ErrNoRows { ... }          // BAD — breaks when wrapped
if errors.Is(err, sql.ErrNoRows) { ... } // GOOD — walks the chain

var pathErr *os.PathError
if errors.As(err, &pathErr) { ... } // typed extraction
```

## Context misuse

- **Forgetting `defer cancel()`** leaks the goroutines waiting on the context.
- **Deriving from `context.Background()`** when a parent context exists breaks cancellation propagation — pass the parent through.
- **Not reading `ctx.Err()`** — distinguish `context.DeadlineExceeded` from `context.Canceled` before blaming the operation.
- **Reusing the request context for background work** — it cancels when the client disconnects. Derive a fresh one in Go 1.21+ with `context.WithoutCancel(r.Context())`.

## Channels

### Send on closed channel panics

Only the sender closes; the receiver reads the zero value and `ok == false` forever after. With multiple senders, coordinate the close with `sync.Once` or a `sync.WaitGroup`:

```go
func producer(ch chan<- int, done <-chan struct{}) {
    defer close(ch)
    for i := 0; ; i++ {
        select {
        case ch <- i:
        case <-done:
            return
        }
    }
}
```

### A closed channel in `select` spins the CPU

A closed channel is always ready for receive, so its case fires continuously at 100% CPU. Nill the channel when you see `!ok` — a nil channel in `select` blocks forever:

```go
case v, ok := <-ch:
    if !ok {
        ch = nil // disable this case
        continue
    }
    process(v)
```

### `select` with `default` in a loop

`default` makes the select non-blocking; wrapped in a `for`, that is a busy-wait. Drop the default (block until a message or `ctx.Done()`) or pace the loop with a sleep/ticker.

## HTTP, files, and resources

### Missing `return` after `http.Error`

`http.Error` only writes the response; execution continues into code that should be skipped, often writing a second response:

```go
if !authorized(r) {
    http.Error(w, "Forbidden", http.StatusForbidden)
    return // without this, doSensitiveAction still runs
}
doSensitiveAction(r)
```

### `sql.Rows` must be closed

Unclosed rows hold a database connection until the pool exhausts. `defer rows.Close()` immediately after the query, check `rows.Err()` after iteration, and use `db.QueryRow`/`db.Exec` where a full `Rows` is not needed.

### `filepath.Join` does not contain paths

`filepath.Join` cleans `..` but does not confine the result to the base directory — `filepath.Join("/srv/files", "../../etc/passwd")` escapes. In Go 1.24+ open a root with `os.OpenRoot` and open paths against it; older versions need a lexical `../` check via `filepath.Rel`.

## Strings and JSON

### `len()` counts bytes, indexing returns bytes

UTF-8 multi-byte runes corrupt counts and slices:

```go
s := "Hello, 世界"
len(s)                     // 13 bytes, not 9 runes
s[:8]                      // cuts a rune in half
utf8.RuneCountInString(s)  // 9
string([]rune(s)[:8])      // 8 runes
```

`for range` iterates runes, not bytes.

### `strings.Trim` strips a character set

`strings.Trim(s, "application/")` removes any of those **characters** from both ends — not the substring. Use `strings.TrimPrefix`/`TrimSuffix` for substrings.

### JSON numbers become `float64` in `interface{}`

Unmarshaling into `map[string]any` turns every number into `float64`; `(int)` assertions panic and integers above 2^53 lose precision. Prefer typed structs; with `any`, force `json.Number`:

```go
dec := json.NewDecoder(bytes.NewReader(data))
dec.UseNumber()
var result map[string]interface{}
dec.Decode(&result)
id, _ := result["id"].(json.Number).Int64()
```

### Unexported JSON fields are silently dropped

Lowercase fields are invisible to `encoding/json` — marshal emits `{}` and unmarshal skips them, no error. Fields must be exported to round-trip. `go vet` warns when unexported fields carry JSON tags.

### Integer conversions truncate silently

Conversions do not range-check: `int8(256)` is `0`, `int32(math.MaxInt64)` wraps. Check bounds before converting untrusted values.

## Types, packages, and initialization

### Pointer-receiver interface satisfaction

A value `T` only implements the interface when all methods are value-receiver. Anything with a `*T` receiver requires a pointer to satisfy the interface:

```go
type Sizer interface{ Size() int }

type File struct{ size int }

func (f *File) Size() int { return f.size }

var s Sizer
s = File{}  // compile error — *File has the method
s = &File{} // ok
```

### `regexp.MustCompile` in a hot path

Compiling a regexp on every call is expensive. Hoist long-lived patterns to package-level vars; only one-shot CLIs and tests compile inline.

### `init()` ordering is fragile

`init()` order is deterministic (file order within a package, alphabetical across files, dependency order across packages) — which is exactly why adding a file silently reorders it. Prefer explicit setup in `main()`; reserve `init()` for self-contained registration (drivers, codecs).