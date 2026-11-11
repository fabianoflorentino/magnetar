// Package scraper extrai magnet links de páginas de torrents.
//
// A extração é genérica: qualquer página HTML que contenha âncoras
// "magnet:?" (ou magnets colados como texto puro) é suportada, sem
// depender da estrutura de um site específico.
package scraper

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Erros sentinela usados pelo scraper.
var (
	// ErrNoMagnetsFound indica que a página não contém magnet links.
	ErrNoMagnetsFound = errors.New("nenhum magnet link encontrado na página")
)

// HTTPClient define apenas o método Do para permitir propagação de contexto
// via requisição e injeção de mocks nos testes.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Compile-time check: http.Client deve satisfazer HTTPClient.
var _ HTTPClient = (*http.Client)(nil)

// MagnetLink representa um magnet link extraído com seu rótulo.
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

// userAgent é enviado nas requisições: muitos sites bloqueiam clientes
// não-navegador mesmo sem challenge completo.
const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

// anchorPattern casa âncoras cujo href é um magnet URI, capturando a URI e o
// texto interno da âncora.
var anchorPattern = regexp.MustCompile(`(?is)<a\b[^>]*href=["'](magnet:\?[^"']+)["'][^>]*>(.*?)</a>`)

// blockOpenPattern casa aberturas de blocos (p, td, li, div) que podem conter
// o texto de contexto de um magnet.
var blockOpenPattern = regexp.MustCompile(`(?is)<(?:p|td|li|div)\b[^>]*>`)

// textPattern casa magnets colados como texto puro (fóruns, listas simples).
// Hashes: v1 em hex (40) ou base32 (32); v2 em hex (64).
var textPattern = regexp.MustCompile(`(?i)\bmagnet:\?xt=urn:btih:[a-z0-9]{32,64}[^\s"'<>]*`)

// btihPattern extrai o info hash para deduplicação.
var btihPattern = regexp.MustCompile(`urn:btih:([a-zA-Z0-9]+)`)

// tagPattern remove tags HTML de rótulos.
var tagPattern = regexp.MustCompile(`<[^>]*>`)

// redirectHosts define os hosts aceitos como destino de redirecionamento,
// além do próprio host da requisição original. Sem isso, um open redirect em
// um host permitido permitiria fetch de hosts arbitrários (bypass do
// ALLOWED_HOSTS).
var (
	redirectMu    sync.RWMutex
	redirectHosts = map[string]struct{}{
		"apachetorrent.com":  {},
		"apachetorrents.com": {},
	}
)

// SetRedirectHosts define os hosts autorizados como destino de
// redirecionamento. Deve ser chamado na inicialização, antes de servir
// requisições, tipicamente com o mesmo valor de ALLOWED_HOSTS.
func SetRedirectHosts(hosts []string) {
	set := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		set[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
	}
	redirectMu.Lock()
	redirectHosts = set
	redirectMu.Unlock()
}

func redirectAllowed(host string) bool {
	redirectMu.RLock()
	defer redirectMu.RUnlock()
	_, ok := redirectHosts[strings.ToLower(host)]
	return ok
}

// DefaultHTTPClient é o cliente usado por Extract quando nenhum é injetado.
//
// Limita conexões ociosas por host para suportar concorrência sem estourar file
// descriptors, conforme a skill golang-performance.
var DefaultHTTPClient HTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("excesso de redirecionamentos")
		}
		if req.URL.Host != via[0].URL.Host && !redirectAllowed(req.URL.Host) {
			return fmt.Errorf("redirecionamento para host não autorizado: %s", req.URL.Host)
		}
		return nil
	},
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// Extract baixa a página em url e retorna todos os magnet links encontrados,
// filtrados por resolução. Usa o contexto para cancelamento e timeouts da
// requisição externa.
func Extract(ctx context.Context, url string) (*ExtractResult, error) {
	return extractWithClient(ctx, DefaultHTTPClient, url)
}

// extractWithClient é a variante testável de Extract: recebe o cliente HTTP
// para permitir injeção de mocks nos testes.
func extractWithClient(ctx context.Context, client HTTPClient, url string) (*ExtractResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao acessar a URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("servidor retornou 403 (possível bloqueio anti-bot, ex.: Cloudflare)")
		}
		return nil, fmt.Errorf("servidor retornou status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta: %w", err)
	}

	links := extractMagnetLinks(string(body))
	if len(links) == 0 {
		return nil, ErrNoMagnetsFound
	}

	result := &ExtractResult{
		Total:    len(links),
		Links:    links,
		Filtered: make(map[string][]MagnetLink, 3),
	}

	result.Filtered["720p"] = filterByResolution(links, "720")
	result.Filtered["1080p"] = filterByResolution(links, "1080")
	result.Filtered["2160p"] = filterByResolution(links, "2160")

	return result, nil
}

// extractMagnetLinks extrai todos os magnet links da página, deduplicando por
// info hash. O rótulo prefere o texto do bloco mais interno que contém a
// âncora (ex.: parágrafo de descrição) e cai para o texto da própria âncora.
func extractMagnetLinks(page string) []MagnetLink {
	blockOpens := blockOpenPattern.FindAllStringIndex(page, -1)

	links := make([]MagnetLink, 0, 8)
	seen := make(map[string]struct{}, 8)
	add := func(magnet, label string) {
		key := magnet
		if btih := btihPattern.FindStringSubmatch(magnet); btih != nil {
			key = strings.ToLower(btih[1])
		}
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		links = append(links, MagnetLink{Label: cleanLabel(label), Magnet: magnet})
	}

	for _, match := range anchorPattern.FindAllStringSubmatchIndex(page, -1) {
		magnet := html.UnescapeString(page[match[2]:match[3]])
		label := page[match[4]:match[5]]
		if context := blockContext(page, blockOpens, match[0]); context != "" {
			label = context
		}
		add(magnet, label)
	}

	for _, match := range textPattern.FindAllStringIndex(page, -1) {
		add(page[match[0]:match[1]], "")
	}

	return links
}

// maxContext limita o tamanho do texto de contexto usado como rótulo, para
// evitar rótulos gigantes em páginas com blocos muito grandes.
const maxContext = 512

// blockContext retorna o texto entre a abertura do bloco (p, td, li ou div)
// mais interna que precede a âncora e a própria âncora — o texto relevante
// (ex.: descrição do torrent), não menus distantes. Retorna "" quando não há
// bloco aplicável.
func blockContext(page string, blockOpens [][]int, anchorStart int) string {
	idx := sort.Search(len(blockOpens), func(i int) bool {
		return blockOpens[i][0] >= anchorStart
	})
	if idx == 0 {
		return ""
	}

	open := blockOpens[idx-1]
	if open[1] >= anchorStart {
		return ""
	}

	context := page[open[1]:anchorStart]
	if len(context) > maxContext {
		context = context[len(context)-maxContext:]
	}
	if cleanLabel(context) == "" {
		return ""
	}
	return context
}

// cleanLabel remove tags HTML, resolve entidades e colapsa espaços.
func cleanLabel(raw string) string {
	clean := tagPattern.ReplaceAllString(raw, "")
	clean = html.UnescapeString(clean)
	return strings.Join(strings.Fields(clean), " ")
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
