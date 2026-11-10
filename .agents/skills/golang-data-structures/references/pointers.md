# Pointer Types

Go provides several pointer types for different use cases: regular pointers (`*T`) for normal references, `unsafe.Pointer` for low-level memory manipulation, and `weak.Pointer[T]` (Go 1.24+) for references that don't prevent garbage collection.

## Regular pointers (`*T`)

### Stack vs heap (escape analysis)

Go's compiler decides whether to allocate on the stack or heap. A variable "escapes" to the heap when its lifetime extends beyond the function.

```go
func noEscape() int {
    x := 42
    return x
}

func escapes() *int {
    x := 42
    return &x
}
```

Use `go build -gcflags="-m"` to see escape analysis decisions. Heap allocations add GC pressure — avoid unnecessary escapes in hot paths.

### `new(T)` vs `&T{}`

Both allocate and return a pointer. `&T{}` is preferred because it allows field initialization.

```go
p := new(Point)
p := &Point{X: 1}
```

## `unsafe.Pointer`

`unsafe.Pointer` bypasses Go's type system for FFI and low-level memory manipulation. Only the 6 patterns from the Go spec are safe; any other pattern is undefined behavior.

### The 6 valid patterns

**Pattern 1: Convert `*T` to `*U` via `unsafe.Pointer`**

```go
f := 1.5
bits := *(*uint64)(unsafe.Pointer(&f))
```

**Pattern 2: Convert `unsafe.Pointer` to `uintptr` and back (same expression)**

```go
p := unsafe.Pointer(uintptr(unsafe.Pointer(&s.field)) + offset)
```

**Pattern 3: `reflect.Value.Pointer()` or `UnsafeAddr()` to `unsafe.Pointer`**

```go
p := unsafe.Pointer(reflect.ValueOf(&x).Pointer())
```

**Pattern 4: `syscall.Syscall` arguments**

```go
syscall.Syscall(SYS_READ, fd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
```

### Critical rule: never store `uintptr` across statements

```go
// ✗ DANGEROUS — GC can move the object between these two lines
u := uintptr(unsafe.Pointer(&x))
p := unsafe.Pointer(u)

// ✓ Safe — single expression
p := unsafe.Pointer(uintptr(unsafe.Pointer(&x)) + offset)
```

### Modern alternatives (prefer these)

| Function | Since | Purpose |
| --- | --- | --- |
| `unsafe.Add(ptr, len)` | Go 1.17 | Pointer arithmetic without `uintptr` conversion |
| `unsafe.Slice(ptr, len)` | Go 1.17 | Create slice from pointer + length |
| `unsafe.String(ptr, len)` | Go 1.20 | Create string from pointer + length |
| `unsafe.SliceData(s)` | Go 1.17 | Get pointer to slice's backing array |
| `unsafe.StringData(s)` | Go 1.20 | Get pointer to string's backing array |

These are safer than manual `uintptr` arithmetic because they keep values as pointers (visible to GC) throughout.

## `weak.Pointer[T]` (Go 1.24+)

A weak pointer holds a reference to an object without preventing garbage collection. When the GC reclaims the object, `Value()` returns `nil`.

```go
strong := new(MyType)
w := weak.Make(strong)

if p := w.Value(); p != nil {
    // object still alive
} else {
    // object was garbage collected
}
```

Use for deduplication caches (intern equivalent values without preventing GC) or automatic cache eviction (cached objects evict when no strong references remain).

### `runtime.AddCleanup` vs `runtime.SetFinalizer`

Prefer `runtime.AddCleanup` (Go 1.24+) over `runtime.SetFinalizer`: multiple cleanups can be registered per object, cleanup function receives a value not a pointer to the collected object, no risk of resurrecting the object, and works correctly with weak pointers.
