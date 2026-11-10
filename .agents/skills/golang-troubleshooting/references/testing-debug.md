# Test-Driven Debugging

Write the failing test **before** the fix. A test that reproduces the bug is the fastest isolated environment you get: it pins the symptom, guards the fix, and doubles as a regression test.

## Capturing the bug in a test

```go
func TestBugDescription(t *testing.T) {
    svc := NewService(testConfig)        // setup: exact trigger conditions

    result, err := svc.Process(badInput) // act: the failing operation

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Status != "ok" {
        t.Errorf("got status %q, want %q", result.Status, "ok")
    }
}
```

## Probing the boundary with table tests

While debugging, expand the edge cases to find how far the bug reaches:

```go
tests := []struct {
    name    string
    input   string
    want    time.Duration
    wantErr bool
}{
    {"valid", "5s", 5 * time.Second, false},
    {"empty", "", 0, true},
    {"negative", "-1s", -time.Second, false},
    {"zero", "0s", 0, false},
    {"overflow", "99999999h", 0, true},
    {"whitespace", " 5s ", 0, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := ParseDuration(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got %v, want %v", got, tt.want)
        }
    })
}
```

## Test flags that matter for debugging

```bash
go test -v ./...                          # verbose output
go test -run TestName -v ./pkg/...        # one test
go test -count=1 ./...                    # disable the result cache
go test -timeout 10s ./...                # reveal hangs
go test -parallel 1 ./...                 # force sequential execution
go test -race ./...                       # race detector
go test -failfast ./...                   # stop at the first failure
go test -shuffle=on ./...                 # randomize test order (Go 1.17+)
go test -cover ./...                      # coverage summary
go test -coverprofile=c.out ./... && go tool cover -html=c.out
```

## Debugging flaky tests

A flaky test (passing and failing without code changes) is usually one of these five causes:

| Cause | Fix |
| --- | --- |
| Shared mutable state | reset globals in `TestMain` or `t.Cleanup` |
| Order dependence | each test sets up its own preconditions; check with `-run` isolation and `-shuffle=on` |
| Timing sensitivity | synchronize with channels/WaitGroups instead of `time.Sleep` |
| Port conflicts | bind `:0` and read the assigned port |
| Filesystem pollution | `t.TempDir()` for per-test directories |

Confirm the flake reliably before treating it:

```bash
go test -count=100 -run TestSuspect ./pkg/... -failfast
go test -parallel 1 -count=10 ./pkg/...
go test -shuffle=on ./pkg/...
```