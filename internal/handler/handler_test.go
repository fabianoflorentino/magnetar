package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"magnetar/internal/scraper"
)

func stubExtract(result *scraper.ExtractResult, err error) extractFunc {
	return func(context.Context, string) (*scraper.ExtractResult, error) {
		return result, err
	}
}

type stubValidator struct {
	err error
}

func (v stubValidator) Validate(context.Context, string) error {
	return v.err
}

func newTestExtract(extract extractFunc, validate validateFunc) *Extract {
	if extract == nil {
		extract = stubExtract(&scraper.ExtractResult{Total: 1}, nil)
	}
	if validate == nil {
		validate = stubValidator{}.Validate
	}
	return NewExtract(extract, validate, slog.New(slog.DiscardHandler))
}

func TestHealth(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestExtractHandler(t *testing.T) {
	t.Parallel()

	okResult := &scraper.ExtractResult{
		Total: 2,
		Links: []scraper.MagnetLink{
			{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:abc"},
		},
		Filtered: map[string][]scraper.MagnetLink{
			"720p": {
				{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:abc"},
			},
		},
	}

	allowAll := stubValidator{}.Validate

	tests := []struct {
		name       string
		method     string
		body       string
		extract    extractFunc
		validate   validateFunc
		wantStatus int
	}{
		{
			name:       "success",
			method:     http.MethodPost,
			body:       `{"url":"https://apachetorrent.com/pagina/"}`,
			extract:    stubExtract(okResult, nil),
			validate:   allowAll,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       `{"url":"https://apachetorrent.com/pagina/"}`,
			validate:   allowAll,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "invalid JSON body",
			method:     http.MethodPost,
			body:       `{invalid}`,
			validate:   allowAll,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing url",
			method:     http.MethodPost,
			body:       `{}`,
			validate:   allowAll,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "url without scheme",
			method:     http.MethodPost,
			body:       `{"url":"apachetorrent.com/pagina"}`,
			validate:   allowAll,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid url format",
			method:     http.MethodPost,
			body:       `{"url":"http://"}`,
			validate:   allowAll,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "blocked by validator",
			method:     http.MethodPost,
			body:       `{"url":"https://evil.com/pagina/"}`,
			extract:    stubExtract(okResult, nil),
			validate:   stubValidator{err: errors.New("fora da lista")}.Validate,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "extract error",
			method:     http.MethodPost,
			body:       `{"url":"https://apachetorrent.com/pagina/"}`,
			extract:    stubExtract(nil, errors.New("connection refused")),
			validate:   allowAll,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "no magnets found",
			method:     http.MethodPost,
			body:       `{"url":"https://apachetorrent.com/pagina/"}`,
			extract:    stubExtract(nil, scraper.ErrNoMagnetsFound),
			validate:   allowAll,
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tt.method, "/api/extract", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			newTestExtract(tt.extract, tt.validate).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d; want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestExtractHandlerSuccessBody(t *testing.T) {
	t.Parallel()

	h := newTestExtract(stubExtract(&scraper.ExtractResult{Total: 1}, nil), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/extract", strings.NewReader(`{"url":"https://apachetorrent.com/pagina/"}`))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d; want %d", w.Code, http.StatusOK)
	}
	var result scraper.ExtractResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
}

func TestExtractHandlerGenericErrorBody(t *testing.T) {
	t.Parallel()

	h := newTestExtract(stubExtract(nil, errors.New("connection refused")), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/extract", strings.NewReader(`{"url":"https://apachetorrent.com/pagina/"}`))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got status %d; want %d", w.Code, http.StatusUnprocessableEntity)
	}
	if strings.Contains(w.Body.String(), "connection refused") {
		t.Error("técnico deve ser oculto do cliente")
	}
}
