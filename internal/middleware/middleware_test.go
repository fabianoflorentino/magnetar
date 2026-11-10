package middleware

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func dummyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		allowedOrigins string
		method         string
		wantOrigin     string
		wantStatus     int
	}{
		{
			name:           "wildcard origin",
			allowedOrigins: "*",
			method:         http.MethodGet,
			wantOrigin:     "*",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "specific origin",
			allowedOrigins: "https://example.com",
			method:         http.MethodPost,
			wantOrigin:     "https://example.com",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "empty defaults to wildcard",
			allowedOrigins: "",
			method:         http.MethodGet,
			wantOrigin:     "*",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "preflight options",
			allowedOrigins: "*",
			method:         http.MethodOptions,
			wantOrigin:     "*",
			wantStatus:     http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := CORS(dummyHandler(), tt.allowedOrigins)

			req := httptest.NewRequest(tt.method, "/", nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d; want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("got origin %q; want %q", got, tt.wantOrigin)
			}
			if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
				t.Error("missing CORS allow-methods header")
			}
			if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
				t.Error("missing CORS allow-headers header")
			}
		})
	}
}

//nolint:paralleltest // subtests compartilham o mesmo RateLimiter e são sequenciais por design (limite → bloqueio).
func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	defer rl.Stop()
	h := rl.Middleware(dummyHandler())

	t.Run("allows up to limit", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "1.2.3.4:1234"
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("request %d: got status %d; want %d", i, w.Code, http.StatusOK)
			}
		}
	})

	t.Run("blocks over limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("got status %d; want %d", w.Code, http.StatusTooManyRequests)
		}
		if w.Header().Get("Retry-After") == "" {
			t.Error("expected Retry-After header on rate limit response")
		}
	})

	t.Run("independent per ip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "5.6.7.8:5678"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("resets after window", func(t *testing.T) {
		rl := NewRateLimiter(2, 50*time.Millisecond)
		defer rl.Stop()
		h := rl.Middleware(dummyHandler())

		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
		}

		time.Sleep(60 * time.Millisecond)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("got status %d; want %d após janela expirar", w.Code, http.StatusOK)
		}
	})
}

func TestRateLimiterXFF(t *testing.T) {
	t.Run("ignores xff without trusted proxies", func(t *testing.T) {
		t.Parallel()
		rl := NewRateLimiter(1, time.Minute)
		defer rl.Stop()
		h := rl.Middleware(dummyHandler())

		// Primeira chamada esgota o limite do IP real.
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.20.30.40:9999"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		// Spoofing de XFF não deve contornar o limite.
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.20.30.40:9999"
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("got status %d; want %d (XFF spoofing não deve funcionar)", w.Code, http.StatusTooManyRequests)
		}
	})

	t.Run("trusts xff from trusted proxy", func(t *testing.T) {
		t.Parallel()
		rl := NewRateLimiter(1, time.Minute, WithTrustedProxies([]string{"10.0.0.0/8"}))
		defer rl.Stop()
		h := rl.Middleware(dummyHandler())

		// Proxy confiável repassa o cliente real em XFF.
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.5:443"
		req.Header.Set("X-Forwarded-For", "203.0.113.9")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		// O mesmo cliente real é bloqueado na segunda chamada.
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.5:443"
		req.Header.Set("X-Forwarded-For", "203.0.113.9")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("got status %d; want %d", w.Code, http.StatusTooManyRequests)
		}
	})
}

//nolint:paralleltest // conta goroutines globais; paralelismo tornaria a medição instável.
func TestRateLimiterStop(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	rl.Stop()
	rl.Stop()
	before := runtime.NumGoroutine()
	time.Sleep(50 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutines cresceram após Stop: %d -> %d", before, after)
	}
}

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	h := RequestLogger(nil)(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}
