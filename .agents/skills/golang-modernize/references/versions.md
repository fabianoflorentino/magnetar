# Go modernizations by release

This is the before/after catalog behind the priority guide. Every item is tagged with the minimum Go version it needs; the module's `go` directive decides which of these are legal today. Projects pinned to an older directive still get suggestions, just with narrower coverage — raise the `go` directive first when it lags. Release notes for each version: Go 1.21 <https://go.dev/doc/go1.21>, Go 1.22 <https://go.dev/doc/go1.22>, Go 1.23 <https://go.dev/doc/go1.23>, Go 1.24 <https://go.dev/doc/go1.24>, Go 1.25 <https://go.dev/doc/go1.25>, Go 1.26 <https://go.dev/doc/go1.26>, Go 1.27 <https://go.dev/doc/go1.27>.

## Go 1.21 (August 2023)

### Built-in `min`, `max`, `clear`

Drop hand-rolled comparisons. `min`/`max` accept any ordered type and variadic arguments; `clear` zeroes map entries and slice elements:

```go
// Before
func lower(a, b int) int {
    if a < b {
        return a
    }
    return b
}
best := lower(a, b)

// After (Go 1.21+)
best := min(a, b)
lowest := min(a, b, c, d)

// Before: clear a map by ranging
for k := range m { delete(m, k) }

// After (Go 1.21+)
clear(m)
```

### `log/slog` for structured logging

`slog` is the standard structured logger; new code should prefer it over `zap`, `logrus`, or `zerolog`. Existing projects invested in a third-party logger can migrate at their own pace.

```go
// Before: zap
logger, _ := zap.NewProduction()
logger.Info("request handled", zap.String("method", r.Method), zap.Int("status", status))

// Before: logrus
logrus.WithFields(logrus.Fields{"method": r.Method, "status": status}).Info("request handled")

// After (Go 1.21+): slog
slog.Info("request handled", "method", r.Method, "status", status)
// or type-safe
slog.Info("request handled", slog.String("method", r.Method), slog.Int("status", status))
```

The `samber/slog-*` handlers route slog output to various backends; `slog.DiscardHandler` (Go 1.24+) silences a logger entirely.

### `slices` instead of `sort` and manual loops

```go
// Before
sort.Strings(names)
sort.Slice(users, func(i, j int) bool {
    return users[i].Name < users[j].Name
})

// After (Go 1.21+)
slices.Sort(names)
slices.SortFunc(users, func(a, b User) int {
    return cmp.Compare(a.Name, b.Name)
})

// Before: hand-rolled membership loop
found := false
for _, v := range items {
    if v == target {
        found = true
        break
    }
}

// After (Go 1.21+)
found := slices.Contains(items, target)

// Before: manual clone via append
clone := append([]string(nil), original...)

// After (Go 1.21+)
clone := slices.Clone(original)
```

### `maps` package

```go
// Before
clone := make(map[string]int, len(original))
for k, v := range original {
    clone[k] = v
}

// After (Go 1.21+)
clone := maps.Clone(original)
```

### `sync.OnceFunc`, `sync.OnceValue`, `sync.OnceValues`

Turn a guarded lazy initializer into a one-liner:

```go
// Before
var (
    once   sync.Once
    client *http.Client
)
func getClient() *http.Client {
    once.Do(func() { client = &http.Client{Timeout: 10 * time.Second} })
    return client
}

// After (Go 1.21+)
var getClient = sync.OnceValue(func() *http.Client {
    return &http.Client{Timeout: 10 * time.Second}
})
```

### Enhanced `context` helpers

```go
ctx = context.WithoutCancel(parent)                          // detach from parent cancellation
ctx, cancel := context.WithTimeoutCause(parent, 5*time.Second, errTimeout)
ctx, cancel := context.WithDeadlineCause(parent, deadline, errDeadline)
stop := context.AfterFunc(ctx, func() { cleanup() })         // run when ctx is done
```

## Go 1.22 (February 2024)

### `range` over integers

```go
// Before
for i := 0; i < n; i++ { process(i) }

// After (Go 1.22+)
for i := range n { process(i) }
for range 10 { fmt.Println("hello") } // index unused
```

### Loop variable semantics

Go 1.22 gave each loop iteration its own variable; the `v := v` shadow copy is now dead weight. The new behavior applies only when `go.mod` says `go 1.22` or later.

