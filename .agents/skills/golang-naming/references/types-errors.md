# Types, Constants & Errors

## Interfaces

Name a single-method interface after its method plus `-er`:

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Stringer interface {
    String() string
}

type Closer interface {
    Close() error
}
```

Multi-method interfaces take a descriptive **noun**, or compose from single-method ones:

```go
type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}

type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Honor canonical method names and signatures. A `Read` that does not match `io.Reader` breaks the ecosystem; never invent `ReadData` or `ToString` where `String` exists.

| Method | Expected interface |
| --- | --- |
| `Read` | `io.Reader` |
| `Write` | `io.Writer` |
| `Close` | `io.Closer` |
| `String` | `fmt.Stringer` |
| `Error` | `error` |
| `Len` | `sort.Interface` |
| `ServeHTTP` | `http.Handler` |

## Structs

MixedCaps nouns for the entity; fields follow the exported/unexported rules:

```go
type Server struct {
    Addr     string        // exported
    Handler  http.Handler  // exported
    timeout  time.Duration // unexported
}
```

Never append `Struct`, `Object`, or `Data` — those suffixes carry no information.

## Constants

MixedCaps, never `ALL_CAPS`; the name states the *role*, not the *value*:

```go
const MaxRetries = 3
const defaultTimeout = 30 * time.Second
const DefaultPort = 8080

// Not ALL_CAPS
const MAX_RETRIES = 3

// Naming the value, not the purpose
const Port8080 = 8080
```

### Enums (`iota`)

Preface enum values with the type name to avoid collisions and read clearly at the call site:

```go
type Status int

const (
    StatusUnknown Status = iota // zero value = unknown/invalid
    StatusReady
    StatusRunning
    StatusDone
)

type Color int

const (
    ColorRed Color = iota + 1 // skip zero to catch the uninitialized
    ColorGreen
    ColorBlue
)
```

The zero value is a contract. `var s Status` is silently 0 — if 0 maps to `StatusReady`, the code behaves as if a state was chosen when none was. Put an explicit `Unknown` sentinel at iota 0, or start at `iota + 1`. This is not stylistic.

## Errors

### Sentinel errors

`Err` prefix on the variable; the string carries a **package prefix** so the origin survives wrapping:

```go
var ErrNotFound = errors.New("mypackage: not found")
var ErrPermissionDenied = errors.New("mypackage: permission denied")

// Bare strings lose their origin once wrapped
var ErrNotFound = errors.New("not found")
```

### Error types

Custom error types use the `Error` suffix:

```go
type PathError struct {
    Op   string
    Path string
    Err  error
}
```

### Error strings

Fully lowercase — acronyms included (`id`, `url`, `http`) — and no trailing punctuation. Error messages are usually printed inside a larger context (`fmt.Errorf("decoding config: %w", err)`), so they must read as a grammatical continuation:

```go
errors.New("image: unknown format")
errors.New("mypackage: invalid message id")
fmt.Errorf("decoding config: %w", err)

// Broke by casing and punctuation
errors.New("Image: Unknown format.")
errors.New("mypackage: invalid message ID")
fmt.Errorf("Failed to decode config: %w.", err)
```