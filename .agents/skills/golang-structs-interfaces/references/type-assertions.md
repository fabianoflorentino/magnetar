# Type Assertions & Type Switches

## Safe type assertion

Assertions use the comma-ok form. The single-value form panics when the dynamic type does not match, turning a recoverable branch into a crash:

```go
// Good — safe
s, ok := val.(string)
if !ok {
    // handle the mismatch
}

// Bad — panics if val is not a string
s := val.(string)
```

## Type switch

Discover the dynamic type of an interface value:

```go
switch v := val.(type) {
case string:
    fmt.Println(v)
case int:
    fmt.Println(v * 2)
case io.Reader:
    io.Copy(os.Stdout, v)
default:
    fmt.Printf("unexpected type %T\n", v)
}
```

Cases are evaluated top to bottom, so put concrete types before the interfaces they satisfy — a concrete type listed after its interface is unreachable.

A `nil` interface value matches `case nil`, not `default`. Add that case explicitly when nil is a valid input, otherwise it falls through to `default` and gets logged as an unexpected type.

## Optional behavior with assertions

Check for a richer capability without demanding it in the declared parameter type — the parameter stays minimal, implementations stay optional:

```go
type Flusher interface {
    Flush() error
}

func writeData(w io.Writer, data []byte) error {
    if _, err := w.Write(data); err != nil {
        return err
    }
    // Flush only when the writer supports it
    if f, ok := w.(Flusher); ok {
        return f.Flush()
    }
    return nil
}
```

The standard library leans on this everywhere — `http.Flusher`, `io.ReaderFrom`, `io.WriterTo`.

## Assert to an interface, not a concrete type

Assert to the smallest interface carrying the behavior you need. `w.(*os.File)` pins the code to one implementation; `w.(interface{ Sync() error })` accepts every type that can do the job.

## Errors

Error inspection has its own helpers — `errors.As` walks the wrap chain, a plain type assertion does not. A bare `err.(*MyError)` misses an error wrapped with `%w` anywhere up the stack.

See `fabianoflorentino/golang-agent-skills@golang-error-handling` for wrapping and inspection patterns.