```go
// Before (Go < 1.22)
for _, v := range items {
    v := v // shadow copy for the closure
    go func() { process(v) }()
}

// After (Go 1.22+)
for _, v := range items {
    go func() { process(v) }()
}
```

### `math/rand` → `math/rand/v2`

`math/rand/v2` needs no global seed, renamed `Intn` to `IntN` (and `Int63n` to `Int64N`), added the generic `rand.N[T]()`, switched to ChaCha8/PCG algorithms, and dropped `Read` — use `crypto/rand` for random bytes.

```go
// Before
import "math/rand"
rand.Seed(time.Now().UnixNano())
n := rand.Intn(100)

// After (Go 1.22+)
import "math/rand/v2"
n := rand.IntN(100)
```

### Stdlib HTTP routing

Go 1.22's `net/http` mux covers simple path-parameter routing without a framework:

```go
// Before: chi
r := mux.NewRouter()
r.HandleFunc("/users/{id}", getUser).Methods("GET")

// After (Go 1.22+)
mux := http.NewServeMux()
mux.HandleFunc("GET /users/{id}", getUser)

func getUser(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
}
```

### `strings.CutPrefix` / `strings.CutSuffix`

Two-value form removes the `HasPrefix` + `TrimPrefix` dance (available since Go 1.20, still widely missed):

```go
// Before
if strings.HasPrefix(s, "Bearer ") {
    token := strings.TrimPrefix(s, "Bearer ")
}

// After (Go 1.20+)
if token, ok := strings.CutPrefix(s, "Bearer "); ok {
    // use token
}
```

### `reflect.TypeFor[T]()`

```go
// Before
t := reflect.TypeOf((*MyInterface)(nil)).Elem()

// After (Go 1.22+)
t := reflect.TypeFor[MyInterface]()
```

### `database/sql.Null[T]`

The generic form replaces the per-type nullables:

```go
// Before
var name sql.NullString
var age  sql.NullInt64

// After (Go 1.22+)
var name sql.Null[string]
var age  sql.Null[int64]
```

### `cmp.Or` for defaults

```go
// Before
addr := os.Getenv("ADDR")
if addr == "" {
    addr = ":8080"
}

// After (Go 1.22+)
addr := cmp.Or(os.Getenv("ADDR"), ":8080")
```

## Go 1.23 (August 2024)

### Iterators (`range` over functions)

Go 1.23 introduced range-over-func with the `iter` package, enabling lazy, allocation-free iteration:

```go
// Before: materialize a slice
func AllUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT ...")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var users []User
    for rows.Next() {
        var u User
        rows.Scan(&u.ID, &u.Name)
        users = append(users, u)
    }
    return users, rows.Err()
}

// After (Go 1.23+): stream one row at a time
func AllUsers(db *sql.DB) iter.Seq2[User, error] {
    return func(yield func(User, error) bool) {
        rows, err := db.Query("SELECT ...")
        if err != nil {
            yield(User{}, err)
            return
        }
        defer rows.Close()
        for rows.Next() {
            var u User
            if err := rows.Scan(&u.ID, &u.Name); err != nil {
                yield(User{}, err)
                return
            }
            if !yield(u, nil) {
                return
            }
        }
        if err := rows.Err(); err != nil {
            yield(User{}, err)
        }
    }
}
```

### Iterator helpers in `slices` and `maps`

```go
for k := range slices.Sorted(maps.Keys(m)) { // sorted keys
    fmt.Println(k, m[k])
}
users := slices.Collect(maps.Values(userMap)) // iterator → slice

for chunk := range slices.Chunk(items, 100) { // batch processing
    processBatch(chunk)
}
```

### `unique` package for interning

`unique.Make` canonicalizes equal values so `==` becomes an identity check:

```go
handle := unique.Make(s) // Handle[string], comparable, memory-efficient
s = handle.Value()
```

### Timer/Ticker behavior

With `go 1.23` or later, `time.Timer` and `time.Ticker` are garbage collected without an explicit `Stop()`, and timer channels are unbuffered (capacity 0 instead of 1). Defer-based `Stop()` calls around short-lived timers can go.

## Go 1.24 (February 2025)

### Generic type aliases

```go
// Valid since Go 1.24
type Set[T comparable] = map[T]struct{}
type Result[T any] = struct {
    Value T
    Err   error
}
```

