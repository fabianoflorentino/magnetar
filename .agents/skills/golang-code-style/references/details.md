# Code Style Details

## Complex conditions

When an `if` condition grows past two or three operands, pull the operand groups into named booleans — the condition then reads as its own sentence:

```go
isAdmin := user.Role == RoleAdmin
isOwner := resource.OwnerID == user.ID
hasOverride := permissions.Contains(PermOverride)
if isAdmin || isOwner || hasOverride {
    allow()
}
```

Against a wall-of-logic inline condition, the named version is self-documenting:

```go
if user.Role == RoleAdmin || resource.OwnerID == user.ID || permissions.Contains(PermOverride) {
    allow()
}
```

**Exception:** when the last operand is an expensive check, keep it inline so short-circuit evaluation can skip it:

```go
if isAdmin || isOwner || expensivePermissionCheck(user, resource) {
    allow()
}

// Wasteful: the expensive check always runs
canOverride := expensivePermissionCheck(user, resource)
if isAdmin || isOwner || canOverride {
    allow()
}
```

## Value vs pointer parameters

This covers function parameters; receiver rules live in `fabianoflorentino/golang-agent-skills@golang-structs-interfaces`.

Small fixed-size types go by value — a `string` is already a (pointer, length) header, and `int`/`bool`/`float64`/`time.Time` copy in a register or two:

```go
func FormatUser(name string, age int, createdAt time.Time) string

// Mutation
func PopulateDefaults(cfg *Config)

// nil is meaningful — an optional field update
func UpdateUser(ctx context.Context, id string, name *string) error

// Unjustified pointer: Greet does not mutate or need nil
func Greet(name *string) string
```

Reach for a pointer when:

- The function mutates the value.
- The struct is large — roughly 128+ bytes — and the copy cost outweighs the dereference.
- `nil` carries meaning.

Avoid pointers when:

- The type is a small value type (`string`, `int`, `bool`, `float64`, `time.Time`).
- The access is read-only on a small struct — a value copy improves cache locality.
- The only goal is "saving memory" — value copies are cheap and stack-resident.

Under hot paths the trade-offs are: values keep small types on the stack with excellent locality and zero indirection, but get expensive once structs pass the copy-cost threshold (~128 bytes); pointers add one dereference (negligible) but risk a cache miss and win decisively for large or mutated structs. When in doubt, benchmark.

See `fabianoflorentino/golang-agent-skills@golang-structs-interfaces` for pointer vs value **receivers**.