# Examples as Documentation

Examples are executable documentation: `go test` runs them and compares stdout to the `// Output:` comment, and `pkg.go.dev` renders them beside the documented symbol. An example that drifts from the API fails the build — unlike a code block in a README.

```go
func ExampleCalculatePrice() {
    price := CalculatePrice(100, 10.0)
    fmt.Printf("Price: %.2f\n", price)
    // Output: Price: 900.00
}

func ExampleCalculatePrice_singleItem() {
    price := CalculatePrice(1, 25.50)
    fmt.Printf("Price: %.2f\n", price)
    // Output: Price: 25.50
}
```

## Naming

The suffix decides where godoc attaches the example, so a typo silently detaches it from its symbol:

| Function name | Documents |
| --- | --- |
| `Example()` | the package itself |
| `ExampleCalculatePrice()` | the `CalculatePrice` function |
| `ExampleStore_Get()` | the `Get` method of `Store` |
| `ExampleStore_Get_cached()` | a named variant of the same method |

The segment after the second underscore MUST start lowercase — otherwise Go reads it as a type or method name and the example is orphaned.

## Output directives

- `// Output:` — stdout must match exactly (surrounding whitespace is trimmed).
- `// Unordered output:` — lines may arrive in any order. For map iteration and concurrent producers, which have no stable order.
- **No output comment** — the example compiles but does not run. Useful for code that needs a live dependency, but it stops verifying behavior; prefer a real output assertion.

## Placement

Examples live in `_test.go` files. Prefer the `package foo_test` external test package: an example that only compiles against the exported API proves the public surface is usable, which is the point of an example.
