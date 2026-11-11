package scraper

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func benchmarkPage() string {
	data, err := os.ReadFile("testdata/odisseia.html")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func BenchmarkExtractMagnetLinks(b *testing.B) {
	page := benchmarkPage()
	b.ReportAllocs()
	for b.Loop() {
		extractMagnetLinks(page)
	}
}

func BenchmarkFilterByResolution(b *testing.B) {
	links := []MagnetLink{
		{Label: "Dual Audio 720p", Magnet: "magnet:?xt=urn:btih:a"},
		{Label: "Dual Audio 1080p", Magnet: "magnet:?xt=urn:btih:b"},
		{Label: "WEB-DL 2160P", Magnet: "magnet:?xt=urn:btih:c"},
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
	client := &benchClient{body: strings.NewReader(benchmarkPage())}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = extractWithClient(ctx, client, "https://example.com/pagina/")
	}
}
