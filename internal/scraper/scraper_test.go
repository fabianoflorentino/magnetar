package scraper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestExtractMagnetLinks_AnchorLabels(t *testing.T) {
	t.Parallel()

	page := `
		<html><body>
			<a href="magnet:?xt=urn:btih:abc123">Show.Name.S01.720p</a>
			<a class="magnet" href="magnet:?xt=urn:btih:def456">Show.Name.S01.1080p</a>
		</body></html>
	`

	links := extractMagnetLinks(page)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Label != "Show.Name.S01.720p" {
		t.Errorf("expected anchor text as label, got: %q", links[0].Label)
	}
	if !strings.HasPrefix(links[0].Magnet, "magnet:?xt=urn:btih:abc123") {
		t.Errorf("unexpected magnet: %s", links[0].Magnet)
	}
}

func TestExtractMagnetLinks_ContextLabel(t *testing.T) {
	t.Parallel()

	// Estrutura no estilo ApacheTorrent: o texto relevante está no parágrafo
	// que precede a âncora, dentro de blocos aninhados.
	page := `
		<html><body>
			<div class="download-block">
				<p class="download-versao">VERSÃO DUBLADA</p>
				<p class="download-desc">DOWNLOAD TORRENT DUBLADO DUAL AUDIO 5.1 MKV WEB-DL 1080P<br>
					<a href="magnet:?xt=urn:btih:ty763&amp;tr=udp%3A%2F%2Ftracker">Download Torrent</a>
				</p>
			</div>
		</body></html>
	`

	links := extractMagnetLinks(page)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if !strings.Contains(links[0].Label, "DUAL AUDIO 5.1 MKV WEB-DL 1080P") {
		t.Errorf("expected paragraph text as label, got: %q", links[0].Label)
	}
	if strings.Contains(links[0].Label, "Download Torrent") {
		t.Errorf("anchor text should not win over paragraph context: %q", links[0].Label)
	}
}

func TestExtractMagnetLinks_DedupeByHash(t *testing.T) {
	t.Parallel()

	page := `
		<a href="magnet:?xt=urn:btih:ABC123&tr=udp://tracker1">Nome 720p</a>
		<a href="magnet:?xt=urn:btih:abc123&tr=udp://tracker2">Nome 720p duplicado</a>
		<a href="magnet:?xt=urn:btih:def456">Outro 1080p</a>
	`

	links := extractMagnetLinks(page)
	if len(links) != 2 {
		t.Fatalf("expected 2 links after dedupe by info hash, got %d", len(links))
	}
}

func TestExtractMagnetLinks_UnescapeEntities(t *testing.T) {
	t.Parallel()

	page := `<p class="download-desc">DUAL &amp; AUDIO 1080P<br><a href="magnet:?xt=urn:btih:abc123&amp;tr=udp%3A%2F%2Fx">dl</a></p>`

	links := extractMagnetLinks(page)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if strings.Contains(links[0].Magnet, "&amp;") {
		t.Errorf("magnet must have entities unescaped: %s", links[0].Magnet)
	}
	if !strings.Contains(links[0].Magnet, "&tr=") {
		t.Errorf("magnet should contain unescaped tracker param: %s", links[0].Magnet)
	}
	if !strings.Contains(links[0].Label, "DUAL & AUDIO") {
		t.Errorf("label should have entities unescaped: %q", links[0].Label)
	}
}

func TestExtractMagnetLinks_PlainText(t *testing.T) {
	t.Parallel()

	page := `
		<html><body>
			<p>Links:</p>
			<p>magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=torrent</p>
		</body></html>
	`

	links := extractMagnetLinks(page)
	if len(links) != 1 {
		t.Fatalf("expected 1 plain text link, got %d", len(links))
	}
	if !strings.HasPrefix(links[0].Magnet, "magnet:?xt=urn:btih:") {
		t.Errorf("unexpected magnet: %s", links[0].Magnet)
	}
}

func TestExtractMagnetLinks_EmptyPage(t *testing.T) {
	t.Parallel()

	links := extractMagnetLinks(`<html><body><p>sem torrents aqui</p></body></html>`)
	if len(links) != 0 {
		t.Errorf("expected no links, got %d", len(links))
	}
}