### `os.Root` for confined file access

Security-critical: `os.Root` blocks path-traversal (CWE-22) at the OS level. Replace `filepath.Clean` + `strings.HasPrefix` guards on user-supplied paths; it rejects symlinks escaping the root and supports `Open`, `Create`, `Stat`, `OpenFile`, `Mkdir`, `Remove`, and more.

```go
// Before: hand-rolled join + validation
path := filepath.Join(baseDir, userInput)
data, err := os.ReadFile(path)

// After (Go 1.24+)
root, err := os.OpenRoot("/opt/data")
if err != nil {
    return err
}
defer root.Close()
data, err := os.ReadFile("data", root.Open)
```

### `omitzero` JSON tag

More correct than `omitempty` for `time.Time`, `bool`, and custom zero-heavy types:

```go
// Before: zero time.Time is NOT omitted
type Event struct {
    At time.Time `json:"at,omitempty"`
}

// After (Go 1.24+)
type Event struct {
    At time.Time `json:"at,omitzero"`
}
```

### Iterator string splitting

`SplitSeq`, `FieldsSeq`, and `Lines` avoid the intermediate `[]string`:

```go
// Before: allocates
parts := strings.Split(csv, ",")
for _, part := range parts {
    process(part)
}

// After (Go 1.24+): lazy, zero-allocation
for part := range strings.SplitSeq(csv, ",") {
    process(part)
}
```

### `t.Context()` in tests

Cancels automatically when the test ends:

```go
// Before
func TestFoo(t *testing.T) {
    ctx := context.Background()
}

// After (Go 1.24+)
func TestFoo(t *testing.T) {
    ctx := t.Context()
}
```

### `b.Loop()` in benchmarks

```go
// Before
func BenchmarkFoo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        foo()
    }
}

// After (Go 1.24+)
func BenchmarkFoo(b *testing.B) {
    for b.Loop() {
        foo()
    }
}
```

### `runtime.AddCleanup` over `runtime.SetFinalizer`

Cleaner semantics and no cycle problems:

```go
// Before
runtime.SetFinalizer(obj, func(o *Object) { o.Close() })

// After (Go 1.24+)
runtime.AddCleanup(obj, func(resource Resource) { resource.Close() }, obj.resource)
```

### `weak` package

Weak references without finalizer games:

```go
import "weak"

ptr := weak.Make(obj)
if v := ptr.Value(); v != nil {
    // object still alive
}
```

### Crypto moved into the stdlib

`golang.org/x/crypto/{sha3,hkdf,pbkdf2}` are now `crypto/{sha3,hkdf,pbkdf2}` — a bare import-path swap:

```go
import "crypto/sha3"
import "crypto/hkdf"
import "crypto/pbkdf2"
```

### `tool` directives in `go.mod`

Replace `tools.go` blank imports with `tool` directives (Go 1.24+):

```bash
go get -tool golang.org/x/tools/cmd/stringer@latest
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go tool stringer -type=Kind
go tool golangci-lint run ./...
```

```go.mod
module example.com/project

go 1.26

tool (
    golang.org/x/tools/cmd/stringer
    github.com/golangci/golangci-lint/v2/cmd/golangci-lint
)
```

