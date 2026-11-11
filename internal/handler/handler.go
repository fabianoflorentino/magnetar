// Package handler expõe os endpoints HTTP do magnetar.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"magnetar/internal/scraper"
)

// ExtractRequest é o corpo aceito pela rota POST /api/extract.
type ExtractRequest struct {
	URL string `json:"url"`
}

// ErrorResponse é o formato padrão de erro retornado pela API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// extractFunc extrai magnet links de uma URL, propagando o contexto do request.
type extractFunc func(ctx context.Context, url string) (*scraper.ExtractResult, error)

// validateFunc valida a segurança do destino antes da extração.
type validateFunc func(ctx context.Context, url string) error

// Extract é o handler de POST /api/extract com suas dependências injetadas.
type Extract struct {
	extract  extractFunc
	validate validateFunc
	logger   *slog.Logger
}

// NewExtract constrói o handler de extração com as dependências fornecidas.
func NewExtract(extract extractFunc, validate validateFunc, logger *slog.Logger) *Extract {
	return &Extract{extract: extract, validate: validate, logger: logger}
}

// ServeHTTP processa a requisição de extração de magnet links.
func (h *Extract) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req ExtractRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		h.writeError(w, "campo 'url' é obrigatório", http.StatusBadRequest)
		return
	}

	if err := validateTarget(rawURL); err != nil {
		h.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if err := h.validate(ctx, rawURL); err != nil {
		h.logger.Warn("destino bloqueado pela validação de segurança", "url", rawURL, "motivo", err.Error())
		h.writeError(w, "URL fora da lista de destinos permitidos", http.StatusForbidden)
		return
	}

	result, err := h.extract(ctx, rawURL)
	if err != nil {
		h.logger.Error("falha ao extrair magnet links", "url", rawURL, "erro", err)
		if errors.Is(err, scraper.ErrNoMagnetsFound) {
			h.writeError(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		h.writeError(w, "não foi possível extrair os magnet links desta página", http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Warn("falha ao escrever resposta de extração", "erro", err)
	}
}

// Health responde com o status da aplicação.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// validateTarget aplica validações sintáticas básicas sobre a URL informada.
func validateTarget(rawURL string) error {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return errors.New("URL deve começar com http:// ou https://")
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return errors.New("URL inválida")
	}

	return nil
}

// writeError serializa uma resposta de erro em JSON com o status informado.
func (h *Extract) writeError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: msg}); err != nil {
		h.logger.Warn("falha ao escrever resposta de erro", "erro", err)
	}
}
