---
name: golang-grpc
description: "Provides gRPC usage guidelines, protobuf organization, and production-ready patterns for Golang microservices. Use when implementing, reviewing, or debugging gRPC servers/clients, writing proto files, setting up interceptors, handling gRPC errors with status codes, configuring TLS/mTLS, testing with bufconn, or working with streaming RPCs."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🌐"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - protoc
    install:
      - kind: brew
        formula: protobuf
        bins: [protoc]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(protoc:*) AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who treats RPC definitions as interface contracts. You pick the right status code, put a deadline on every client call, and never leave a server goroutine alive once the caller has moved on.

**Modes:**

- **Build** — creating a `.proto`, an RPC server, or a client from scratch.
- **Review** — auditing an existing gRPC surface for contract drift, security, and operability defects.
- **Debug** — tracing a failing call across status codes, metadata propagation, and connection linger.

**When to use:** any task centered on service-to-service RPC in Go. If the API must be consumed from browsers or mobile over plain HTTP JSON, prefer `golang-rest` or `golang-graphql`. Load `golang-context`, `golang-error-handling`, and `golang-observability` alongside; gRPC errors, deadlines, and traces flow through those skills' conventions.

## Transport choice

| Scenario | Prefer |
| --- | --- |
| Internal services that stream or shuffle large payloads | gRPC |
| Public / browser-facing JSON contract | gRPC-gateway or REST |
| Schema-typed, polyglot service mesh | gRPC with protobuf |
| Telemetry, health checks, and internal tooling | gRPC |

gRPC is a transport, not an architecture: keep domain logic behind the generated stubs so swapping transports stays mechanical.

## Key packages and tools

| Concern | Choice |
| --- | --- |
| Service definition | `.proto` files, built with `buf` or `protoc` |
| Code generation | `protoc-gen-go`, `protoc-gen-go-grpc` |
| Status codes and errors | `google.golang.org/grpc/status` + `codes` |
| Rich error detail | `google.golang.org/genproto/googleapis/rpc/errdetails` |
| Cross-cutting middleware | `grpc.ChainUnaryInterceptor` / `ChainStreamInterceptor` |
| Ecosystem middleware | `github.com/grpc-ecosystem/go-grpc-middleware` |
| In-memory testing | `google.golang.org/grpc/test/bufconn` |
| Transport credentials | `google.golang.org/grpc/credentials` |
| Readiness probes | `google.golang.org/grpc/health` |

## Proto organization

- Version the package per API, not per repo: `proto/checkout/v1/checkout.proto`.
- Name packages after the domain (`package checkout.v1`), not after the Go module.
- Wrap each operation in a dedicated `*Request`/`*Response` message. A wall of scalar fields cannot evolve; a message can.
- Set `option go_package` to the full module path so generated imports resolve inside `go.work`.

```proto
syntax = "proto3";
package checkout.v1;
option go_package = "github.com/you/checkout/gen/checkoutv1;checkoutv1";

service CheckoutService {
  rpc Checkout(CheckoutRequest) returns (CheckoutResponse);
}

message CheckoutRequest {
  repeated CartItem items = 1;
}
```

Generate with `buf generate` when the module also needs linting and breaking-change gates (`buf breaking --against .git`); `protoc` alone is fine for plain codegen. Full reference: [protoc & buf reference](references/protoc-reference.md).

## Server side

Register the health service (`grpc_health_v1`) so orchestrators can resolve readiness. Attach cross-cutting concerns via chained interceptors instead of inline in handlers, and drain with `GracefulStop` bounded by a timer.

```go
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(requestIDInterceptor, authInterceptor),
)
pb.RegisterCheckoutServiceServer(srv, svc)
healthpb.RegisterHealthServer(srv, health.NewServer())

done := make(chan struct{})
go func() { srv.GracefulStop(); close(done) }()
select {
case <-done: // drained
case <-time.After(15 * time.Second):
    srv.Stop() // force-drop stragglers
}
```

Interceptors should stay generic — extract metadata, add trace IDs, recover panics — and never encode business rules.

```go
func authInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
    if token, ok := metadata.FromIncomingContext(ctx); !ok || len(token["authorization"]) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing credentials")
    }
    return handler(ctx, req)
}
```

Enable reflection in development only; serving it in production publishes the full surface to anyone who probes the endpoint.

## Streaming

| Pattern | Typical use |
| --- | --- |
| Server-streaming | tailing logs, paging result sets |
| Client-streaming | uploads, aggregation batches |
| Bidirectional | chat, coordinating workers |

Streaming beats oversized single messages when payloads can be chunked: it sidesteps the per-message size cap and keeps memory bounded. Always size packets deliberately when the peer buffers the whole stream.

```go
func (s *server) Feed(stream pb.FeedService_FeedServer) error {
    for {
        req, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            return nil
        }
        if err != nil {
            return status.Errorf(codes.Aborted, "receive: %v", err)
        }
        if err := stream.Send(&pb.FeedItem{Chunk: req.Data}); err != nil {
            return err
        }
    }
}
```

