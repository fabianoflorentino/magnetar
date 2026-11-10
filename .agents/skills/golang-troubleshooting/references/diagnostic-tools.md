# Diagnostic Tools

## Runtime diagnostics via GODEBUG

### Reading Go docs

Use `go doc`, not `go tool doc` — Go 1.26 removed the old `cmd/doc` path. Go 1.27 adds versioned lookups (`go doc golang.org/x/tools/cmd/stringer@v0.30.0`) and `-ex` to list executable examples.

### GC tracing

```bash
GODEBUG=gctrace=1 ./app
```

Sample line:

```
gc 123 @45.67s 4%: 0.8+10+0.3 ms clock, 6+5/10/0 ms cpu, 512->300->150 MB
```

| Field | Meaning |
| --- | --- |
| `4%` | GC CPU overhead — sustained >10% signals over-allocation |
| `512->300->150 MB` | heap at GC start → heap at GC end → live heap |
| long pause | allocation storm |

### Scheduler tracing

```bash
GODEBUG=schedtrace=1000,scheddetail=1 ./app
```

| Signal | Meaning |
| --- | --- |
| `runqueue` high | CPU saturated, goroutines waiting to run |
| `idleprocs=0` | fully busy, at capacity |
| `spinningthreads` | threads spinning, often lock contention |
| `threads > gomaxprocs` | goroutines blocked in syscalls |

### GOTRACEBACK

Full stack traces on panic:

```bash
GOTRACEBACK=all ./app
```

| Level | Shows |
| --- | --- |
| `none` | no stack traces |
| `single` | current goroutine only (default) |
| `all` | every goroutine — use for deadlocks |
| `system` | every goroutine plus runtime frames |

## Escape analysis

Ask the compiler where values are allocated:

```bash
go build -gcflags=-m ./pkg/...
```

Lines like `moved to heap` or `escapes to heap` reveal allocations on paths you believed were stack-only — the usual starting point for a memory suspect.

## Execution tracer

Capture scheduler activity — goroutine lifecycle, blocking, and syscalls over a short interval:

```bash
go test -run TestSuspect -trace=trace.out ./pkg/...
go tool trace trace.out
```

Where pprof shows cost, the trace shows *why*: waits, wakeups, and scheduling stalls that no sample profile makes visible.

## Delve debugger

Install with `go install github.com/go-delve/delve/cmd/dlv@latest`.

```bash
dlv debug ./cmd/myapp        # run and debug a package
dlv test ./mypackage         # debug a test
dlv attach 12345             # attach to a running process
dlv exec ./myapp -- --flag=v # debug an existing binary
```

Interactive commands:

```
break main.main      # set breakpoint
break file.go:42     # break at a line
continue             # resume execution (c)
next                 # step over (n)
step                 # step into (s)
stepout              # step out of the frame
print variable       # inspect a value (p)
locals               # all local variables
args                 # function arguments
goroutines           # list all goroutines
goroutine 5          # switch to goroutine 5
stack                # show the stack
```

### IDE integration

VS Code (`.vscode/launch.json`):

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Package",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}",
      "env": { "GOTRACEBACK": "all" }
    }
  ]
}
```

GoLand: Run → Edit Configurations → Go Build; set breakpoints in the gutter and use the Debugger tab.

## Advanced analysis

Escape-analysis interpretation, assembly inspection, and compiler diagnostics (SSA dumps, inlining decisions) → the `fabianoflorentino/golang-agent-skills@golang-benchmark` skill. Execution-tracer analysis in depth → its `trace.md` reference.