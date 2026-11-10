# Distributed Tracing with OpenTelemetry

Per-service logs can't answer "why was *this* request slow across the fleet." A trace stitches every hop and its timing for one request into a single path. `golang-context` covers propagating context across boundaries; `golang-samber-oops` covers structured errors in spans. For up-to-date API signatures, consult the OTel Go SDK docs.

## SDK setup

Initialise the `TracerProvider` early, before any spans exist:

```go
func initTracer(ctx context.Context) (func(), error) {
    exporter, err := otlptracegrpc.New(ctx)
    if err != nil {
        return nil, fmt.Errorf("creating OTLP exporter: %w", err)
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String("my-service"),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("creating resource: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)

    return func() { _ = tp.Shutdown(context.Background()) }, nil
}
```

## Creating spans

Every meaningful operation gets a span — the building block that shows where time went:

```go
var tracer = otel.Tracer("myapp/orders")

func (s *OrderService) Create(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    ctx, span := tracer.Start(ctx, "OrderService.Create")
    defer span.End()

    span.SetAttributes(
        attribute.String("order.payment_method", req.PaymentMethod),
        attribute.Float64("order.amount", req.Amount),
    )

    return s.repo.Insert(ctx, req.ToOrder())
}
```

Span service methods, database queries, external API calls, and message queue publish/consume — anything that takes measurable time or can fail.

## otelhttp

Incoming and outgoing HTTP are instrumented automatically:

```go
// incoming — wrap the handler
mux.Handle("/orders", otelhttp.NewHandler(orderHandler, "CreateOrder"))

// outgoing — clients must use otelhttp Transport to propagate context
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

## Errors

On failure call both `RecordError` and `SetStatus(codes.Error, ...)` so the span is flagged and searchable. Success needs no status change.

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, "operation failed")
    return err
}
```

## Structured errors with `samber/oops`

Plain Go errors surface in a trace as a bare message — no stack, no context. `oops` wraps errors with a stack trace, structured attributes, and a machine-searchable code that lands in both spans and slog lines:

```go
return nil, oops.
    In("order-service").
    Code("order_insert_failed").
    With("order_id", req.OrderID).
    With("user_id", req.UserID).
    Wrapf(err, "inserting order")
```

`errors.Is`/`errors.As` and `span.RecordError` keep working unchanged — see `golang-error-handling` and `golang-samber-oops`.

## Sampling

Across a fleet, 100% tracing is unaffordable. Sample:

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)),
    sdktrace.WithBatcher(exporter),
    sdktrace.WithResource(res),
)
```

`ParentBased` wraps any sampler so child services honour the caller's decision and traces stay complete across boundaries. Record error spans regardless of sampling rate.

## Cost of tracing

Tracing is typically the priciest signal: every span is serialized, shipped, stored, and indexed. A 10k req/s service at 5 spans per request produces 50k spans/s before any sampling.

- Pricing scales with volume (Jaeger, Tempo, Datadog all charge on it); large span attributes multiply every byte.
- Start at 10% (`TraceIDRatioBased(0.1)`) and tune to traffic and budget.
- Tail-based sampling (keep only errors, or slow requests) preserves the interesting traces at a fraction of the cost.
- Keep payloads out of span attributes — log them and correlate by `trace_id`.
