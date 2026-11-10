package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func main() {
	url := "https://apachetorrent.com/westworld-3-temporada-baixar-torrent/"

	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao acessar a URL: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler resposta: %v\n", err)
		os.Exit(1)
	}

	html := string(body)

	dualSection := extractDualAudioSection(html)
	if dualSection == "" {
		fmt.Println("Seção Dual Áudio não encontrada.")
		os.Exit(1)
	}

	links := extractMagnetLinks(dualSection)

	links720p := filterByResolution(links, "720")
	links1080p := filterByResolution(links, "1080")

	fmt.Printf("Links magnéticos Dual Áudio encontrados: %d\n\n", len(links))

	fmt.Printf("=== 720p (%d links) ===\n\n", len(links720p))
	for i, link := range links720p {
		fmt.Printf("[%d] %s\n", i+1, link.label)
		fmt.Printf("    %s\n\n", link.magnet)
	}

	fmt.Printf("=== 1080p (%d links) ===\n\n", len(links1080p))
	for i, link := range links1080p {
		fmt.Printf("[%d] %s\n", i+1, link.label)
		fmt.Printf("    %s\n\n", link.magnet)
	}
}

type magnetLink struct {
	label  string
	magnet string
}

func extractDualAudioSection(html string) string {
	dualPattern := regexp.MustCompile(`(?i)id=['"]versao-dual-audio['"]`)
	legendadaPattern := regexp.MustCompile(`(?i)id=['"]versao-legendada['"]`)

	dualLoc := dualPattern.FindStringIndex(html)
	if dualLoc == nil {
		return ""
	}

	legendadaLoc := legendadaPattern.FindStringIndex(html[dualLoc[1]:])

	end := len(html)
	if legendadaLoc != nil {
		end = dualLoc[1] + legendadaLoc[0]
	}

	return html[dualLoc[0]:end]
}

func extractMagnetLinks(section string) []magnetLink {
	re := regexp.MustCompile(`(?s)<p\s+class=['"]text-center['"]>(.*?)<a\s+href=['"](magnet:\?[^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(section, -1)

	var links []magnetLink
	for _, match := range matches {
		label := cleanLabel(match[1])
		magnet := strings.TrimSpace(match[2])
		if strings.Contains(strings.ToUpper(label), "DUAL") || strings.Contains(strings.ToUpper(magnet), "DUAL") {
			links = append(links, magnetLink{label: label, magnet: magnet})
		}
	}
	return links
}

func cleanLabel(raw string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	clean := re.ReplaceAllString(raw, "")
	clean = strings.TrimSpace(clean)
	return clean
}

func filterByResolution(links []magnetLink, resolution string) []magnetLink {
	var filtered []magnetLink
	for _, link := range links {
		if strings.Contains(link.label, resolution) {
			filtered = append(filtered, link)
		}
	}
	return filtered
}
