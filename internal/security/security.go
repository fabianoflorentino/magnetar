// Package security valida URLs de destino contra SSRF (Server-Side Request Forgery).
package security

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// Validator restringe os hosts que o servidor pode buscar e bloqueia redes
// privadas, loopback e link-local para evitar requisições SSRF.
type Validator struct {
	allowedHosts map[string]struct{}
}

// NewValidator constrói um Validator a partir de uma lista de hosts permitidos.
// Um host sem barra permite o host exato e seus subdomínios.
func NewValidator(allowedHosts []string) *Validator {
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		host = strings.TrimSpace(strings.ToLower(host))
		if host != "" {
			allowed[host] = struct{}{}
		}
	}
	return &Validator{allowedHosts: allowed}
}

// Validate verifica se rawURL é um endpoint permitido e não resolve para redes
// internas. Retorna erro descritivo quando o destino não é seguro.
func (v *Validator) Validate(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("esquema não permitido: %s", u.Scheme)
	}

	if u.Hostname() == "" {
		return fmt.Errorf("host não pode ser vazio")
	}

	known := false
	host := strings.ToLower(u.Hostname())
	for allowed := range v.allowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("host %q não está na lista de permitidos", u.Hostname())
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil {
		return fmt.Errorf("falha ao resolver %s: %w", u.Hostname(), err)
	}

	for _, ip := range ips {
		if addr, ok := netip.AddrFromSlice(ip.IP); ok {
			if isBlocked(addr.Unmap()) {
				return fmt.Errorf("host %s resolve para rede bloqueada (%s)", u.Hostname(), ip.IP)
			}
		}
	}

	return nil
}

// isBlocked retorna true para endereços de redes não roteáveis na internet.
func isBlocked(addr netip.Addr) bool {
	return addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}
