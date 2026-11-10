# Choosing an Architecture for Your Go Service

Architecture is a tool, not a badge. It earns its place only when an unorganized package tree costs more than the ceremony. When starting fresh, ask which style the team actually wants before scaffolding anything.

## Match ceremony to size

| Codebase | Suggested shape |
| --- | --- |
| Script or small CLI (< 500 lines) | Flat — `main.go` plus a few files, no layers |
| Medium service (500–5K lines) | Simple tiers: `handler/`, `service/`, `repository/` |
| Large service or monolith (5K+ lines) | Clean, hexagonal, or DDD — decided with the team |

A 100-line CLI does not need a domain layer, ports and adapters, or a DI container. Start flat and refactor at the point the layering genuinely pays for itself.

## Keep the domain free of dependencies

Business types and rules never import HTTP, SQL, or transport code. Infrastructure packages depend on the domain; the domain depends on nothing outside other domain packages.

```go
// domain/order.go — no infrastructure imports
package domain

type Order struct {
    ID     string
    Items  []Item
    Status OrderStatus
}

func (o *Order) AddItem(item Item) error {
    if o.Status != StatusDraft {
        return ErrOrderNotEditable
    }
    o.Items = append(o.Items, item)
    return nil
}
```

This direction of dependence holds in every style covered here — clean, hexagonal, and DDD all enforce the same rule.

## Validate at the boundaries, then trust

Check input once, where it enters the system: HTTP handlers, CLI argument parsing, and message consumers. Once data is inside the domain, treat it as valid. Re-validating in every layer duplicates code and lets the rules drift apart.

```go
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var req CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    if req.UserID == "" {
        http.Error(w, "user_id is required", http.StatusBadRequest)
        return
    }
    order, err := h.service.CreateOrder(r.Context(), req.UserID, req.Items)
    // ...
}
```

## Make illegal states unrepresentable

Prefer a type system that cannot express an invalid state over a comment that asks callers to behave.

```go
type OrderStatus int

const (
    OrderStatusUnknown   OrderStatus = iota // zero = unset
    OrderStatusDraft
    OrderStatusConfirmed
)

type Order struct {
    Status OrderStatus // only the constants above are assignable
}
```

For values with real constraints, expose a constructor returning `(T, error)` and keep the field unexported:

```go
func NewEmail(raw string) (Email, error) {
    if !validEmail(raw) {
        return Email{}, fmt.Errorf("invalid email: %q", raw)
    }
    return Email{address: raw}, nil
}
```

## Prefer the explicit over the implicit

Hidden behavior — reflection-driven defaults, package-level globals a handler reaches for — reads as magic. Make defaults visible and dependency injection explicit.

```go
// default visible in code
func NewConfig() Config { return Config{Port: 8080} }

// dependency via struct field, no global
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    user := h.db.FindUser(r.Context(), id)
}
```

## Where to go deeper

For the 5K+ line projects that warrant a formal style:

- [`ddd.md`](./ddd.md) — aggregates, value objects, bounded contexts.
- [`clean-architecture.md`](./clean-architecture.md) — use cases, the dependency rule, adapters.
- [`hexagonal-architecture.md`](./hexagonal-architecture.md) — ports and adapters around a pure core.

See `golang-project-layout` for 12-factor conventions and directory layout.