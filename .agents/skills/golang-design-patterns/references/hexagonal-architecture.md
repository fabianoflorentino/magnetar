# Hexagonal Architecture (Ports & Adapters) in Go

Hexagonal architecture isolates the application core from every external system by drawing abstract boundaries — ports — that concrete adapters implement. Choose it when the same business logic is driven through several entry points (HTTP, gRPC, CLI, message consumers) or touches many external systems. Skip it for small CRUD apps and libraries.

## Core vocabulary

- **Domain/core** — business logic and types, no external imports.
- **Primary (driving) port** — how the outside world calls into the application, e.g. an `OrderService` interface.
- **Secondary (driven) port** — how the application calls out, e.g. `OrderRepository` or `PaymentGateway`.
- **Primary adapter** — inbound entry: HTTP handlers, gRPC servers, CLI commands.
- **Secondary adapter** — outbound implementation: Postgres repository, payment client, cache.

The domain is driven through primary ports and reaches out through secondary ports; neither side knows a concrete adapter.

## Layout

```
order-service/
├── cmd/
│   ├── server/main.go          # HTTP wiring
│   └── worker/main.go          # consumer wiring
└── internal/
    ├── domain/                 # order.go, item.go, status.go
    ├── port/
    │   ├── incoming.go         # primary ports (OrderService)
    │   └── outgoing.go         # secondary ports (OrderRepository, PaymentGateway)
    ├── service/order_service.go    # implements primary ports
    └── adapter/
        ├── primary/http/order_handler.go   # calls OrderService
        ├── primary/grpc/order_server.go    # calls OrderService
        └── secondary/postgres/order_repo.go
```

## The pieces, concretely

### Domain

```go
// internal/domain/order.go
package domain

type Order struct {
    ID     string
    Items  []Item
    Status OrderStatus
}

func (o *Order) Ship() error {
    if o.Status != StatusPaid {
        return ErrOrderNotPaid
    }
    o.Status = StatusShipped
    return nil
}
```

### Ports

```go
// internal/port/incoming.go
type OrderService interface {
    PlaceOrder(ctx context.Context, items []domain.Item) (string, error)
    ShipOrder(ctx context.Context, orderID string) error
    GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
}
```

```go
// internal/port/outgoing.go
type OrderRepository interface {
    Save(ctx context.Context, order *domain.Order) error
    FindByID(ctx context.Context, id string) (*domain.Order, error)
}

type PaymentGateway interface {
    Charge(ctx context.Context, orderID string, amount int64) error
}
```

### Service — implements the primary port, consumes the secondary ones

```go
// internal/service/order_service.go
package service

type orderService struct {
    orders   port.OrderRepository
    payments port.PaymentGateway
}

func NewOrderService(orders port.OrderRepository, payments port.PaymentGateway) port.OrderService {
    return &orderService{orders: orders, payments: payments}
}

func (s *orderService) PlaceOrder(ctx context.Context, items []domain.Item) (string, error) {
    order := domain.NewOrder(items)
    if err := s.payments.Charge(ctx, order.ID, order.Total()); err != nil {
        return "", fmt.Errorf("charging payment: %w", err)
    }
    if err := s.orders.Save(ctx, order); err != nil {
        return "", fmt.Errorf("saving order: %w", err)
    }
    return order.ID, nil
}
```

### Primary adapter

```go
func (h *OrderHandler) HandlePlaceOrder(w http.ResponseWriter, r *http.Request) {
    var req PlaceOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    id, err := h.svc.PlaceOrder(r.Context(), req.Items)
    if err != nil {
        http.Error(w, err.Error(), mapToHTTPStatus(err))
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"id": id})
}
```

### Secondary adapter

```go
// internal/adapter/secondary/postgres/order_repo.go
type OrderRepo struct{ db *sql.DB }

func (r *OrderRepo) Save(ctx context.Context, order *domain.Order) error { /* upsert */ }
func (r *OrderRepo) FindByID(ctx context.Context, id string) (*domain.Order, error) { /* query */ }
```

## Multiple entry points

This is where the pattern pays off: one `OrderService` driven by HTTP for external clients, gRPC for internal services, and a message consumer for async events. Each entry point ships in its own `cmd/` binary and wires only what it needs — the core never changes when a new adapter arrives.

## Wiring

Construct the adapters and the service in `cmd/server/main.go`. For DI library options see `golang-dependency-injection`.