# Functions, Methods & Options

## Function names

Functions that return a value are named like **nouns** (what they yield); functions that do work are named like **verbs** (what they perform):

```go
func UserName() string { ... }          // noun — returns
func DefaultConfig() Config { ... }     // noun
func WriteFile(name string, data []byte) error { ... }   // verb — acts
func SendNotification(user *User) error { ... }          // verb
```

Never repeat the package name inside a function name — the call site already states it:

```go
package http

func Get(url string) (*Response, error)  // http.Get, not http.HTTPGet
```

Functions that take a format string plus variadic arguments end in **`f`**, matching the `fmt` family:

```go
func Errorf(format string, args ...any) error
func Wrapf(err error, format string, args ...any) error
func Logf(format string, args ...any)
```

A plain `Error(format, args...)` reads like it takes a literal string; `WrapError(...)` hides the format semantics.

## Getters and setters

Getters drop the `Get` prefix — the field name, capitalized, is the getter:

```go
func (u *User) Name() string         { return u.name }
func (u *User) SetName(name string)  { u.name = name }  // setter keeps Set

// Bad
func (u *User) GetName() string      { return u.name }
```

Reserve `Get` for concepts that are inherently "get" (an HTTP GET against a remote resource). Expensive or blocking reads deserve an honest verb: `Fetch` or `Compute`.

Boolean predicates are the exception and keep their question prefix — this matches the standard library (`reflect.Type.IsVariadic()`, `net.IP.IsLoopback()`, `big.Int.IsInt64()`):

```go
func (s *Server) IsHealthy() bool            { return s.healthy }
func (s *Server) Port() int                  { return s.port }

// A bare adjective as a bool is ambiguous
func (s *Server) Healthy() bool {...}

// A richer return type justifies the plain noun
func (s *Server) Healthy() (HealthStatus, error)
```

## Constructors

`New()` when the package centers on one constructible type; `NewTypeName` when several types are constructed:

```go
package ring

func New(size int) *Ring            // single primary type
```

```go
package http

func NewRequest(method, url string, body io.Reader) (*Request, error)
func NewServeMux() *ServeMux        // multiple types need qualification
```

## Named returns

Name returns only when the names earn their place — several values of the same type, or names that double as documentation:

```go
// Good: disambiguates the two int64s
func Copy(dst Writer, src Reader) (written int64, err error)

// Good: plain (int, error) needs no names
func Write(p []byte) (int, error)

// Avoid: names add nothing and hide meaning
func Read(p []byte) (bytes int, e error)   // shadows the bytes package; "e" vs "err"
```

Never reach for named returns just to enable a bare `return`; naked returns hurt readability outside the shortest functions.

## Functional options

For constructors with 3+ optional parameters that are likely to grow, use functional options:

- Options struct: `ServerOptions`, `ClientOptions` — never `Opts`, `Params`, `Settings`, or `Config`.
- Option function type: `ServerOption` (singular).
- Builder funcs: `WithPort()`, `WithTimeout()`, `WithLogger()`.
- A `DefaultServerOptions()` factory for sane baselines.