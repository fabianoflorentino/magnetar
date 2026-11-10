// Command magnetar é o entrypoint HTTP do magnetar SaaS.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"magnetar/internal/handler"
	"magnetar/internal/middleware"
	"magnetar/internal/scraper"
	"magnetar/internal/security"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8080")
	rateLimit := getEnvInt("RATE_LIMIT", 10)
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "*")
	allowedHosts := splitList(getEnv("ALLOWED_HOSTS", "apachetorrent.com"))
	trustedProxies := splitList(getEnv("TRUSTED_PROXIES", ""))

	rl := middleware.NewRateLimiter(
		rateLimit,
		time.Minute,
		middleware.WithTrustedProxies(trustedProxies),
	)
	defer rl.Stop()

	validator := security.NewValidator(allowedHosts)
	extractHandler := handler.NewExtract(scraper.Extract, validator.Validate, logger)

	mux := http.NewServeMux()
	mux.Handle("POST /api/extract", extractHandler)
	mux.HandleFunc("GET /health", handler.Health)

	var h http.Handler = mux
	h = middleware.RequestLogger(logger)(h)
	h = rl.Middleware(h)
	h = middleware.CORS(h, allowedOrigins)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("magnetar iniciado",
			"porta", port,
			"rate_limit", rateLimit,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("servidor falhou", "erro", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("desligando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("servidor forçado a desligar", "erro", err)
		os.Exit(1)
	}

	logger.Info("servidor encerrado")
}

// getEnv retorna o valor da variável de ambiente ou o fallback quando vazio.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt retorna a variável como inteiro ou o fallback em valores inválidos.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// splitList separa valores de ambiente separados por vírgula.
func splitList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
