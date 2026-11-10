# Clean Architecture in Go

Clean architecture arranges the code in concentric layers — entities at the core, then use cases, then adapters, then technical details — so business logic depends on nothing external. Choose it for a service around 2K+ lines where testability and a stable core matter; skip it for small CLIs and scripts.

## The dependency rule

Dependencies point inward and only inward:

```
Frameworks & Drivers → Interface Adapters → Use Cases → Entities
```

Each layer defines the interfaces it needs, and the layer outside implements them. The payoff: entities and use cases import nothing from HTTP, the database, or a router.

## Suggested layout

```
order-service/
├── cmd/server/main.go            # wiring only — assembles the dependency graph
└── internal/
    ├── entity/                   # order.go, item.go, status.go — pure rules
    ├── order/                    # place.go, cancel.go, port.go — use cases
    ├── adapter/
    │   ├── handler/order_handler.go     # HTTP handler
    │   ├── repository/order_postgres.go # implements the repository port
    │   └── gateway/payment_client.go    # external payment API
    └── infrastructure/           # router.go, database.go, config.go
```

`cmd/server/main.go` is the only place that constructs concrete dependencies.

## Pieces, with code

### Entity

```go
// internal/entity/order.go
package entity

type Order struct {
    ID     string
    Items  []Item
    Status OrderStatus
}

func (o *Order) Cancel() error {
    if o.Status == StatusShipped {
        return ErrCannotCancelShipped
    }
    o.Status = StatusCancelled
    return nil
}
```

### Port — what the use case needs

```go
// internal/order/port.go
package order

type OrderRepository interface {
    Save(ctx context.Context, order *entity.Order) error
    FindByID(ctx context.Context, id string) (*entity.Order, error)
}

type PaymentGateway interface {
    Charge(ctx context.Context, orderID string, amount int64) error
}
```

### Use case

```go
// internal/order/place.go
package order

type PlaceOrderUseCase struct {
    orders   OrderRepository
    payments PaymentGateway
}

func NewPlaceOrderUseCase(orders OrderRepository, payments PaymentGateway) *PlaceOrderUseCase {
    return &PlaceOrderUseCase{orders: orders, payments: payments}
}

func (uc *PlaceOrderUseCase) Execute(ctx context.Context, orderID string) error {
    order, err := uc.orders.FindByID(ctx, orderID)
    if err != nil {
        return fmt.Errorf("finding order: %w", err)
    }
    if err := uc.payments.Charge(ctx, order.ID, order.Total()); err != nil {
        return fmt.Errorf("charging payment: %w", err)
    }
    order.Status = entity.StatusPlaced
    return uc.orders.Save(ctx, order)
}
```

### Adapter and handler

```go
// internal/adapter/repository/order_postgres.go
type OrderPostgres struct{ db *sql.DB }

func (r *OrderPostgres) FindByID(ctx context.Context, id string) (*entity.Order, error) { /* query + scan */ }
func (r *OrderPostgres) Save(ctx context.Context, order *entity.Order) error             { /* upsert */ }
```

```go
// internal/adapter/handler/order_handler.go
func (h *OrderHandler) HandlePlaceOrder(w http.ResponseWriter, r *http.Request) {
    orderID := r.PathValue("id")
    if err := h.placeOrder.Execute(r.Context(), orderID); err != nil {
        http.Error(w, err.Error(), mapToHTTPStatus(err))
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

## Where interfaces live

Interfaces belong where they are consumed. `OrderRepository` is declared in the order use-case package and implemented by the adapter — the use case never knows the storage package exists.

## Wiring

Construct everything in `cmd/server/main.go` and pass it in through constructors: repository into use case, use case into handler, handler into router. If a DI library suits the project, see `golang-dependency-injection`.