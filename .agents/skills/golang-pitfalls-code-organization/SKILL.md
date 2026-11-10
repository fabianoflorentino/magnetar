---
name: golang-pitfalls-code-organization
description: "Golang code and project organization — unintended variable shadowing, unnecessary nested code, misusing init functions, overusing getters/setters, interface pollution, producer/consumer-side interfaces, returning interfaces, any, generics, type embedding, the functional options pattern, package layout, utility packages, package name collisions, code documentation, and linters. Distilled from mistakes #1-16 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang project/package structure, interfaces, generics, or documentation."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📂"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Code & Project Organization

Source material: mistakes #1-16 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when writing Go code related to code structure, packages, interfaces, and project layout. The number in each heading maps to the corresponding mistake in the book.

## 1. Unintended variable shadowing (#1)

- Variable shadowing (redeclaring a name in an inner block) can make code compile while assigning to a different variable than expected.
- Avoid shadowing; run `go vet` or a shadowing linter to detect it. Only reusing idiomatic names like `err` is occasionally acceptable.
- Prefer unique names in inner scopes.

## 2. Unnecessary nested code (#2)

- Reduce nesting; keep the happy path aligned left and return early.
- If an `if` block returns, drop the `else`.
- Flip conditions so non-happy paths return early:

```go
// instead of
if s != "" {
    // ...
} else {
    return errors.New("empty string")
}
// write
if s == "" {
    return errors.New("empty string")
}
// ...
```

## 3. Misusing init functions (#3)

- `init()` takes no args, returns nothing, and runs at package initialization.
- Problems: limited error handling, complicated testing (external deps must be set up), and state handled through global variables.
- In most cases handle initialization through explicit ad hoc functions. `init` is acceptable only for things like static configuration.

## 4. Overusing getters and setters (#4)

- Go has no automatic getters/setters and it is not idiomatic to add them blindly.
- Don't add getters/setters unless they bring value or you need forward compatibility.
- Be pragmatic; simplicity is a core Go value.

## 5. Interface pollution (#5)

- Abstractions should be **discovered, not created**. Don't design with interfaces; wait for a concrete need.
- Interface pollution = unnecessary abstractions that add indirection and complexity. If an interface doesn't clearly improve the code, remove it.
- Good reasons to use an interface:
  - **Common behavior** factored across types (e.g., `sort.Interface`: `Len`, `Less`, `Swap`).
  - **Decoupling** from an implementation to enable testing/mocks (e.g., a small `customerStorer` interface instead of `mysql.Store`).
  - **Restricting behavior** (e.g., expose only a `Get() int` getter to make config read-only).
- Keep interfaces small: "The bigger the interface, the weaker the abstraction."
- Consider the granularity trade-off. Small, composable interfaces (`io.Reader`, `io.Writer`, `io.ReadWriter`) are ideal.

## 6. Interface on the producer side (#6)

- Interfaces are satisfied implicitly in Go. In most cases an interface should live on the **consumer** (client) side, not the producer side.
- Only put an interface on the producer side when you *know* (not foresee) it helps consumers; keep it minimal to maximize reusability.

## 7. Returning interfaces (#7)

- A function should generally return concrete types, not interfaces — returning interfaces restricts flexibility and couples all clients to one abstraction.
- Accept interfaces whenever possible; return concrete types.
- Only return an interface when you *know* an abstraction helps clients; otherwise let clients discover their own abstractions.

## 8. `any` says nothing (#8)

- Use `any` only when you genuinely need to accept/return any possible type (e.g., `json.Marshal`).
- Otherwise `any` loses type safety and expresses nothing. A little duplication beats loss of expressiveness.

## 9. Being confused about when to use generics (#9)

- Generics (type parameters) factor out boilerplate, but don't use them prematurely — unnecessary abstraction adds complexity.
- Use generics when you have a concrete need, e.g.:
  - Data structures (linked list, tree, heap): `type Node[T any] struct`
  - Functions over slices/maps/channels of any type: `func getKeys[K comparable, V any](m map[K]V) []K`
  - Factoring out behaviors
- Constraints: interface types; use `~int` to allow types whose underlying type is `int`, `|` for unions. Check the `constraints` package before writing your own.
- Don't use generics when merely calling a method of the type argument (use the interface instead) or when it makes code more complex.
- Remember: methods cannot have type parameters.

## 10. Type embedding problems (#10)

- Embedding promotes fields/methods of the embedded type. It's rarely a necessity — mostly convenience.
- Don't embed just as syntactic sugar (e.g., `foo.Baz()` instead of `foo.Bar.Baz()`); use a named field instead.
- Don't promote data or behavior that should stay hidden.
- Be aware of unintended promotion causing visibility bugs.

## 11. Not using the functional options pattern (#11)

- Functional options is the idiomatic way to handle configuration options in an API-friendly manner.

```go
type options struct {
    port *int
}

type Option func(options *options) error

func WithPort(port int) Option {
    return func(options *options) error {
        if port < 0 {
            return errors.New("port should be positive")
        }
        options.port = &port
        return nil
    }
}

func NewServer(addr string, opts ...Option) (*http.Server, error) {
    var options options
    for _, opt := range opts {
        if err := opt(&options); err != nil {
            return nil, err
        }
    }
    // ...
}
```

- Option functions hold validation and configuration; the constructor applies each option.
- The builder pattern is a valid alternative, but functional options handle defaults and errors more idiomatically.

## 12. Project misorganization (#12)

- Choose an organization (by context vs. by layer) that fits the use case and stay consistent.
- Avoid premature packaging; let the project evolve.
- Avoid nano packages (one/two files) and huge packages that dilute meaning.
- Name packages after **what they provide**, short, concise, single lowercase word.
- Minimize exports by default; export only what's needed (exception: exported fields for `encoding/json` unmarshaling).
- Reference: the official Go layout guideline `go.dev/doc/modules/layout`.

## 13. Creating utility packages (#13)

- Avoid packages named `common`, `util`, `shared` — they add no value for the reader.
- Refactor into meaningful, specific package names describing what they provide.

## 14. Ignoring package name collisions (#14)

- A variable name colliding with a package name prevents reusing the package and causes ambiguity.
- Rename the variable or use an import alias to change the qualifier.

## 15. Missing code documentation (#15)

- Every exported element must be documented with a comment starting with its name.
- Comments should be complete sentences ending with punctuation; describe **what** a function does, not **how**.
- Document packages with `// Package <name>`; keep the first line concise (it appears in package docs).
- For variables/constants, document purpose; content details may stay private.

## 16. Not using linters (#16)

- Use linters and formatters to improve quality and consistency:
  - `go vet` (standard analyzer)
  - `errcheck` (error checker)
  - `gocyclo` (cyclomatic complexity)
  - `goconst` (repeated string constants)
  - `gofmt` / `goimports` (formatters)
  - `golangci-lint` (facade over many linters, runs in parallel)
- Automate execution in CI or a pre-commit hook.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-project-layout` for project structure and workspace conventions
- → See `fabianoflorentino/golang-agent-skills@golang-naming` for naming conventions
- → See `fabianoflorentino/golang-agent-skills@golang-design-patterns` for the functional options and builder patterns
- → See `fabianoflorentino/golang-agent-skills@golang-code-style` and `fabianoflorentino/golang-agent-skills@golang-lint` for formatting, linting, and linter configuration
