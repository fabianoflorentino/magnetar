# magnetar

API que extrai **todos os magnet links de qualquer página de torrents** — em vez de clicar link por link, você envia a URL e recebe os magnets em JSON, com rate limit por IP e proteção contra SSRF. O nome vem do [magnetar](https://en.wikipedia.org/wiki/Magnetar) — estrela de nêutrons com o campo magnético mais intenso do universo.

## Resumo

O magnetar recebe a URL de uma página de torrents via API e devolve, em JSON, todos os magnet links encontrados — em âncoras `magnet:?` ou colados como texto — deduplicados por info hash, com rótulo extraído do texto ao redor e já filtrados por resolução (720p, 1080p e 2160p). A extração é genérica e não depende da estrutura de um site específico. Foi construído em Go com **zero dependências externas** (apenas a biblioteca padrão).

> **Limitação conhecida:** sites protegidos por challenge JavaScript (ex.: Cloudflare "Just a moment...") não podem ser buscados por nenhum cliente HTTP puro — a API retorna 403 com essa indicação. Exemplos atuais: eztvx.to e rargb.to.

## Como começar

Pré-requisitos: Go 1.27+ (para rodar localmente) ou Docker.

```bash
# Rodar a API localmente
make run

# Rodar os testes
make test

# Buildar o binário
make build
```

O servidor sobe em `http://localhost:8080`.

### Usando a API

```bash
curl -X POST http://localhost:8080/api/extract \
  -H "Content-Type: application/json" \
  -d '{"url":"https://apachetorrents.com/filme-exemplo/"}'
```

Resposta:

```json
{
  "total": 2,
  "links": [
    { "label": "DOWNLOAD TORRENT DUBLADO DUAL AUDIO 5.1 MKV WEB-DL 1080P", "magnet": "magnet:?xt=urn:btih:..." },
    { "label": "DOWNLOAD TORRENT LEGENDADO 5.1 MKV WEB-DL 720P", "magnet": "magnet:?xt=urn:btih:..." }
  ],
  "filtered": {
    "720p": [
      { "label": "DOWNLOAD TORRENT LEGENDADO 5.1 MKV WEB-DL 720P", "magnet": "magnet:?xt=urn:btih:..." }
    ],
    "1080p": [
      { "label": "DOWNLOAD TORRENT DUBLADO DUAL AUDIO 5.1 MKV WEB-DL 1080P", "magnet": "magnet:?xt=urn:btih:..." }
    ],
    "2160p": []
  }
}
```

Health check:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## Configuração

O serviço segue os princípios 12-Factor; toda configuração é feita por variáveis de ambiente.

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `PORT` | `8080` | Porta do servidor HTTP |
| `RATE_LIMIT` | `10` | Requisições por minuto por IP |
| `ALLOWED_ORIGINS` | `*` | Origens permitidas no CORS |
| `ALLOWED_HOSTS` | `apachetorrents.com,apachetorrent.com` | Hosts que a API pode buscar (SSRF); configure os sites que você quer extrair |
| `TRUSTED_PROXIES` | *(vazio)* | CIDRs de proxies reversos para confiar em `X-Forwarded-For` |

### Rodando com Docker

```bash
make docker-up   # sobe o serviço com docker compose
make docker-down # derruba
```

A imagem final pesa ~10 MB e executa com `GOMEMLIMIT` configurado para evitar OOM.

## Segurança

- **SSRF:** só hosts em `ALLOWED_HOSTS` são aceitos, destinos que resolvem para redes privadas/loopback/link-local são bloqueados antes da busca, e redirecionamentos cross-host só são seguidos para hosts também autorizados.
- **Rate limit:** `X-Forwarded-For` só é honrado quando o proxy está em `TRUSTED_PROXIES`; caso contrário, o cabeçalho é ignorado para evitar bypass por spoofing.
- **Erros:** erros técnicos são logados no servidor (JSON estruturado via `log/slog`) e mensagens genéricas vão para o cliente.

## Contribuindo

1. Clone o repositório.
2. Rode `go test ./... `, `go vet ./...` e `go test -race ./...`.
3. Abra um PR com mudanças pequenas e testadas.

## Licença

MIT — veja o arquivo [LICENSE](LICENSE).