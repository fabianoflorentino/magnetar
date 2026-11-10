# Memory Safety Rules

Go's memory safety is a feature, not a given - the escape hatches and unchecked arithmetic are where exploits begin. Rules:

1. Check integer arithmetic that touches external input for overflow at the boundary.
2. Avoid `unsafe` in application code; confine it to low-level libraries under review.
3. Detect data races with `-race` in CI.

## Integer overflow - High

Wrapping arithmetic on attacker-sized values turns a norm into a negative, or a buffer into a slice OOB. Validate before computing:

```go
func allocateBuffer(rows, cols int) ([]byte, error) {
    if rows <= 0 || cols <= 0 || rows > math.MaxInt/cols {
        return nil, errors.New("buffer size overflow")
    }
    size := rows * cols
    const maxBufferSize = 100 * 1024 * 1024
    if size > maxBufferSize {
        return nil, errors.New("buffer exceeds limit")
    }
    return make([]byte, size), nil
}
```

The classic `rows * cols` allocation is the pattern to hunt: multiply into a large type, or check the division inverse (`result/cols != rows`) when the operands are already trusted.

## math/big.Rat blowup - Low

`big.Rat` keeps exact fractions, and denominators can balloon to thousands of bits with innocent-looking loops. Bound the precision when numerators or denominators come from untrusted sources:

```go
const maxBits = 1000

func safeFraction(n int) (*big.Rat, error) {
    r := big.NewRat(1, 1)
    for i := 0; i < n; i++ {
        r.Mul(r, big.NewRat(int64(i+1), int64(i+2)))
        if r.Num().BitLen() > maxBits || r.Denom().BitLen() > maxBits {
            return nil, errors.New("fraction precision exceeded")
        }
    }
    return r, nil
}
```

## Memory aliasing - Medium

Two slices sharing a backing array can corrupt each other mid-iteration. Before destructive in-place work, decide whether overlap is possible:

```go
func checkOverlap(a, b []byte) bool {
    if len(a) == 0 || len(b) == 0 {
        return false
    }
    aStart := uintptr(unsafe.Pointer(&a[0]))
    aEnd := aStart + uintptr(len(a))
    bStart := uintptr(unsafe.Pointer(&b[0]))
    bEnd := bStart + uintptr(len(b))
    return aStart < bEnd && bStart < aEnd
}

func safeCopy(dest, src []byte) {
    if checkOverlap(dest, src) {
        dest = append(dest[:0], src...) // staged through a fresh allocation
    } else {
        copy(dest, src)
    }
}
```

Prefer `append(dest[:0], src...)` and `slices.Concat` over manual overlap reasoning in new code.

## Misuse of unsafe - High

`unsafe` bypasses type and memory safety - one bad offset reads or writes out of bounds with no runtime guard. String-to-`[]byte` via `reflect.StringHeader`, and float-from-bits type punning, are classic footguns:

```go
// Bad - aliasing a string's memory as a mutable []byte
b := unsafe.StringData(s)

// Good - a real (copying) conversion
b := []byte(s)
```

For numeric reinterpretation use `encoding/binary` and `math.Float64frombits`, which are well-defined and portable:

```go
var buf [8]byte
binary.LittleEndian.PutUint64(buf[:], value)
f := math.Float64frombits(binary.LittleEndian.Uint64(buf[:]))
```

## Data races - High

Races corrupt in-memory state and can smuggle a past-due authz decision past the check. Lock, or go atomic; the race detector is the enforcement:

```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

// atomic for single-counter cases
var value atomic.Int64
value.Add(1)
```

## Always run the race detector

```bash
go test -race ./...
```

Run it routinely, not only on suspicion - races that appear benign today become the production outage next month.

## CWE references

- CWE-190 - integer overflow or wraparound
- CWE-119 - improper restriction of operations within bounds
- CWE-125 - out-of-bounds read
- CWE-787 - out-of-bounds write
- CWE-362 - race condition
- CWE-367 - time-of-check time-of-use (TOCTOU)