func TestFilterByResolution(t *testing.T) {
	t.Parallel()

	links := []MagnetLink{
		{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:abc"},
		{Label: "Dual Audio 1080p", Magnet: "magnet:?xt=urn:btih:def"},
		{Label: "WEB-DL 2160P", Magnet: "magnet:?xt=urn:btih:ghi"},
		{Label: "Dual Audio 4K", Magnet: "magnet:?xt=urn:btih:jkl"},
	}

	if got := len(filterByResolution(links, "720")); got != 1 {
		t.Errorf("expected 1 link for 720p, got %d", got)
	}
	if got := len(filterByResolution(links, "1080")); got != 1 {
		t.Errorf("expected 1 link for 1080p, got %d", got)
	}
	if got := len(filterByResolution(links, "2160")); got != 1 {
		t.Errorf("expected 1 link for 2160p, got %d", got)
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
		{"collapses whitespace", "a\n\t b   c", "a b c"},
		{"unescapes entities", "Dual &amp; Audio", "Dual & Audio"},
	}

	for _, tt := range tests {
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

const genericPage = `
	<html><body>
		<table>
			<tr>
				<td>Show.Name.S01.720p</td>
				<td><a href="magnet:?xt=urn:btih:abc123&amp;tr=udp%3A%2F%2Ftracker">Magnet</a></td>
			</tr>
		</table>
		<p class="download-desc">DOWNLOAD DUAL AUDIO 1080P<br><a href="magnet:?xt=urn:btih:def456&amp;tr=udp%3A%2F%2Ftracker">Download Torrent</a></p>
		<p>magnet:?xt=urn:btih:abc123&amp;dn=duplicado</p>
	</body></html>
`

func TestExtract(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{resp: htmlResponse(http.StatusOK, genericPage)}

		result, err := extractWithClient(t.Context(), client, "https://example.com/pagina")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Total != 2 {
			t.Errorf("expected total 2 (dedupe), got %d", result.Total)
		}
		if len(result.Filtered["720p"]) != 0 {
			t.Errorf("expected 0 links in 720p filter, got %d", len(result.Filtered["720p"]))
		}
		if len(result.Filtered["1080p"]) != 1 {
			t.Errorf("expected 1 link in 1080p filter, got %d", len(result.Filtered["1080p"]))
		}
		for _, link := range result.Links {
			if strings.Contains(link.Magnet, "&amp;") {
				t.Errorf("magnet must be unescaped: %s", link.Magnet)
			}
			if strings.Contains(link.Magnet, "def456") && link.Label != "DOWNLOAD DUAL AUDIO 1080P" {
				t.Errorf("expected context label, got: %q", link.Label)
			}
		}
	})

	t.Run("http error", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{err: fmt.Errorf("connection refused")}

		_, err := extractWithClient(t.Context(), client, "https://example.com/pagina")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrNoMagnetsFound) {
			t.Fatal("expected transport error, got ErrNoMagnetsFound")
		}
	})

	t.Run("non ok status", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{resp: htmlResponse(http.StatusNotFound, "not found")}

		_, err := extractWithClient(t.Context(), client, "https://example.com/pagina")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("forbidden hints anti-bot blocking", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{resp: htmlResponse(http.StatusForbidden, "challenge")}

		_, err := extractWithClient(t.Context(), client, "https://example.com/pagina")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("expected 403 hint in error, got: %v", err)
		}
	})

	t.Run("no magnets found", func(t *testing.T) {
		t.Parallel()
		client := &mockHTTPClient{resp: htmlResponse(http.StatusOK, `<html><div id="outra">x</div></html>`)}

		_, err := extractWithClient(t.Context(), client, "https://example.com/pagina")
		if !errors.Is(err, ErrNoMagnetsFound) {
			t.Errorf("expected ErrNoMagnetsFound, got %v", err)
		}
	})
}

// TestCanaryApacheTorrent valida a extração contra cópias reais das páginas
// atuais do site. Se o site mudar a estrutura e a extração quebrar, este
// teste aponta o drift.
func TestCanaryApacheTorrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		file          string
		total         int
		want720       int
		want1080      int
		firstContains string
	}{
		{
			name:          "coyote (dublada, 1080p)",
			file:          "testdata/coyote.html",
			total:         1,
			want720:       0,
			want1080:      1,
			firstContains: "DUAL AUDIO",
		},
		{
			name:          "odisseia (dublada + legendada)",
			file:          "testdata/odisseia.html",
			total:         4,
			want720:       1,
			want1080:      3,
			firstContains: "DUBLADO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}

			links := extractMagnetLinks(string(data))
			if len(links) != tt.total {
				t.Fatalf("expected %d links, got %d", tt.total, len(links))
			}

			if !strings.Contains(strings.ToUpper(links[0].Label), tt.firstContains) {
				t.Errorf("first label should mention %q, got: %q", tt.firstContains, links[0].Label)
			}

			for _, link := range links {
				if strings.Contains(link.Magnet, "&amp;") {
					t.Errorf("magnet must have entities unescaped: %s", link.Magnet)
				}
			}

			if got := len(filterByResolution(links, "720")); got != tt.want720 {
				t.Errorf("expected %d links in 720p, got %d", tt.want720, got)
			}
			if got := len(filterByResolution(links, "1080")); got != tt.want1080 {
				t.Errorf("expected %d links in 1080p, got %d", tt.want1080, got)
			}
		})
	}
}
