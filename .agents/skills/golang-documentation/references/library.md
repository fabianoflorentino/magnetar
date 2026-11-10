# Library Documentation

## Public vs private

| Artifact | Public library | Private library |
| --- | --- | --- |
| Doc comments on exported symbols | Required | Required |
| Package comments | Required | Required |
| README.md | Required | Required |
| Examples in doc comments | Generous | Generous |
| `ExampleXxx` test functions | Recommended | Recommended |
| Go Playground demos | Recommended | N/A |
| pkg.go.dev / godoc | Primary docs surface | `go doc` or internal pkgsite |
| Documentation website | Large projects | Multi-team libraries |
| Registration (Context7, DeepWiki) | Recommended | N/A |
| llms.txt | Recommended | Optional |
| CHANGELOG.md | Recommended | Recommended |
| CONTRIBUTING.md | Recommended | Recommended |

Keep comments and examples first-rate even for internal code: teams rotate, people forget, and tools need context to help effectively. The difference is only the public-facing artifacts — playground demos, pkg.go.dev, registries.

## Example test functions

`Example` functions in `_test.go` are executable: godoc renders them, and `go test` verifies them against `// Output:`. Without an output comment the example compiles but asserts nothing.

```go
// In map_example_test.go
package mypackage_test

func ExampleMap() {
    result := mypackage.Map([]int{1, 2, 3}, func(x int) int { return x * 2 })
    fmt.Println(result)
    // Output: [2 4 6]
}
```

Naming:

| Function name | Documents |
| --- | --- |
| `Example()` | The whole package |
| `ExampleFuncName()` | A package function |
| `ExampleTypeName()` | A type |
| `ExampleTypeName_MethodName()` | A type's method |
| `ExampleFuncName_suffix()` | Another variant (lowercase suffix) |

See `golang-testing` for the full example-testing workflow.

## Code examples in doc comments

Show the common use, an edge case, and error handling without routing readers elsewhere:

```go
// NewClient creates an HTTP client.
//
// Basic:
//
//	client := NewClient()
//
// With options:
//
//	client := NewClient(
//	    WithTimeout(10 * time.Second),
//	    WithRetries(3),
//	)
func NewClient(opts ...Option) *Client {
```

## Go Playground demos

A `Play:` line in a doc comment gives public-library readers a runnable demo with zero install. Keep the program self-contained (imports, `main`, printed output) and start with the most common use. Create links at <https://go.dev/play/> or via an integration when one is available.

## godoc and pkg.go.dev

Doc comments render automatically once a release is tagged and the package is imported.

How godoc renders:

- First sentences appear in the package index.
- `// Package foo provides...` becomes the package description.
- Blocks indented by one tab render as code.
- `# Heading` (Go 1.19+) creates sections.
- `[Link text]` and `[Identifier]` create links.
- `Deprecated:` gets distinct styling.

Private modules never reach pkg.go.dev. Use `go doc` locally, or run a shared pkgsite instance on the internal network:

```bash
go doc -all github.com/{owner}/{repo}
go tool pkgsite -http=:6060
```

## Documentation website

For large libraries a dedicated site — Docusaurus (React, native versioning) or MkDocs Material (simpler, strong search) — can beat a wall of READMEs.

Organize along the Diátaxis split:

| Section | Answers | Example |
| --- | --- | --- |
| Getting Started | "Where do I start?" | "Install and run your first query in 5 minutes" |
| Tutorial | "Teach me, step by step" | "Build a REST API with authentication" |
| How-to guides | "How do I do X?" | "Configure connection pooling" |
| Reference | "What is this API?" | Generated from godoc |
| Deep dives | "How does it work?" | "How the scheduler algorithm works" |

### llms.txt

A root-level `llms.txt` helps AI agents and tooling find the project's entry points; keep it current alongside the README.

### Discoverability

Register the library so agents and documentation aggregators index it:

- Context7 — <https://context7.com>
- DeepWiki — <https://deepwiki.com>
- OpenDeep — <https://opendeep.wiki>
- zRead — <https://zread.ai>