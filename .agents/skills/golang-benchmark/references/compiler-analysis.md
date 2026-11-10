# Compiler Analysis

`go build` diagnostic flags reveal the compiler's optimization decisions — where values escape to the heap, why a function is or is not inlined, the SSA transformation passes, and the final assembly. Reach for these when pprof names a hot function and you need to know *why* it allocates or *why* it will not inline. Analysis happens at build time, so it is free at runtime.

## Escape analysis

Escape analysis decides whether a value lives on the stack (reclaimed when the function returns) or must be heap-allocated (GC-managed). "Escapes to heap" means the compiler could not prove the value stays put.

### Commands

```bash
# Allocation and inlining decisions, one line per decision
go build -gcflags="-m" ./... 2>&1 | grep "escapes to heap"
go build -gcflags="-m" ./... 2>&1 | grep "moved to heap"

# Verbose: spells out the reason for each decision
go build -gcflags="-m -m" ./...

# Narrow to one package or file
go build -gcflags="-m" ./pkg/parser 2>&1 | grep "escapes"
go build -gcflags="-m" ./pkg/parser/parse.go 2>&1

# Include every dependency (noisy; only when debugging an odd case)
go build -gcflags="all=-m" ./...

# What survives on the stack
go build -gcflags="-m" ./pkg/parser 2>&1 | grep "does not escape"
```

### Reading the output

```
./pkg/parser/parse.go:15:6: can inline Parse
./pkg/parser/parse.go:42:13: &result escapes to heap
./pkg/parser/parse.go:42:13:   flow: ~r0 = &result:
./pkg/parser/parse.go:42:13:     from &result (address-of) at ./pkg/parser/parse.go:42:13
./pkg/parser/parse.go:42:13:     from return &result (return) at ./pkg/parser/parse.go:42:6
```

The `-m -m` form prints the escape chain: here `result` has its address taken and that pointer is returned, so it must outlive the call and moves to the heap.

### Common escape causes

| Cause | Shape | Why it escapes |
| --- | --- | --- |
| Returning a local's address | `return &result` | The value must outlive the function |
| Interface boxing | `var x any = myStruct` | A concrete value in an interface is copied to the heap |
| Closure capture | `go func() { use(local) }()` | The closure may run after the caller returns |
| Slice growth | `append` at capacity | New backing array allocation |
| Pointer into an opaque call | `json.Marshal(&data)` | The compiler cannot prove the callee drops it |
| Storing into an escaping field | `obj.Field = &local` | Heap containers force their contents to the heap |
| `fmt.Sprintf` and friends | `fmt.Sprintf("%d", n)` | Arguments box into `any`; the string itself allocates |
| Sending a pointer on a channel | `ch <- &data` | Receiving goroutine has an independent lifetime |

Not every escape is a problem. Investigate only escapes inside functions the heap profile flags as allocation-heavy; a startup path may allocate freely.

## Inlining

Inlining replaces a call with the callee's body at the call site, removing call overhead and enabling follow-on optimization — better escape analysis, constant folding, dead-code elimination.

### Commands

```bash
# What can and cannot be inlined, with reasons
go build -gcflags="-m" ./... 2>&1 | grep "can inline"
go build -gcflags="-m" ./... 2>&1 | grep "cannot inline"

# Where inlining was actually applied
go build -gcflags="-m" ./... 2>&1 | grep "inlining call to"

# Cost budget and blocking reason
go build -gcflags="-m -m" ./... 2>&1 | grep "inline"

# Both concerns at once
go build -gcflags="-m" ./pkg/handler 2>&1 | grep -E "(inline|escape|moved to heap)"
```

### Reading the output

```
./pkg/handler/handler.go:20:6: can inline validateInput
./pkg/handler/handler.go:35:6: cannot inline HandleRequest: function too complex: cost 120 exceeds budget 80
./pkg/handler/handler.go:42:19: inlining call to validateInput
```

The inline budget is measured in AST-node cost, roughly 80 nodes on Go 1.22 and raised since. `-m -m` prints the actual cost and threshold for the toolchain in use.

### Common inlining blockers

