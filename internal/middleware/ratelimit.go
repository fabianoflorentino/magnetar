// Package middleware contém middlewares HTTP (rate limit, CORS e logging).
package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// visitor rastreia o consumo dentro da janela de tempo para um IP.
type visitor struct {
	count    int
	lastSeen time.Time
}

// RateLimiter limita requisições por IP dentro de uma janela deslizante.
type RateLimiter struct {
	visitors  map[string]*visitor
	mu        sync.Mutex
	limit     int
	window    time.Duration
	trusted   []netip.Prefix
	done      chan struct{}
	stopOnce  sync.Once
	cleanupWg sync.WaitGroup
}

// Option é uma opção funcional para configuração do RateLimiter.
type Option func(*RateLimiter)

// WithTrustedProxies configura CIDRs de proxies reversos cujo cabeçalho
// X-Forwarded-For deve ser considerado. Sem isso, o cabeçalho é ignorado,
// evitando bypass de limite por falsificação de cliente.
func WithTrustedProxies(cidrs []string) Option {
	return func(rl *RateLimiter) {
		for _, c := range cidrs {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if p, err := netip.ParsePrefix(c); err == nil {
				rl.trusted = append(rl.trusted, p)
				continue
			}
			if a, err := netip.ParseAddr(c); err == nil {
				rl.trusted = append(rl.trusted, netip.PrefixFrom(a, a.BitLen()))
			}
		}
	}
}

// NewRateLimiter constrói um RateLimiter com limite de requisições por janela.
func NewRateLimiter(limit int, window time.Duration, opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
		done:     make(chan struct{}),
	}
	for _, opt := range opts {
		opt(rl)
	}
	rl.cleanupWg.Add(1)
	go rl.cleanup()
	return rl
}

// Stop encerra a goroutine de limpeza e aguarda sua saída.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.done)
	})
	rl.cleanupWg.Wait()
}

// cleanup expira visitantes inativos da memória periodicamente.
func (rl *RateLimiter) cleanup() {
	defer rl.cleanupWg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Allow reporta se ip ainda pode realizar uma requisição nesta janela.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
		return true
	}

	if time.Since(v.lastSeen) > rl.window {
		v.count = 1
		v.lastSeen = time.Now()
		return true
	}

	if v.count >= rl.limit {
		return false
	}

	v.count++
	v.lastSeen = time.Now()
	return true
}

// Middleware envolve next, aplicando o limite por IP do cliente.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)

		if !rl.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"limite de requisições atingido, tente novamente mais tarde"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP descobre o IP real do cliente. Só confia em X-Forwarded-For quando o
// par imediato está em trusted; caso contrário usa o endereço da conexão.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remote := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}

	peer, err := netip.ParseAddr(remote)
	if err != nil || !rl.isTrusted(peer) {
		return remote
	}

	return rl.firstUntrustedXFF(r.Header.Get("X-Forwarded-For"), remote)
}

// isTrusted verifica se o par remoto está dentro dos CIDRs configurados.
func (rl *RateLimiter) isTrusted(addr netip.Addr) bool {
	for _, p := range rl.trusted {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// firstUntrustedXFF percorre o cabeçalho da direita para a esquerda, devolvendo
// o primeiro IP que não pertence a proxies confiáveis; se todos forem
// confiáveis, retorna fallback.
func (rl *RateLimiter) firstUntrustedXFF(header, fallback string) string {
	parts := strings.Split(header, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(parts[i])
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		if !rl.isTrusted(addr) {
			return ip
		}
	}
	return fallback
}
