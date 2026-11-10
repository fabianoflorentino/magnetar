# Advanced Patterns

Composition, stdlib interop, monadic integration, lazy iteration, and the measured moves to `lom`/`lop`.

## Composing transformations

Helpers return plain collections, so pipelines chain directly. One compound example — active verified users grouped by role:

```go
emailsByRole := lo.GroupBy(
    lo.Map(
        lo.Filter(users, func(u User, _ int) bool {
            return u.Active && u.EmailVerified
        }),
        func(u User, _ int) UserEmail {
            return UserEmail{Role: u.Role, Email: u.Email}
        },
    ),
    func(ue UserEmail, _ int) string { return ue.Role },
)
```

For chains beyond two or three steps, name the intermediates — a variable per stage reads better and is easier to step through:

```go
active := lo.Filter(users, func(u User, _ int) bool { return u.Active })
names := lo.Map(active, func(u User, _ int) string { return u.Name })
unique := lo.Uniq(names)
```

## `lo` + stdlib interop

If `slices.*` or `maps.*` covers the operation, use it; `lo` earns its keep on predicates, grouping, transforms, and error variants.

| Operation | stdlib (prefer) | `lo` when the stdlib lacks it |
| --- | --- | --- |
| Contains | `slices.Contains(s, v)` | `lo.ContainsBy(s, fn)` |
| Sort | `slices.SortFunc` / `slices.Sort` | — (lo has no sort) |
| Keys | `slices.Collect(maps.Keys(m))` | `lo.UniqKeys(m)` |
| Clone | `slices.Clone(s)` | `lo.Map` when cloning must transform |
| Min/Max | `slices.Min(s)` | `lo.MinBy(s, fn)` / `lo.MaxBy(s, fn)` |

## `lo` + `samber/mo` integration

Monadic values drop into transforms cleanly. `mo.Option` inside `FilterMap` collapses to a boolean:

```go
emails := lo.FilterMap(users, func(u User, _ int) (string, bool) {
    email, ok := u.Email.Get() // mo.Option[string]
    return email, ok
})
```

`mo.Result` contributes successes by unwrapping and keeping only the `Ok` cases:

```go
results := lo.FilterMap(urls, func(url string, _ int) (Response, bool) {
    res := fetchURL(url) // mo.Result[Response]
    val, err := res.Get()
    return val, err == nil
})
```

## Lazy iterators (`loi`)

Go 1.23+ `range`-over-func pipelines skip the intermediate allocations an eager chain pays per step:

```go
// Eager — two intermediate slices
result := lo.Map(lo.Filter(bigSlice, filterFn), mapFn)

// Lazy — zero intermediates
for v := range loi.Map(loi.Filter(bigSlice, filterFn), mapFn) {
    process(v)
}
```

A classic bounded chain: filter, map, take first 10.

```go
for name := range loi.Take(
    loi.Map(
        loi.Filter(records, func(r Record) bool { return r.Score > 0.8 }),
        func(r Record) string { return r.Name },
    ),
    10,
) {
    fmt.Println(name)
}
```

## Performance-sensitive patterns

Switch to `lom` only when `go tool pprof -alloc_objects` names `lo.Filter` or `lo.Map` as top allocators in a hot path:

```go
// Before — fresh slice every call
filtered := lo.Filter(events, isValid)

// After — zero allocations, input rewritten
events = lom.Filter(events, isValid)
// Warning: 'events' now holds only the survivors.
```

Switch to `lop` only when `go tool pprof -cpu` shows a CPU-heavy transform dominating a large dataset:

```go
results := lop.Map(largeSlice, expensiveTransform)
```

## Testing with `lo`

Helpers generate fixtures and tighten assertions:

```go
users := lo.Times(100, func(i int) User {
    return User{ID: i, Name: fmt.Sprintf("user-%d", i)}
})

assert.True(t, lo.Every(
    expectedIDs,
    lo.Map(actual, func(u User, _ int) int { return u.ID }),
))

ids := lo.Times(50, func(_ int) string {
    return lo.RandomString(16, lo.AlphanumericCharset)
})

counts := lo.CountValues(results)
assert.Equal(t, 3, counts["success"])
```

## Slice-to-map conversion

Build lookup maps in one pass:

```go
userByID := lo.SliceToMap(users, func(u User) (int, User) {
    return u.ID, u
})

userByEmail := lo.KeyBy(users, func(u User) string {
    return u.Email
})

activeByID := lo.FilterSliceToMap(users, func(u User) (int, User, bool) {
    return u.ID, u, u.Active
})
```

`FilterSliceToMap` merges the filter and the conversion, skipping entries whose predicate returns false.