# Domain-Driven Design in Go

Use DDD when the business model is genuinely rich — entities with rules, invariants that must hold, and several subdomains that share a vocabulary. It pays for itself at 5K+ lines with multiple bounded contexts; a CRUD service or a CLI gets the ceremony without the benefit.

## Building blocks

| Concept | Go shape | Role |
| --- | --- | --- |
| Entity | Struct with an identity field | Mutable object tracked by ID across its lifecycle |
| Value object | Immutable struct, compared by value | Quantity, measure, descriptor — no identity |
| Aggregate | Root entity plus children | Consistency boundary; all mutations enter through the root |
| Repository | Interface in the domain, implementation outside | Persistence contract for aggregates |
| Domain service | Function or struct in the domain package | Logic spanning several aggregates |
| Domain event | Struct recording a fact | Cross-context communication |

## Layout by bounded context

Group vertically per context (domain → application → adapters), not by technical role — a "models" plus "services" split across the whole tree hides ownership.

```
order-service/
├── cmd/server/main.go
└── internal/
    ├── order/                          # bounded context: Order
    │   ├── domain/                     # order.go, item.go, status.go, repository.go, events.go
    │   ├── application/                # PlaceOrderHandler, GetOrderHandler
    │   └── adapters/
    │       ├── persistence/postgres.go # implements domain.Repository
    │       └── http/handler.go         # HTTP transport
    ├── billing/                        # another context, same vertical shape
    ├── shared/money.go                 # value objects reused across contexts
    └── events/publisher.go             # shared event bus (infrastructure)
```

Keep shared infrastructure small and clearly labeled; cross-context usage is the exception, not the rule.

## Building blocks in code

### Value object — money in cents

```go
// internal/shared/money.go
package shared

type Money struct {
    amount   int64  // cents — never float for money
    currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
    if currency == "" {
        return Money{}, errors.New("currency is required")
    }
    return Money{amount: amount, currency: currency}, nil
}

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, fmt.Errorf("cannot add %s to %s", other.currency, m.currency)
    }
    return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}
```

### Aggregate root

```go
type Order struct {
    id     string
    items  []Item
    status Status
    total  shared.Money
}

func NewOrder(id string) *Order {
    return &Order{id: id, status: StatusDraft}
}

func (o *Order) AddItem(item Item) error {
    if o.status != StatusDraft {
        return ErrOrderNotEditable
    }
    o.items = append(o.items, item)
    return o.recalculateTotal()
}

func (o *Order) Place() (OrderPlaced, error) {
    if len(o.items) == 0 {
        return OrderPlaced{}, ErrEmptyOrder
    }
    o.status = StatusPlaced
    return OrderPlaced{OrderID: o.id, Total: o.total}, nil
}
```

The aggregate owns its invariants: no code path can mutate items or status except through the root, and every mutation is a method on it.

### Repository interface in the domain

```go
type Repository interface {
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
}
```

The Postgres implementation lives in the context's `adapters/persistence` package and depends on the domain — never the reverse.

### Application service

```go
type PlaceOrderHandler struct {
    orders domain.Repository
    events EventPublisher
}

func (h *PlaceOrderHandler) Handle(ctx context.Context, cmd PlaceOrderCommand) error {
    order, err := h.orders.FindByID(ctx, cmd.OrderID)
    if err != nil {
        return fmt.Errorf("finding order: %w", err)
    }
    evt, err := order.Place()
    if err != nil {
        return fmt.Errorf("placing order: %w", err)
    }
    if err := h.orders.Save(ctx, order); err != nil {
        return fmt.Errorf("saving order: %w", err)
    }
    return h.events.Publish(ctx, evt)
}
```

The application layer runs a use case; it holds no business rules of its own.

## Bounded contexts and the anti-corruption layer

Contexts communicate through domain events or explicit anti-corruption layers, never by importing one another's internals. If `billing/` must react to `order.OrderPlaced`, it translates the event into its own types:

```go
// internal/billing/adapters/events/order_events.go
func (s *OrderPlacedSubscriber) OnOrderPlaced(evt events.OrderPlaced) error {
    return s.invoices.Create(evt.OrderID, evt.Total) // billing-typed invoice
}
```

This keeps order's vocabulary from leaking into billing. As contexts grow large enough, promote each to its own Go module inside a `go.work` workspace — see `golang-project-layout`.

## Wiring

`cmd/server/main.go` wires repositories, handlers, and the event bus by hand. For DI library alternatives see `golang-dependency-injection`.