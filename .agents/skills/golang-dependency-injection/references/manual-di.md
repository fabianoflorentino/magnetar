# Manual Constructor Injection

Manual DI is the simplest wiring there is: constructors take their dependencies, `main` composes them in dependency order. No library, no codegen, no reflection - just readable, explicitly ordered Go.

## Complete application example

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    // Configuration
    cfg := LoadConfig()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Infrastructure
    db, err := postgres.Connect(cfg.DatabaseURL)
    if err != nil {
        logger.Error("database connection failed", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    cache := redis.NewClient(cfg.RedisURL)
    defer cache.Close()

    mailer := smtp.NewMailer(cfg.SMTPAddr)

    // Repositories
    userRepo := postgres.NewUserRepository(db)
    orderRepo := postgres.NewOrderRepository(db)

    // Services
    userSvc := service.NewUserService(userRepo, cache, mailer, logger)
    orderSvc := service.NewOrderService(orderRepo, userSvc, logger)
    paymentSvc := service.NewPaymentService(orderRepo, cfg.StripeKey, logger)

    // Transport
    handler := http.NewHandler(userSvc, orderSvc, paymentSvc, logger)
    server := http.NewServer(cfg.Port, handler)

    go server.ListenAndServe()
    <-ctx.Done()
    server.Shutdown(context.Background())
}
```

Dependencies must initialize in order - configuration, then infrastructure, then repositories, then services, then transport. The failure mode the pattern prevents is a service discovering its own dependencies behind `init()` or magic env reads.

## When it works well

- Small-to-medium projects (roughly under 15 services).
- Clear layering with modest cross-dependencies.
- No lazy loading or elaborate lifecycle needs.
- Teams that value visible, greppable wiring over framework behavior.

## When it breaks down

- A new service forces edits to `main()` and careful ordering changes in call sites downstream.
- Lifecycle work - health checks, graceful shutdown - is hand-rolled with `defer` and signal handling at the top level.
- Everything is eagerly constructed at startup, even components that run rarely.
- Cross-cutting concerns (logging, tracing, metrics) must be threaded through every constructor signature.
- Past ~30 services the wiring block becomes fragile and hard to review.

Manual is the default for small projects. Reaching for a container is a response to measured pain - growing service count, lifecycle needs, or test friction - not a fashion choice.