| Blocker | Why | Mitigation |
| --- | --- | --- |
| Function too complex | Body cost exceeds the budget | Split it; keep the hot core small |
| `defer` | Adds cleanup the inline pass cannot fold | Drop defer from tiny hot functions |
| `recover()` | Forceps a preserved stack frame | Wrapper function around it |
| `go` statement | Implicit goroutine-launch complexity | Extract the goroutine body |
| Type switch / interface dispatch | Runtime dispatch defeats compile-time analysis | Concrete types in hot paths |
| `select` | Complex runtime interaction | Simplify channel patterns |
| Large body | Many statements accumulate cost | Extract cold paths so the inner hot function inlines |

Receiver choice is not a lever here: pointer-receiver methods inline too, and value receivers do not guarantee inlining. Verify with `-m -m` rather than guessing.

## SSA dump

`GOSSAFUNC` writes `ssa.html`, a pass-by-pass view of the compiler's Static Single Assignment representation — where dead code is eliminated, bounds checks removed, constants folded, and registers assigned.

```bash
# HTML dump for one function
GOSSAFUNC=Parse go build ./pkg/parser

# A method
GOSSAFUNC='(*Parser).Parse' go build ./pkg/parser

# Disambiguate a name
GOSSAFUNC=myapp/pkg/parser.Parse go build ./...

# Choose the output directory
GOSSAFUNC=Parse GOSSADIR=/tmp/ssa go build ./pkg/parser
# -> /tmp/ssa/ssa.html
```

The file walks the stages — source, AST, initial SSA (`Start`), the optimization pass (`Opt`), architecture lowering (`Lower`), register allocation (`Regalloc`), final codegen (`Genssa`). Clicking a value highlights it across every pass: red values were eliminated as dead, green were introduced.

What to look for:

- **Surviving bounds checks** — `IsInBounds`/`IsSliceInBounds` ops not eliminated; explicit length guards can help the pass.
- **Dead code that persists** — computed-but-unused values suggest a hidden side effect.
- **Missed constant folding** — constant expressions resolving at compile time is the expected outcome.
- **Register spills** — values forced to the stack under register pressure.

## Assembly output

Inspect the actual machine code: SIMD use, leftover bounds checks, calls that should not be there.

```bash
# Package assembly (verbose — head it)
go build -gcflags="-S" ./pkg/parser 2>&1 | head -200

# One function
go build -gcflags="-S" ./pkg/parser 2>&1 | grep -A 50 '"".Parse'

# Dependencies included
go build -gcflags="all=-S" ./... 2>&1 | grep -A 50 'myapp/pkg/parser.Parse'

# Disassemble a binary instead
go build -o myapp ./cmd/server
go tool objdump -s Parse myapp
go tool objdump -S -s Parse myapp         # interleaved with source
go tool objdump -start 0x4a3b00 -end 0x4a3c00 myapp

# List symbols
go tool nm myapp | grep Parse

# Peek at another architecture
GOARCH=arm64 go build -gcflags="-S" ./pkg/parser 2>&1 | head -200
```

Reading it:

```
"".Parse STEXT size=240 args=0x18 locals=0x48
    0x0000 MOVQ (TLS), CX           ; goroutine stack check
    0x0009 LEAQ -64(SP), AX
    0x000e CMPQ AX, 16(CX)          ; stack overflow check
    0x0012 JLS  228                 ; jump to stack growth
    0x0018 SUBQ $72, SP             ; allocate stack frame
    0x001c MOVQ BP, 64(SP)          ; save base pointer
    0x0021 LEAQ 64(SP), BP          ; set new base pointer
    ; ... function body ...
    0x00e0 CALL runtime.makeslice(SB) ; heap allocation in the hot path
```

Watch for:

- `CALL runtime.makeslice` / `runtime.newobject` — allocations inside the hot path.
- `CALL runtime.growslice` — slice growth triggering a copy.
- `CMPQ` + `JCC` bounds-check sequences that could be hoisted.
- SIMD instructions (`VMOVDQU`, `VPSHUFB`, `VPADDB`) — confirming vectorized code.
- `CALL runtime.morestack_noctxt` — stack growth; frequent calls mean deep recursion.
- Ignore `PCDATA`/`FUNCDATA` — GC metadata, irrelevant to performance.

### Before/after diffing

```bash
go build -gcflags="-S" ./pkg/parser 2>&1 > asm-before.txt
go build -gcflags="-S" ./pkg/parser 2>&1 > asm-after.txt
diff asm-before.txt asm-after.txt
```