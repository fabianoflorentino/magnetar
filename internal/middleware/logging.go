package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"time"
)

// statusRecorder captura o status code para logging sem quebrar o fluxo.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader armazena o status antes de passá-lo adiante.
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// RequestLogger registra cada requisição com método, caminho, status e duração.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.Info("requisição",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_ip", remoteIP(r),
			)
		})
	}
}

// remoteIP extrai apenas o endereço IP do RemoteAddr para evitar cardinalidade
// alta nos logs (porta descartada).
func remoteIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
