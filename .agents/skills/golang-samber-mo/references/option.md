# `mo.Option[T]` Reference

The full surface of the nullability type: construction, inspection, extraction, transforms, and the serialization/DB interfaces that make it drop into structs.

## Constructors

| Function | Behavior |
| --- | --- |
| `mo.Some[T](v)` | Present value |
| `mo.None[T]()` | Absent value |
| `mo.TupleToOption[T](v, ok)` | `(value, bool)` → Some/None |
| `mo.EmptyableToOption[T](v)` | None when the value is zero, Some otherwise |
| `mo.PointerToOption[T](p)` | None on nil pointer, Some otherwise |

## Inspection and extraction

| Method | Returns | Behavior |
| --- | --- | --- |
| `IsPresent()` / `IsSome()` | `bool` | Present check |
| `IsAbsent()` / `IsNone()` | `bool` | Absent check |
| `Size()` | `int` | 1 or 0 |
| `Get()` | `(T, bool)` | Value plus presence |
| `MustGet()` | `T` | Value or panic; reserved for `mo.Do` and guaranteed presence |
| `OrElse(fallback)` | `T` | Value or fallback |
| `OrEmpty()` | `T` | Value or zero value |
| `ToPointer()` | `*T` | Pointer, nil when absent |

## Transforms

`Map` runs on present values and can demote to `None` through its boolean:

```go
opt := mo.Some(42)
doubled := opt.Map(func(v int) (int, bool) {
    return v * 2, true
}) // Some(84)

filtered := opt.Map(func(v int) (int, bool) {
    return v, v > 100
}) // None
```

`MapValue` transforms without the filter — its callback returns just `T`, so a present option stays present:

```go
doubled := mo.Some(42).MapValue(func(v int) int { return v * 2 })
```

`MapNone` fills an absent option:

```go
filled := mo.None[int]().MapNone(func() (int, bool) {
    return 42, true
}) // Some(42)
```

`FlatMap` chains Options of the same type:

```go
refreshed := findUser("123").FlatMap(func(u User) mo.Option[User] {
    return refreshUser(u)
})
```

**Type-change caveat:** `Map` and `FlatMap` require matching input/output types because Go methods cannot add type parameters. For `Option[int] → Option[string]` chains, use the sub-package (`option.Map`, `option.FlatMap`) or `mo.Do` — see [pipelines.md](./pipelines.md).

`Match` handles both branches; `ForEach` runs a side effect only when present:

```go
opt.Match(
    func(v int) (int, bool) { fmt.Println("got", v); return v, true },
    func() (int, bool) { return 0, false },
)
opt.ForEach(func(v int) { fmt.Println("value", v) })
```

## Equality

```go
mo.Some(42).Equal(mo.Some(42))       // true
mo.Some(42).Equal(mo.None[int]())    // false
mo.None[int]().Equal(mo.None[int]()) // true
```

## Serialization

`Option` implements the encoding interfaces out of the box:

| Interface | Wire behavior |
| --- | --- |
| `json.Marshaler` / `json.Unmarshaler` | Some → value, None → `null` |
| `encoding.TextMarshaler` / `TextUnmarshaler` | Text encoding |
| `encoding.BinaryMarshaler` / `BinaryUnmarshaler` | Binary encoding |
| `encoding/gob.GobEncoder` / `GobDecoder` | Gob encoding |

## Database support

`Option` is a `sql.Scanner` and `driver.Valuer`, so nullable columns scan and bind directly:

```go
type User struct {
    ID    int
    Phone mo.Option[string] // nullable column
}

err := row.Scan(&u.ID, &u.Phone)
_, err = db.Exec("INSERT INTO users (id, phone) VALUES ($1, $2)", u.ID, u.Phone)
```

## Go 1.24+ `omitzero`

`IsZero()` reports true for `None`, so `omitzero` works as expected — the field is dropped from JSON entirely instead of serializing `null`:

```go
type Response struct {
    Data  string            `json:"data"`
    Extra mo.Option[string] `json:"extra,omitzero"`
}
```