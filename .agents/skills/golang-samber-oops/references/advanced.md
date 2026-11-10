# samber/oops — Advanced Patterns

Practical extensions beyond the fluent builder: invariant assertions, process-wide configuration, and wiring structured errors into logging pipelines.

## Assertions

`oops.Assert` / `oops.Assertf` panic on a failed invariant. They are for states that are logically impossible — a genuine bug — and belong inside a `Recover` guard so the panic becomes a structured error instead of a crash:

```go
func ProcessPayment(amount int) error {
    return oops.
        In("payment-service").
        Recover(func() {
            oops.Assertf(amount > 0, "amount must be positive, got %d", amount)
            // payment logic
        })
}
```

Keep them rare. If the condition can be triggered by real input, that input needs a normal error path, not an assertion.

## Configuration

Package-level knobs tune what an error carries and how it renders:

```go
oops.StackTraceMaxDepth = 20         // cap captured stack frames
oops.SourceFragmentsHidden = false   // embed source snippets in the trace
loc, _ := time.LoadLocation("America/New_York")
oops.Local = loc                     // timezone for error timestamps
```

Source fragments aid local debugging but leak surrounding code into logs; leave them hidden in production.

## Logger integration

`samber/oops` does not bind to a specific logger. The error type exposes its metadata directly:

```go
func logErr(err error) {
    oopsErr, ok := err.(oops.OopsError)
    if !ok {
        slog.Error("operation failed", "error", err)
        return
    }
    slog.Error("operation failed",
        "code", oopsErr.Code(),
        "domain", oopsErr.Domain(),
        "user_id", oopsErr.User(),
        "error", oopsErr,
    )
}
```

For `slog` the one-call form is enough — `slog.Error(err.Error(), slog.Any("error", err))` embeds code, domain, attributes, and stack into a single record. Formatters ship for `zerolog` (`log.Error().Err(err)`) and `logrus` (`log.WithError(err)`) when those loggers are in use.

Use `errors.As(err, &oopsErr)` rather than a bare type assertion wherever the error may have passed through additional wrapping.