`go install tool` installs all module-pinned tools; `go get -u tool` updates them deliberately. (The `go` directive above is an example, not a requirement — do not change the module's `go` version just to add tools.)

### `fmt.Appendf` / `fmt.Appendln`

Older than most realize (Go 1.19+) and still often missed:

```go
// Before
buf = append(buf, fmt.Sprintf("count: %d", n)...)

// After (Go 1.19+)
buf = fmt.Appendf(buf, "count: %d", n)
```

## Go 1.25 (August 2025)

### `sync.WaitGroup.Go`

Fire-and-wait simplicity for goroutines that return nothing and must not panic:

```go
// Before
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    process()
}()
wg.Wait()

// After (Go 1.25+)
var wg sync.WaitGroup
wg.Go(func() {
    process()
})
wg.Wait()
```

### `testing/synctest` for concurrent tests

Deterministic concurrent-and-time tests via a fake clock. In Go 1.25+ use `synctest.Test`/`synctest.Wait` — the Go 1.24 experimental `synctest.Run` API is superseded.

```go
// After (Go 1.25+)
func TestConcurrent(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        var count atomic.Int32
        go func() { count.Add(1) }()
        synctest.Wait() // until all goroutines park
        if count.Load() != 1 {
            t.Fatal("expected 1")
        }
    })
}
```

### `runtime/trace.FlightRecorder`

Always-on ring-buffer tracing for production:

```go
fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{})
if err := fr.Start(); err != nil {
    return err
}
// on error, dump recent trace data:
fr.WriteTo(file)
```

### Container-aware `GOMAXPROCS`

Go 1.25 respects cgroup CPU limits on Linux automatically. Drop `go.uber.org/automaxprocs`:

```go
// Before
import _ "go.uber.org/automaxprocs"

// After (Go 1.25+): remove the import; GOMAXPROCS comes from the cgroup
```

### `encoding/json/v2` (experimental)

The JSON revision was experimental behind `GOEXPERIMENT=jsonv2` in 1.25-1.26 and became the default in Go 1.27 — see the Go 1.27 section for the stable API and migration hazards.

### Smaller 1.25 additions to prefer

- `net/http.CrossOriginProtection` — stdlib helper for cross-origin/CSRF-style protection.
- `reflect.TypeAssert[T](v)` — prefer over `v.Interface().(T)`.
- `os.Root.FS` and extra `os.Root` methods — confined filesystem APIs.
- New vet checks flag `waitgroup` misuse and hand-formatted host:port strings — use `net.JoinHostPort`.

## Go 1.26 (February 2026)

### `errors.AsType[T]()`

Genericized error unwrapping:

```go
// Before
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    fmt.Println(pathErr.Path)
}

// After (Go 1.26+)
if pathErr, ok := errors.AsType[*os.PathError](err); ok {
    fmt.Println(pathErr.Path)
}
```

### Value-initializing `new(expr)`

`new` now accepts an expression, not just a type, and returns a pointer to the computed value:

```go
// Before: helper function
func ptr[T any](v T) *T { return &v }
cfg := Config{Timeout: ptr(30)}

// After (Go 1.26+)
cfg := Config{Timeout: new(30)} // *int initialized to 30, not 0
```

### `crypto/hpke`

Hybrid Public Key Encryption (RFC 9180) lands in the standard library.

### Prefer OAEP/HPKE over new PKCS#1 v1.5 encryption

New encryption should use `rsa.EncryptOAEP`/`rsa.EncryptOAEPWithOptions` or a modern KEM/HPKE design rather than `crypto/rsa.EncryptPKCS1v15`.

### Green Tea GC on by default

Re-measure GC and allocation tuning under the new collector; drop legacy tuning only when profiles support it. Keep `GOMEMLIMIT` where it reflects a real service memory ceiling, and drop `automaxprocs` unless measured — Go 1.25+ is container-aware.

### Test artifacts

`t.ArtifactDir()`, `b.ArtifactDir()`, and `f.ArtifactDir()` give tests, benchmarks, and fuzzers a place to persist files for inspection.

### `slog.NewMultiHandler`

Simple fan-out to several handlers without a third-party composer.

### `httputil.ReverseProxy{Rewrite: ...}`

Prefer `Rewrite` over `Director` for new proxies:

```go
proxy := &httputil.ReverseProxy{
    Rewrite: func(pr *httputil.ProxyRequest) {
        pr.SetURL(targetURL)
        pr.SetXForwarded()
    },
}
```

### Smaller 1.26 API preferences

- `bytes.Buffer.Peek(n)` — inspect upcoming bytes without consuming them.
- Reflect iterators (`Type.Fields()`, `Type.Methods()`, `Type.Ins()`, `Type.Outs()`, `Value.Fields()`, `Value.Methods()`) replace `NumField`/`Field(i)` loops where clearer.

### Goroutine leak profile (experimental)

Go 1.26 ships an experimental goroutine leak profile gated by `GOEXPERIMENT=goroutineleakprofile`; it becomes a standard profile in Go 1.27 without the flag.

### Command-line and module notes

- Use `go doc`, not `go tool doc` — the old `cmd/doc` path was removed in 1.26.
- With a 1.26+ toolchain, `go mod init` may still write an older default `go` directive. If the project intentionally targets 1.26+ APIs, set the directive deliberately (`go mod edit -go=1.26 && go mod tidy`); never use APIs newer than the directive until the module agrees to upgrade.
- `go fix` was rewritten to apply a subset of modernize-style analyzers automatically. `go fix ./...` applies the enabled safe transformations; `go tool fix help` lists exact coverage.

## Go 1.27 (August 2026)

### Generic methods

Go 1.27 lifts the pre-generics restriction ([go.dev/issue/77273](https://go.dev/issue/77273), [spec: Method declarations](https://go.dev/ref/spec#Method_declarations)): a method can declare its own type parameters, so a helper that serves one type need not live at package scope. Interface methods still cannot declare type parameters, and a generic method cannot satisfy an interface — keep the package-level function when the operation must fit an interface contract.

```go
// Before: package-scope function, disconnected from the type
func FilterInts[T any](s []T, pred func(T) bool) []T { /* ... */ }

// After (Go 1.27+): method scoped to the receiver
func (s Set[T]) Filter[U comparable](pred func(T) U) Set[T] { /* ... */ }
```

The stdlib reference is `(*rand.Rand).N[Int intType](n Int) Int` in `math/rand/v2`.

### `strings.CutLast` / `bytes.CutLast`

Three-value split on the final separator, no index arithmetic:

```go
// Before
if i := strings.LastIndex(path, "/"); i >= 0 {
    dir, file := path[:i], path[i+1:]
}

// After (Go 1.27+)
if dir, file, ok := strings.CutLast(path, "/"); ok {
    // dir, file
}
```

`bytes.CutLast(b, sep []byte) (before, after []byte, found bool)` mirrors the string form.

### `net/url` clone methods

`Values.Clone()` deep-copies the backing slices; `URL.Clone()` produces a shareable copy:

```go
clone := original.Clone()
u2 := u.Clone()
```

### `math/big.Int.Divide`

Rounding-mode division replaces the sign-correction dance:

```go
// Before: floor/ceil correction
q, r := new(big.Int).QuoRem(x, y, new(big.Int))
if r.Sign() != 0 && (r.Sign() < 0) != (y.Sign() < 0) {
    q.Sub(q, big.NewInt(1))
}

// After (Go 1.27+)
q, r := new(big.Int).Divide(x, y, new(big.Int), big.Floor)
// modes: big.Trunc, big.Floor, big.Round, big.Ceil
```

### Stdlib `uuid`

Drop the dependency for plain uses:

```go
// Before
import "github.com/google/uuid"
id := uuid.NewString()

// After (Go 1.27+)
import "uuid"
id := uuid.New().String()
```

`uuid.New()`, `uuid.NewV4()`, and `uuid.NewV7()` return values without errors; v7 UUIDs are time-ordered, so prefer `uuid.NewV7()` for database primary keys. Before removing the dependency, check `go mod why -m github.com/google/uuid` — v3/v5 namespace UUIDs, `sql.Scanner`/`driver.Valuer` integration, and other RFC-specific variants are not yet in the stdlib package.

### `encoding/json/v2` is the default

Stable since Go 1.27 (`encoding/json` becomes a thin wrapper over it, and unmarshal is significantly faster). For new code, use the v2 API directly:

```go
// After (Go 1.27+)
import "encoding/json/v2"
data, err := json.Marshal(v)        // same names, v2 semantics
err = json.UnmarshalRead(r, &v)     // stream straight from an io.Reader
```

Use `encoding/json/jsontext` (`Encoder`, `Decoder`, `Token`, `Value`) for syntactic, streaming-level JSON work instead of hand-rolled `json.RawMessage` juggling.

**Migrate deliberately — the default got stricter:**

- Duplicate object member names are rejected; v1 silently kept the last one.
- Invalid UTF-8 in JSON strings is rejected; v1 replaced it silently.
- The `format` and `unknown` struct tags, `DiscardUnknownMembers`, and `SkipFunc` are gone.
- The `inline` tag is renamed `embed`.
- `GOEXPERIMENT=nojsonv2` is an escape hatch for rollback, not a long-term stance.

### `synctest.Sleep` inside a bubble

Advances the fake clock deterministically:

```go
synctest.Test(t, func(t *testing.T) {
    go worker()
    synctest.Sleep(time.Second) // bubble's virtual clock
})
```

### `httptest.NewTestServer`

In-memory fake network that composes with `testing/synctest`:

```go
synctest.Test(t, func(t *testing.T) {
    srv := httptest.NewTestServer(handler)
    defer srv.Close()
})
```

(The previous `httptest.NewServer` binds a real socket, forcing real goroutines and timers.)

### `goroutineleak` profile, generally available

The leak profile from Go 1.26's experiment is now a standard `runtime/pprof` profile, also served at `/debug/pprof/goroutineleak` — no build flag required. It reports goroutines blocked on a concurrency primitive that can never unblock; leaks reachable from global variables or still-runnable goroutines are not detected. For leak investigations, → See the `golang-concurrency` and `golang-troubleshooting` skills.

### `go fix` additions

New analyzers: `atomictypes`, `embedlit`, `slicesbackward`, `unsafefuncs`. `waitgroup` was renamed `waitgroupgo`; `fmtappendf` was removed. Run `go fix ./...` after upgrading the toolchain — full command reference in [tooling.md](tooling.md).

### `stdversion` vet check in `go test`

`go test` now flags uses of stdlib symbols newer than the file's effective Go version (the `go` directive plus build tags). If CI starts failing after a toolchain upgrade, bump the directive or gate the newer API behind build tags — do not silence the check.

### `go mod tidy` block layout

For modules declaring `go 1.27` or later, `go mod tidy` consolidates duplicate `require` blocks into the standard two-block layout (direct, then indirect), preserving existing comments. Run it once after a bump to clean up merge-conflict remnants.

### Smaller 1.27 API preferences

- `hash/maphash.Hasher` and `maphash.ComparableHasher` — contracts between a type and future hash-based structures (hash tables, Bloom filters).
- `database/sql.ConvertAssign` and `driver.RowsColumnScanner` — for database driver authors.
- `runtime/secret.Do` — goroutines started in secret mode inherit secret mode.

## Go 1.27+ version-bump risk checklist (verify, don't rewrite)

These need a review pass on the way to `go 1.27` — none require a code rewrite, but skipping them risks a build failure or a silent behavior change:

- **Removed `GODEBUG` settings** — `asynctimerchan`, `tlsunsafeekm`, `tlsrsakex`, `tls3des`, `tls10server`, `x509keypairleaf`, `gotypesalias`. A `godebug` line in `go.mod` or a `//go:debug` comment pinning one of these to its old value now **fails the build**; pinning to the current default value is accepted. Grep with `grep -rn 'go:debug\|godebug' go.mod **/*.go`.
- **json/v2 strictness** — re-run integration tests against real payloads, not just unit tests, before the bump ships.
- **Size-specialized allocator** — up to 30% faster allocations under 80 bytes, roughly 1% faster overall, at the cost of ~60 KB binary size. On by default; `GOEXPERIMENT=nosizespecializedmalloc` disables it, but the flag is scheduled for removal in Go 1.28.
- **Darwin floor raised to macOS 13 (Ventura)** — binaries built with this toolchain no longer run on older macOS.
- **`linux/ppc64` now builds ELFv2 binaries** and requires Linux kernel 3.13+ (RHEL 7's 3.10 kernel with backports) — relevant only to ppc64 deployments.
- **`bzr` support removed** from the `go` command — irrelevant unless a module vendors from Bazaar.
- **Tracebacks include `runtime/pprof` goroutine labels** for `go 1.27+` modules by default; disable with `GODEBUG=tracebacklabels=0` if labels leak sensitive data into crash output.

## General modernization (any version, still commonly missed)

### `any`, not `interface{}`

```go
func process(data any) any { /* ... */ }
```

### Generics over `interface{}` + type assertions

```go
// Before
func Contains(slice []interface{}, item interface{}) bool { /* ... */ }

// After (Go 1.18+)
func Contains[T comparable](slice []T, item T) bool { /* ... */ }
// Or, on Go 1.21+: slices.Contains
```

### `errors.Join` over multi-error libraries

```go
// Before: hashicorp/go-multierror or uber-go/multierr
errs = multierror.Append(errs, err1)

// After (Go 1.20+)
return errors.Join(err1, err2)
```

### `net.JoinHostPort`

Correct IPv6 bracketing where `fmt.Sprintf` breaks:

```go
// Before: produces "::1:8080" for IPv6 — wrong
addr := fmt.Sprintf("%s:%d", host, port)

// After (correct: [::1]:8080)
addr := net.JoinHostPort(host, strconv.Itoa(port))
```
