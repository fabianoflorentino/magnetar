package scraper

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func benchmarkHTML() string {
	return `
		<html>
			<body>
				<div id="versao-dual-audio">
					<p class="text-center">Dual Audio 720p <a href="magnet:?xt=urn:btih:abc123">link</a></p>
					<p class="text-center">Dual Audio 1080p <a href="magnet:?xt=urn:btih:def456">link</a></p>
					<p class="text-center">Legendado 720p <a href="magnet:?xt=urn:btih:ghi789">link</a></p>
				</div>
				<div id="versao-legendada">
					<p class="text-center">Legendado 1080p <a href="magnet:?xt=urn:btih:jkl012">link</a></p>
				</div>
			</body>
		</html>
	`
}

func BenchmarkExtractDualAudioSection(b *testing.B) {
	html := benchmarkHTML()
	b.ReportAllocs()
	for b.Loop() {
		extractDualAudioSection(html)
	}
}

func BenchmarkExtractMagnetLinks(b *testing.B) {
	section := benchmarkHTML()
	b.ReportAllocs()
	for b.Loop() {
		extractMagnetLinks(section)
	}
}

func BenchmarkFilterByResolution(b *testing.B) {
	links := []MagnetLink{
		{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:a"},
		{Label: "Dual Audio 1080p", Magnet: "magnet:?xt=urn:btih:b"},
		{Label: "Dual Audio 4K", Magnet: "magnet:?xt=urn:btih:c"},
		{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:d"},
	}
	b.ReportAllocs()
	for b.Loop() {
		filterByResolution(links, "720")
	}
}

type benchClient struct {
	body io.Reader
}

func (c *benchClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(c.body),
	}, nil
}

func BenchmarkExtract(b *testing.B) {
	client := &benchClient{body: strings.NewReader(benchmarkHTML())}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = extractWithClient(ctx, client, "https://apachetorrent.com/pagina/")
	}
}
