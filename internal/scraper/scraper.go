// Package scraper contém a lógica de extração de magnet links de páginas do ApacheTorrent.
package scraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Erros sentinela usados pelo scraper.
var (
	// ErrSectionNotFound indica que a página não possui a seção de Dual Áudio.
	ErrSectionNotFound = errors.New("seção Dual Áudio não encontrada na página")
)

// HTTPClient define apenas o método Do para permitir propagação de contexto
// via requisição e injeção de mocks nos testes.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Compile-time check: http.Client deve satisfazer HTTPClient.
var _ HTTPClient = (*http.Client)(nil)

// MagnetLink representa um magnet link extraído com seu rótulo de resolução.
type MagnetLink struct {
	Label  string `json:"label"`
	Magnet string `json:"magnet"`
}

// ExtractResult agrega o total, os links e a filtragem por resolução.
type ExtractResult struct {
	Total    int                     `json:"total"`
	Links    []MagnetLink            `json:"links"`
	Filtered map[string][]MagnetLink `json:"filtered"`
}

var dualAudioPattern = regexp.MustCompile(`(?i)id=['"]versao-dual-audio['"]`)
var legendadaPattern = regexp.MustCompile(`(?i)id=['"]versao-legendada['"]`)
var magnetLinkPattern = regexp.MustCompile(`(?s)<p\s+class=['"]text-center['"]>(.*?)<a\s+href=['"](magnet:\?[^'"]+)['"]`)
var tagPattern = regexp.MustCompile(`<[^>]*>`)

// DefaultHTTPClient é o cliente usado por Extract quando nenhum é injetado.
//
// Limita conexões ociosas por host para suportar concorrência sem estourar file
// descriptors, conforme a skill golang-performance.
var DefaultHTTPClient HTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// Extract baixa a página em url, localiza a seção de Dual Áudio e retorna os
// magnet links encontrados, filtrados por resolução. usa o contexto para
// cancelamento e timeouts da requisição externa.
func Extract(ctx context.Context, url string) (*ExtractResult, error) {
	return extractWithClient(ctx, DefaultHTTPClient, url)
}

// extractWithClient é a variante testável de Extract: recebe o cliente HTTP
// para permitir injeção de mocks nos testes.
func extractWithClient(ctx context.Context, client HTTPClient, url string) (*ExtractResult, error) {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para %s: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao acessar a URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("servidor retornou status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta: %w", err)
	}

	dualSection := extractDualAudioSection(string(body))
	if dualSection == "" {
		return nil, ErrSectionNotFound
	}

	links := extractMagnetLinks(dualSection)

	result := &ExtractResult{
		Total:    len(links),
		Links:    links,
		Filtered: make(map[string][]MagnetLink, 2),
	}

	result.Filtered["720p"] = filterByResolution(links, "720")
	result.Filtered["1080p"] = filterByResolution(links, "1080")

	return result, nil
}

// extractDualAudioSection isola o trecho HTTP entre a seção dual-audio e a
// legendada, para que apenas o conteúdo relevante seja processado.
func extractDualAudioSection(html string) string {
	dualLoc := dualAudioPattern.FindStringIndex(html)
	if dualLoc == nil {
		return ""
	}

	end := len(html)
	if legendadaLoc := legendadaPattern.FindStringIndex(html[dualLoc[1]:]); legendadaLoc != nil {
		end = dualLoc[1] + legendadaLoc[0]
	}

	return html[dualLoc[0]:end]
}

// extractMagnetLinks percorre os parágrafos da seção e mantém apenas os links
// cujo rótulo ou magnet indique Dual Áudio.
func extractMagnetLinks(section string) []MagnetLink {
	matches := magnetLinkPattern.FindAllStringSubmatch(section, -1)

	links := make([]MagnetLink, 0, len(matches))
	for _, match := range matches {
		label := cleanLabel(match[1])
		magnet := strings.TrimSpace(match[2])
		if strings.Contains(strings.ToUpper(label), "DUAL") || strings.Contains(strings.ToUpper(magnet), "DUAL") {
			links = append(links, MagnetLink{Label: label, Magnet: magnet})
		}
	}
	return links
}

// cleanLabel remove as tags HTML e espaços em excesso do rótulo extraído.
func cleanLabel(raw string) string {
	clean := tagPattern.ReplaceAllString(raw, "")
	return strings.TrimSpace(clean)
}

// filterByResolution retorna os links cujo rótulo contém a resolução informada.
func filterByResolution(links []MagnetLink, resolution string) []MagnetLink {
	filtered := make([]MagnetLink, 0, len(links))
	for _, link := range links {
		if strings.Contains(link.Label, resolution) {
			filtered = append(filtered, link)
		}
	}
	return filtered
}
