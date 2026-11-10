# Struct Fields: Tags and Copy Safety

## Struct field tags

Field tags drive every reflection-based serializer. Exported fields on serialized structs must carry tags — without one the encoder falls back to the Go field name, so renaming the field silently changes the wire format:

```go
type Order struct {
    ID        string    `json:"id"         db:"id"`
    UserID    string    `json:"user_id"    db:"user_id"`
    Total     float64   `json:"total"      db:"total"`
    Items     []Item    `json:"items"      db:"-"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    DeletedAt time.Time `json:"-"          db:"deleted_at"`
    Internal  string    `json:"-"          db:"-"`
}
```

| Directive | Effect |
| --- | --- |
| `json:"name"` | field name in JSON output |
| `json:"name,omitempty"` | omit when the field holds its zero value |
| `json:"name,omitzero"` | omit when zero, type-aware (Go 1.24+) |
| `json:"-"` | always exclude from JSON |
| `json:"-,"` | field literally named `-` |
| `json:",string"` | encode number/bool as a JSON string |
| `db:"column"` | database column mapping (sqlx, etc.) |
| `yaml:"name"` | YAML field name |
| `xml:"name,attr"` | XML attribute |
| `validate:"required"` | struct validation (go-playground/validator) |

Notes that bite in production:

- **Unexported fields never serialize**, tag or not; a tag there is dead weight.
- **`omitempty` is zero-based, not semantic.** It drops `0`, `""`, and `false`, so an explicit zero is indistinguishable from an absent field. When the difference matters, use a pointer, `mo.Option[T]`, or `omitzero`.
- **Tag syntax is unchecked at compile time.** A stray space or a missing backtick silently disables the tag. Run `go vet ./...` in CI — its `structtag` analyzer flags malformed or duplicate tags.

Diagnose tag drift with a round-trip test: marshal, unmarshal, compare. That catches a field renames the compiler cannot see.

## Preventing struct copies with `noCopy`

Structs holding a mutex, channel, or internal pointers must not be copied: a copy duplicates lock state, so two goroutines end up guarding two separate mutexes and the invariant quietly disappears. Embed a `noCopy` sentinel to make `go vet` reject copies:

```go
// noCopy marks structs that must not be copied after first use.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type ConnPool struct {
    noCopy noCopy
    mu     sync.Mutex
    conns  []*Conn
}
```

`vet`'s `copylocks` analyzer flags any copy — passed by value, assigned, returned, or ranged over — of a type whose fields implement `Lock`/`Unlock`. Detection is purely name-based, which is why the sentinel works: this is the same trick the standard library uses for `sync.WaitGroup`, `sync.Mutex`, `strings.Builder`, and friends.

Pass such structs by pointer:

```go
// Good
func process(pool *ConnPool) { ... }

// Bad — go vet flags this
func process(pool ConnPool) { ... }
```

Diagnose with `go vet ./...` (reports every value copy of a lock-bearing type) plus `go test -race ./...` (surfaces the data race a silent copy introduces).
