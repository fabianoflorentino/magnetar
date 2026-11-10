---
name: golang-pitfalls-control-structures
description: "Golang control flow — range loop value copies, single evaluation of range arguments, per-iteration loop variables, wrong assumptions during map iteration, break/continue scoping with labels, defer inside loops, and deterministic select draining. Distilled from mistakes #30-35 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang loops, switch, select, labeled statements, and defer usage."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔀"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Control Structures

Source material: mistakes #30-35 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when writing Go loops and control flow.

## 30. Elements are copied in `range` loops (#30)

- The `value` element of a `range` loop is a **copy**. Mutating it does not mutate the collection element (unless the element/field is a pointer).
- To mutate structs in a slice: access via index `s[i]` (range or classic `for`).

```go
for i := range customers {
    customers[i].age = age // works
}
for _, c := range customers {
    c.age = age // no-op on the actual element
}
```

## 31. How arguments are evaluated in `range` loops (#31)

- The expression passed to `range` is evaluated **only once**, before the loop, by copying it (whatever the type — including arrays).
- Mutations made during iteration to an array are not seen by the copied range target:

```go
a := [3]int{0, 1, 2}
for i, v := range a {
    a[2] = 10 // the loop still iterates over the original copy
    // at i==2, v is still 2, not 10
}
```

- This also matters for channel iteration: the channel is evaluated once; assignments elsewhere don't change the iterator.

## 32. Pointer elements in `range` loops (#32)

- Not relevant anymore from Go 1.22 — per-iteration loop variables are now used (`range` variables are new each iteration). No re-binding workaround is needed on Go 1.22+.

## 33. Wrong assumptions during map iterations (#33)

- A map:
  - does not order keys,
  - does not preserve insertion order,
  - has no deterministic iteration order,
  - does not guarantee an element added during iteration will be produced during that iteration.
- If predictable output is required, sort keys first or use an ordered data structure.

## 34. Ignoring how `break` works (#34)

- `break` (and `continue`) terminate the **innermost** `for`, `switch`, or `select`.
- A `break` inside a `switch` inside a `for` breaks the `switch`, not the loop.
- Use a label to break a specific statement:

```go
loop:
    for i := 0; i < 5; i++ {
        switch i {
        case 2:
            break loop
        }
    }
```

## 35. Using `defer` inside a loop (#35)

- `defer` schedules execution when the **surrounding function** returns, not at each loop iteration.
- `defer file.Close()` inside a loop keeps every file descriptor open until the function returns — resource leak.
- Fix: extract the loop body into its own function (or a closure called each iteration) so `defer` runs per iteration.

```go
for path := range ch {
    if err := readFile(path); err != nil { // readFile uses defer internally
        return err
    }
}
```

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-data-structures` for slice, map, and channel iteration semantics
- → See `fabianoflorentino/golang-agent-skills@golang-concurrency` for `select` and channel draining patterns
- → See `fabianoflorentino/golang-agent-skills@golang-modernize` for Go 1.22+ per-iteration loop variables
