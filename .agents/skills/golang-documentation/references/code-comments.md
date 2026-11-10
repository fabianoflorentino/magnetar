# Doc Comments in Code

## Explain why, not what

The code already shows what it does; a doc comment earns its bytes by saying what the reader cannot see — why the function exists, when to reach for it, its constraints, and what can fail:

```go
// GetUser retrieves the full profile for an authenticated endpoint.
// For listing or searching, use ListUsers — it returns lighter projections.
//
// Returns ErrNotFound if no user has the given ID.
func GetUser(id string) (*User, error) {
```

## Anti-patterns to remove on sight

| Anti-pattern | Example | Fix |
| --- | --- | --- |
| Pure paraphrase | `// GetUser gets a user` | Add when/constraints/errors after the name |
| Signature restatement | `// Returns a string and an error` | Name the error and the condition |
| Marketing vocabulary | `seamlessly`, `robust`, `enterprise-grade` | State facts instead |
| Invented rationale | `// designed to improve scalability` | Only document real behavior |
| Groundless future claims | `// supports future extensibility` | Back it with code, or cut it |
| Hollow filler | `// It's worth noting that...` | Cut and restate the fact directly |

## Format

Every exported item's comment starts with its name, then a verb phrase; that first sentence becomes the godoc index entry.

```go
// FuncName verb phrase describing what it does.
```

## A full comment template

Use for exported functions and complex internals; drop sections that do not apply:

```go
// FuncName summarizes what this function does in one sentence.
// Additional context explaining behavior, algorithms, or design decisions
// that callers need to know.
//
// Parameters:
//   - paramName: what this parameter represents, with valid ranges
//   - anotherParam: description with constraints
//
// Returns description of the return value(s).
// Returns ErrSomething if [condition].
// Returns ErrAnother if [different condition].
//
// It is safe for concurrent use.
func FuncName(paramName Type, ...) (ResultType, error) {
```

## What gets comments

| Element | Document? |
| --- | --- |
| Exported functions, types, interfaces | Always |
| Exported constants and variables | Always |
| Complex internal functions | Yes — algorithms, non-obvious logic |
| Simple internal helpers | Only if the name is not self-explanatory |
| Test functions | No |
| Trivial getters/setters | Brief one-liner |

`TODO` comments refer to a tracking issue when one exists (for example `// TODO(#123): ...`) and identify an owner otherwise.

## Error cases and limitations

State every error a function can return, plus edge cases:

```go
// Parse parses a duration string such as "300ms", "1.5h", or "2h45m".
//
// Parameters:
//   - s: valid units are "ns", "us", "ms", "s", "m", "h".
//
// Returns the parsed duration.
// Returns ErrInvalidDuration if s is empty or malformed.
// Returns ErrOverflow if the duration exceeds math.MaxInt64 nanoseconds.
func Parse(s string) (time.Duration, error) {
```

## Deprecated functions

Use the `// Deprecated:` marker — tooling keys on the exact wording:

```go
// OldFunc does something.
//
// Deprecated: use NewFunc instead. OldFunc will be removed in v3.0.0.
func OldFunc() {}
```

## Interfaces

Document the contract implementations must satisfy, both at the interface level and per method:

```go
// Store is a persistent key-value backend.
// Implementations must be safe for concurrent use.
// All methods respect context cancellation.
type Store interface {
    // Get returns the value for key.
    // Returns ErrNotFound if the key does not exist.
    Get(ctx context.Context, key string) ([]byte, error)

    // Set stores a key-value pair with an optional TTL.
    // A ttl of 0 means the entry never expires.
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

    // Delete removes key. Deleting a missing key returns nil.
    Delete(ctx context.Context, key string) error
}
```

## Methods on structs

```go
// Close gracefully shuts down the server, waiting for active connections
// up to the configured timeout. Returns an error on timeout or drain failure.
// Close is idempotent; calling it multiple times is safe. It is NOT safe to
// call Close concurrently from multiple goroutines.
func (s *Server) Close() error {
```

## Code examples inside comments

Indent example blocks by one tab so godoc renders them as code:

```go
// Transform applies fn to each element and returns a new slice.
//
// Example:
//
//	names := []string{"alice", "bob"}
//	upper := Transform(names, strings.ToUpper)
//	// upper: ["ALICE", "BOB"]
func Transform[T, U any](slice []T, fn func(T) U) []U {
```

### Playground links

A `Play:` line gives public-library readers a one-click runnable program:

```go
// Map applies fn to each element of a slice.
//
// Play: https://go.dev/play/p/abc123xyz
//
// Example:
//
//	doubled := Map([]int{1, 2, 3}, func(x int) int { return x * 2 })
//	// doubled: [2, 4, 6]
func Map[T, U any](s []T, fn func(T) U) []U {
```

## File and package comments

### Package comment

One package comment, either atop the main `.go` file or in a dedicated `doc.go` for bigger packages. It ends with the package name:

```go
// Package httputil provides HTTP helpers for request parsing, response
// writing, and middleware chaining, built on the standard net/http package.
package httputil
```

Use `doc.go` when the package spans 3+ files or the comment exceeds about 10 lines:

```go
// Package auth implements authentication and authorization for the API server.
//
// # Architecture
//
// Each strategy (JWT, API key, OAuth2) implements the Authenticator interface;
// strategies are chained and tried in order until one succeeds.
//
// # Token lifecycle
//
// Access tokens expire after 15 minutes, refresh tokens after 7 days.
// Rotation is automatic — each refresh issues a new token and invalidates the old.
//
// # Thread safety
//
// All exported functions and types are safe for concurrent use.
package auth
```

### File-level descriptions

For files implementing an algorithm or a complex flow, a description block below the imports explains the design; ASCII art earns its keep for state machines and flows. Skip it for simple CRUD handlers and data models, and skip it when the file name already says everything.

### Headings in comments

Go 1.19+ `# Heading` lines structure long doc comments (see the auth example above) and render as real godoc sections.