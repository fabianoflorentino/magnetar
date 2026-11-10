# Benchmarks in a Test Suite

Measurement methodology — `benchstat`, profiling from benchmarks, noise control, CI regression detection — lives in the `fabianoflorentino/golang-agent-skills@golang-benchmark` skill. This page covers only writing a benchmark that sits next to the package's tests.

## Shape

```go
func BenchmarkStringConcatenation(b *testing.B) {
    b.Run("plus-operator", func(b *testing.B) {
        for b.Loop() {
            result := "a" + "b" + "c"
            _ = result
        }
    })

    b.Run("strings.Builder", func(b *testing.B) {
        for b.Loop() {
            var builder strings.Builder
            builder.WriteString("a")
            builder.WriteString("b")
            builder.WriteString("c")
            _ = builder.String()
        }
    })
}
```

Sub-benchmarks give each variant its own row in the output — the row `benchstat` compares. A single benchmark mixing both variants emits one number and hides the difference.

## Varying input size

```go
func BenchmarkFibonacci(b *testing.B) {
    sizes := []int{10, 20, 30}
    for _, size := range sizes {
        b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
            b.ReportAllocs()
            for b.Loop() {
                Fibonacci(size)
            }
        })
    }
}
```

Size-parameterized sub-benchmarks expose growth: a jump that outpaces the size increase points at a superlinear algorithm, which no single-size benchmark reveals.

## `b.Loop()` vs `b.N`

Write new benchmarks with `b.Loop()` (Go 1.24+): it keeps setup outside the timed region and stops the compiler from optimizing the loop body away — the two failure modes that make `b.N` benchmarks report impossibly fast numbers. Fall back to the legacy `b.N` loop only when the module targets Go <1.24 or you intentionally preserve existing benchmark code.

See `fabianoflorentino/golang-agent-skills@golang-benchmark` for measurement methodology and regression detection.
