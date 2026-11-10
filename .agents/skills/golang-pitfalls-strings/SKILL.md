---
name: golang-pitfalls-strings
description: "Golang strings and bytes — the rune concept, len(s) returning bytes not runes, inaccurate string iteration over rune start indices, misusing trim functions, under-optimized string concatenation with strings.Builder, useless string to byte conversions, and substring memory leaks. Distilled from mistakes #36-41 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang code that manipulates strings, runes, or bytes."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧵"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Strings & Bytes

Source material: mistakes #36-41 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when manipulating strings in Go.

## 36. Not understanding the concept of rune (#36)

- A charset is a set of characters; an encoding translates characters to binary.
- A Go string references an immutable slice of arbitrary bytes. Source-code literals are UTF-8, but strings from elsewhere may not be.
- A `rune` is a Unicode code point, encoded in UTF-8 using 1 to 4 bytes.
- **`len(s)` returns the number of bytes, not runes.** (`"hêllo"` has 5 runes but `len` == 6.)

## 37. Inaccurate string iteration (#37)

- `for i := range s` iterates over the **starting byte index of each rune**, not each rune.
- `s[i]` returns a single byte — printing it corrupts multi-byte runes (e.g., `ê` prints as `Ã`).
- To print all runes, use the value element: `for i, r := range s { ... }`.
- To access the ith rune, convert to `[]rune`: `[]rune(s)[i]`. This conversion costs O(n) — avoid it in hot loops; prefer `range` with the value when iterating everything.

## 38. Misusing trim functions (#38)

- `strings.TrimRight(s, cutset)` / `strings.TrimLeft` remove all trailing/leading runes **contained in the cutset**.
  - `strings.TrimRight("123oxo", "xo")` → `"123"`.
- `strings.TrimSuffix(s, suffix)` / `strings.TrimPrefix(s, prefix)` remove only the exact provided suffix/prefix (a single occurrence).
- Pick the right function for the intent.

## 39. Under-optimized string concatenation (#39)

- Strings are immutable; `s += value` in a loop reallocates a new string every iteration — slow.
- Use `strings.Builder`:

```go
sb := strings.Builder{}
for _, value := range values {
    sb.WriteString(value)
}
return sb.String()
```

- Preallocate with `sb.Grow(totalLen)` when total length is known — ~78% faster than without, ~99% faster than `+=` in benchmarks.
- `strings.Builder` is not safe for concurrent use. For concatenating just a few strings, `+=` or `fmt.Sprintf` is clearer and fine.

## 40. Useless string conversions (#40)

- Most I/O works with `[]byte` (`io.Reader`, `io.Writer`, `io.ReadAll`), not strings.
- The `bytes` package matches every `strings` operation (`Split`, `Count`, `Contains`, `Index`, ...).
- Consider implementing the whole workflow with `[]byte` to avoid repeated string↔byte conversions.

## 41. Substring and memory leaks (#41)

- A substring shares the backing array with the original string — keeping the substring keeps the whole string alive.
- Substring indexes are byte-based, not rune-based.
- To release memory, copy the substring manually or use `strings.Clone` (Go 1.18+).

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-performance` for allocation-aware string building
- → See `fabianoflorentino/golang-agent-skills@golang-safety` for nil and boundary mishandling in string processing
