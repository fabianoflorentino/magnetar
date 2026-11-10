# Nil Safety Deep Dive

Nil in Go is a value, not an exception. It is safe to *read* in many positions (a nil map, a nil interface check) and fatal in others (a nil map write, a method call that dereferences a nil receiver). The trap cases below are where the language's leniency hides a crash.

## Nil pointer receivers

Nothing stops you calling a method on a nil pointer — whether it panics depends on whether the method dereferences the receiver:

```go
type Logger struct {
    prefix string
}

func (l *Logger) IsEnabled() bool {
    return l != nil // does not dereference; safe even on nil
}

func (l *Logger) Log(msg string) {
    fmt.Printf("[%s] %s\n", l.prefix, msg) // dereferences l — panics on nil
}

var l *Logger
l.IsEnabled() // false — works
l.Log("test") // panic: nil pointer dereference
```

Do not rely on the safe case: calling a method on a nil pointer is a bug regardless of whether it happens to survive. But when a nil receiver is a *valid state* — an optional component — guard explicitly and document it:

```go
func (l *Logger) Log(msg string) {
    if l == nil {
        return // silently skip when no logger is configured
    }
    fmt.Printf("[%s] %s\n", l.prefix, msg)
}
```

This pattern earns its place for optional dependencies. Use it sparingly: a nil receiver usually signals a bug, not an intentional state.

## Nil function values

Calling a nil `func` variable panics, so validate before calling:

```go
type Worker struct {
    onComplete func(result string)
}

func (w *Worker) Finish(result string) {
    w.onComplete(result) // panic if onComplete was never set
}
```

```go
func (w *Worker) Finish(result string) {
    if w.onComplete != nil {
        w.onComplete(result)
    }
}
```

Better still, seed a no-op default in the constructor so call sites never check:

```go
func NewWorker(opts ...Option) *Worker {
    w := &Worker{
        onComplete: func(string) {}, // no-op default
    }
    for _, opt := range opts {
        opt(w)
    }
    return w
}
```

## The typed-nil interface trap in error returns

An interface is nil only when both its type and its value are nil. A function that returns `error` must return the untyped `nil`, not a typed nil pointer — returning a typed nil pointer yields a non-nil `error` that equates to nothing:

```go
func validate(s string) error {
    var err *ValidationError // typed nil
    if s == "" {
        err = &ValidationError{Field: "name"}
    }
    return err // even when err is nil, the interface is non-nil
}
```

```go
func validate(s string) error {
    if s == "" {
        return &ValidationError{Field: "name"}
    }
    return nil
}
```

`errors.Is(err, nil)` returns true only for a truly nil error — it cannot rescue a typed-nil interface, because the trap has already happened by the time the error reaches it.

## Nil in generics

**`comparable` does not mean nillable.** A `T comparable` value can be compared with `==`, but "zero" differs per type — `nil` for a pointer, `0` for an `int` — so `IsZero` is only meaningful if you say what empty means:

```go
// Confusing — "zero" for *Foo is nil, for int is 0.
func IsZero[T comparable](v T) bool {
    var zero T
    return v == zero
}

// Explicit — the constraint itself names the only nillable shape.
func IsNil[T interface{ ~*U }, U any](v T) bool {
    return v == nil
}
```

**Unconstrained type parameters cannot be compared to nil at all** — it is a compile error:

```go
func Check[T any](v T) bool {
    return v == nil // compile error: cannot compare T with nil
}

func IsNilPtr[T any](v *T) bool {
    return v == nil
}
```

## Patterns for nil-safe APIs

**Constructor with defaults** — make the zero value impossible to construct for external callers:

```go
type Client struct {
    httpClient *http.Client
    baseURL    string
}

func NewClient(baseURL string) *Client {
    return &Client{
        httpClient: http.DefaultClient,
        baseURL:    baseURL,
    }
}
```

**Lazy initialization** — keep the zero value usable while still allocating on first write:

```go
type Cache struct {
    mu   sync.Mutex
    data map[string]any
}

func (c *Cache) Get(key string) (any, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.data == nil {
        return nil, false
    }
    v, ok := c.data[key]
    return v, ok
}

func (c *Cache) Set(key string, val any) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.data == nil {
        c.data = make(map[string]any)
    }
    c.data[key] = val
}
```

→ See `fabianoflorentino/golang-agent-skills@golang-error-handling` skill for the full family of nil-error-interface pitfalls.