## Client side

- Reuse one connection per target. HTTP/2 multiplexes every call over it; dialing per request burns handshake cycles.
- Give every call a deadline with `context.WithTimeout`, or a hung upstream holds goroutines open forever.
- For retry and load balancing, ship a default service config instead of hand-rolling loops.
- Attach auth and trace IDs through `metadata.NewOutgoingContext`.

```go
conn, err := grpc.NewClient("dns:///checkout-service:50051",
    grpc.WithTransportCredentials(creds),
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingPolicy": "round_robin",
        "methodConfig": [{
            "name": [{"service": "checkout.v1.CheckoutService"}],
            "timeout": "5s",
            "retryPolicy": {
                "maxAttempts": 3,
                "initialBackoff": "0.1s",
                "maxBackoff": "1s",
                "backoffMultiplier": 2.0,
                "retryableStatusCodes": ["UNAVAILABLE", "ABORTED"]
            }
        }]
    }`),
)
defer conn.Close()
client := pb.NewCheckoutServiceClient(conn)

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
resp, err := client.Checkout(ctx, &pb.CheckoutRequest{Items: items})
```

`dns:///` resolves per name lookup and pairs with `round_robin` across headless Kubernetes pods; a plain `passthrough` target pins you to one address.

## Error handling

Map failures to canonical codes so clients can act on the code, not on string matching. A raw `error` surfaces as `codes.Unknown`, which no caller can interpret.

| Code | Trigger |
| --- | --- |
| `InvalidArgument` | malformed input |
| `NotFound` | missing entity |
| `AlreadyExists` | duplicate create |
| `PermissionDenied` | insufficient rights |
| `Unauthenticated` | missing / bad token |
| `FailedPrecondition` | wrong system state |
| `ResourceExhausted` | quota or rate limit |
| `Unavailable` | transient backend failure (retryable) |
| `DeadlineExceeded` | timeout |
| `Aborted` | conflicting write / concurrency clash |

```go
return nil, status.Errorf(codes.NotFound, "cart %q does not exist", req.CartId)
```

For validation detail, attach `errdetails.BadRequest` with `status.WithDetails` instead of stuffing arrays into the message string.

## Testing

Exercise the full stack — serialization, interceptors, metadata — with `bufconn` in-memory listeners; no sockets, no leaked ports, and the same code paths a real connection uses.

```go
lis := bufconn.Listen(1 << 20)
pb.RegisterCheckoutServiceServer(srv, svc)
go srv.Serve(lis)

conn, err := grpc.NewClient("passthrough:///bufnet",
    grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
        return lis.Dial()
    }),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
defer conn.Close()
```

Assert both the payload and the status code on error paths — a mutation test that returns `Unknown` instead of `NotFound` is exactly the regression you want to catch. See [testing patterns](references/testing.md).

## Security

- Enforce TLS in any environment that carries credentials; metadata is not encrypted by default.
- For service identity, use mTLS or delegate to a mesh (Istio, Linkerd) rather than shared secrets in headers.
- For end users, implement `credentials.PerRPCCredentials` and validate tokens in an auth interceptor.
- Turn reflection off in production.

## Performance

| Knob | Effect | Typical |
| --- | --- | --- |
| `keepalive.ServerParameters.Time` | idle ping interval | 30s |
| `keepalive.ServerParameters.Timeout` | ping ack timeout | 10s |
| `grpc.MaxRecvMsgSize` | raise the 4 MB default for big payloads | 16 MB |
| `WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(...))` | per-call cap on big responses | match service caps |

Profile before bolting on connection pools — multiplexing already serves most workloads; pools help only high-throughput streaming.

## Common mistakes

| Mistake | Consequence | Fix |
| --- | --- | --- |
| Returning a plain `error` | becomes `codes.Unknown`; clients cannot retry correctly | `status.Errorf` with a specific code |
| No deadline on client calls | goroutines pile up on a slow upstream | wrap every call in `context.WithTimeout` |
| Dialing per request | redundant TCP/TLS handshakes | one connection, reused |
| Reflection in production | anyone can enumerate the API | enable only in dev/staging |
| `Internal` for everything | hides retryable failures from clients | use `Unavailable` / `Aborted` where retry makes sense |
| Scalar RPC arguments | fields cannot be added later | envelope `*Request` messages |
| Missing health service | orchestrator kills healthy pods during rollouts | register `grpc_health_v1` |
| Ignoring `ctx.Err()` | work continues after the caller gave up | drain `RequestStream.Context()` in loops |

## Cross-references

- `golang-context` — deadlines, cancellation, and context propagation.
- `golang-error-handling` — mapping gRPC codes to idiomatic Go errors.
- `golang-observability` — interceptors for logs, metrics, and traces.
- `golang-testing` — table-driven and integration test habits.
- `golang-swagger` — when the gRPC API is exposed as REST via grpc-gateway.
