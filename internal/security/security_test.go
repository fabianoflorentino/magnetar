package security

import (
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	v := NewValidator([]string{"apachetorrent.com"})

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"host permitido", "https://apachetorrent.com/pagina/", false},
		{"subdomínio permitido", "https://www.apachetorrent.com/pagina/", false},
		{"sem barra final", "https://apachetorrent.com", false},
		{"host fora da lista", "https://evil.com/pagina/", true},
		{
			"host com sufixo enganoso",
			"https://apachetorrent.com.evil.com/pagina/",
			true,
		},
		{"esquema não permitido", "ftp://apachetorrent.com/pagina/", true},
		{"sem esquema", "apachetorrent.com/pagina/", true},
		{"host vazio", "https:///pagina/", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := v.Validate(t.Context(), tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.url, err)
			}
		})
	}
}

func TestValidateBlocksInternalResolution(t *testing.T) {
	t.Parallel()

	v := NewValidator([]string{"localhost"})
	err := v.Validate(t.Context(), "http://localhost:8080/")
	if err == nil {
		t.Fatal("expected error for host resolving to loopback")
	}
}

func TestNewValidatorEmpty(t *testing.T) {
	t.Parallel()

	v := NewValidator(nil)
	tests := []struct {
		name string
		url  string
	}{
		{"qualquer host bloqueado", "https://example.com/"},
		{"host hostname sem lista", "https://apachetorrent.com/"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := v.Validate(t.Context(), tt.url); err == nil {
				t.Errorf("expected error for %q com lista vazia", tt.url)
			}
		})
	}
}
