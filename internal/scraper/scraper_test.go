package scraper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExtractDualAudioSection(t *testing.T) {
	t.Parallel()

	html := `
		<div id="versao-dual-audio">
			<p class="text-center">Dual Audio 720p <a href="magnet:?xt=urn:btih:abc123">link</a></p>
			<p class="text-center">Dual Audio 1080p <a href="magnet:?xt=urn:btih:def456">link</a></p>
			<p class="text-center">Legendado 1080p <a href="magnet:?xt=urn:btih:jkl012">link</a></p>
		</div>
		<div id="versao-legendada">
			<p class="text-center">Legendado 720p</p>
		</div>
	`

	section := extractDualAudioSection(html)
	if section == "" {
		t.Fatal("expected non-empty section")
	}
	if !strings.Contains(section, "versao-dual-audio") {
		t.Error("section should contain dual audio id")
	}
	if strings.Contains(section, "versao-legendada") {
		t.Error("section should not contain legendada id")
	}
}

func TestExtractDualAudioSection_NotFound(t *testing.T) {
	t.Parallel()

	html := `<div id="something-else">content</div>`
	section := extractDualAudioSection(html)
	if section != "" {
		t.Error("expected empty section for non-matching html")
	}
}

func TestExtractMagnetLinks(t *testing.T) {
	t.Parallel()

	section := `
		<div id="versao-dual-audio">
			<p class="text-center">Dual Audio 720p <a href="magnet:?xt=urn:btih:abc123">link</a></p>
			<p class="text-center">Dual Audio 1080p <a href="magnet:?xt=urn:btih:def456">link</a></p>
			<p class="text-center">Legendado 720p <a href="magnet:?xt=urn:btih:ghi789">link</a></p>
		</div>
	`

	links := extractMagnetLinks(section)
	if len(links) != 2 {
		t.Fatalf("expected 2 dual audio links, got %d", len(links))
	}

	for _, link := range links {
		if !strings.Contains(strings.ToUpper(link.Label), "DUAL") {
			t.Errorf("expected label to contain DUAL, got: %s", link.Label)
		}
		if !strings.HasPrefix(link.Magnet, "magnet:?") {
			t.Errorf("expected magnet URI, got: %s", link.Magnet)
		}
	}
}

func TestFilterByResolution(t *testing.T) {
	t.Parallel()

	links := []MagnetLink{
		{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:abc"},
		{Label: "Dual Audio 1080p", Magnet: "magnet:?xt=urn:btih:def"},
		{Label: "Dual Audio 4K", Magnet: "magnet:?xt=urn:btih:ghi"},
	}

	filtered720 := filterByResolution(links, "720")
	if len(filtered720) != 1 {
		t.Errorf("expected 1 link for 720p, got %d", len(filtered720))
	}

	filtered1080 := filterByResolution(links, "1080")
	if len(filtered1080) != 1 {
		t.Errorf("expected 1 link for 1080p, got %d", len(filtered1080))
	}
}

func TestCleanLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"span tag", "<span>Dual Audio 720p</span>", "Dual Audio 720p"},
		{"paragraph tag", `<p class="test">text</p>`, "text"},
		{"leading and trailing spaces", "  spaces  ", "spaces"},
		{"no tags", "no tags", "no tags"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cleanLabel(tt.input)
			if result != tt.expected {
				t.Errorf("cleanLabel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

type mockHTTPClient struct {
	resp *http.Response
	err  error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func htmlResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const extractHTML = `
	<html>
		<body>
			<div id="versao-dual-audio">
				<p class="text-center">Dual Audio 720p <a href="magnet:?xt=urn:btih:abc123">link</a></p>
				<p class="text-center">Dual Audio 1080p <a href="magnet:?xt=urn:btih:def456">link</a></p>
			</div>
			<div id="versao-legendada">
				<p class="text-center">Legendado 720p <a href="magnet:?xt=urn:btih:ghi789">link</a></p>
			</div>
		</body>
	</html>
`

func TestExtract(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{
			resp: htmlResponse(http.StatusOK, extractHTML),
		}

		result, err := extractWithClient(t.Context(), client, "https://apachetorrent.com/pagina/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Total != 2 {
			t.Errorf("expected total 2, got %d", result.Total)
		}
		if len(result.Links) != 2 {
			t.Errorf("expected 2 links, got %d", len(result.Links))
		}
		if len(result.Filtered["720p"]) != 1 {
			t.Errorf("expected 1 link in 720p filter, got %d", len(result.Filtered["720p"]))
		}
		if len(result.Filtered["1080p"]) != 1 {
			t.Errorf("expected 1 link in 1080p filter, got %d", len(result.Filtered["1080p"]))
		}
		if result.Links[0].Magnet != "magnet:?xt=urn:btih:abc123" {
			t.Errorf("unexpected first magnet: %s", result.Links[0].Magnet)
		}
	})

	t.Run("http error", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{
			err: fmt.Errorf("connection refused"),
		}

		_, err := extractWithClient(t.Context(), client, "https://apachetorrent.com/pagina/")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrSectionNotFound) {
			t.Fatal("expected transport error, got ErrSectionNotFound")
		}
	})

	t.Run("non ok status", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{
			resp: htmlResponse(http.StatusNotFound, "not found"),
		}

		_, err := extractWithClient(t.Context(), client, "https://apachetorrent.com/pagina/")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("dual audio section not found", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{
			resp: htmlResponse(http.StatusOK, `<html><div id="outra">x</div></html>`),
		}

		_, err := extractWithClient(t.Context(), client, "https://apachetorrent.com/pagina/")
		if !errors.Is(err, ErrSectionNotFound) {
			t.Errorf("expected ErrSectionNotFound, got %v", err)
		}
	})
}

func TestExtractURLTrailingSlash(t *testing.T) {
	t.Parallel()

	client := &mockHTTPClient{resp: htmlResponse(http.StatusOK, extractHTML)}
	if _, err := extractWithClient(t.Context(), client, "https://apachetorrent.com/pagina"); err != nil {
		t.Fatalf("URL sem barra final deve funcionar: %v", err)
	}
}
