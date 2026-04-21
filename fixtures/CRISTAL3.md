# CRISTAL3 — Análise do Sistema Implementado

**Data da análise**: 2026-04-21
**Escopo**: Componentes já implementados — Crawler (`site-research`) + MCP Server (`site-research-mcp`)
**Projeto**: CRISTAL (Consulta e Relatórios Inteligentes de Transparência Automatizado Local)
**Autor**: Rosemberg Maia Gomes (COTDI/STI/TRE-PI)

---

## 1. Visão Geral

O repositório `cristal3` contém a **Fase 1** completa do projeto CRISTAL: um pipeline de descoberta, crawl, sumarização e indexação do subsite de transparência do portal TRE-PI (`https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`), somado a um **servidor MCP** que expõe o catálogo resultante como ferramentas para clientes como Claude Desktop, Claude Code e Cowork.

As Fases 2–4 (extração de PDFs/CSVs, análise tabular, jobs assíncronos e insights) estão **apenas em especificação** — nenhum código dessas fases foi implementado ainda. O que existe hoje é:

| Componente | Status | Binário |
|---|---|---|
| `site-research` — Crawler + catálogo | ✅ Implementado | `bin/site-research` |
| `site-research-mcp` — Servidor MCP stdio | ✅ Implementado | `bin/site-research-mcp` |
| CRISTAL (Fases 2–4) — extractors, workers, tools cristal_* | 📝 Apenas spec | — |

Premissa arquitetural central: **um catálogo hierárquico com `mini_summary` bem-feitos basta para roteamento de pesquisa sem RAG, embeddings nem vector store.** O LLM roteia por título + mini-resumo; o conteúdo canônico vive no filesystem.

---

## 2. Linguagem e Stack

- **Go 1.25** (módulo `github.com/bergmaia/site-research`)
- **SQLite / FTS5** via driver CGO-free `modernc.org/sqlite` — permite binários estáticos e cross-compilação sem toolchain C
- **goquery** (`PuerkitoBio/goquery`) para parsing HTML
- **cobra** para CLI
- **robotstxt** (`temoto/robotstxt`) para honrar `robots.txt`
- **yaml.v3** para config
- **log/slog** (stdlib) para logging estruturado JSON
- Provider LLM: Anthropic (default `claude-haiku-4-5`), consumido via `ANTHROPIC_API_KEY`

Sem Docker, sem Redis, sem Python: todo o pipeline da Fase 1 é um único binário Go.

---

## 3. Arquitetura

### 3.1 Hexagonal (ports & adapters)

```
cmd/site-research          ← CLI (cobra), parse de flags, orquestra app.*
cmd/site-research-mcp      ← Entry point MCP stdio; lê env vars, valida catálogo, sobe server
    ↓
internal/app/*             ← Casos de uso: Discover, Crawl, Summarize, BuildCatalog,
                             Search, Inspect, Stats
    ↓
internal/domain/ports      ← Interfaces: Fetcher, HTMLExtractor, PageStore,
                             CatalogBuilder, SearchIndex, LLMProvider,
                             PageClassifier, URLCanonicalizer, SitemapSource, Clock
    ↓
internal/adapters/*        ← Implementações: httpfetch, htmlextract, fsstore,
                             catalog, sqlitefts, sitemap, llm
    ↑
internal/domain            ← Tipos puros: Page, Catalog, Metadata, Document,
                             MiniSummary, PageType, URLRef, ChildLink, Links, …
                             (sem dependências externas)
```

Regras:
- `internal/domain` não importa nada do projeto.
- Orquestradores em `internal/app` compõem adapters por injeção de dependência.
- `cmd/*` são as únicas camadas que conhecem cobra / env vars / stdin / stdout.

### 3.2 Filesystem como fonte de verdade

- Cada página crawleada vira `./data/<segmento1>/<segmento2>/.../_index.json` (schema v2).
- `./data/catalog.json` e `./data/catalog.sqlite` são **derivados**, totalmente rederiváveis com `site-research build-catalog`.
- Sem banco externo — o SQLite serve apenas como índice FTS5 sobre o catálogo.

---

## 4. Componente 1 — Crawler (`site-research`)

### 4.1 CLI (subcomandos)

```
site-research discover          # baixa sitemap, canonicaliza, filtra por scope
site-research crawl             # fetch → parse → extract → store _index.json
site-research summarize         # mini_summary via LLM (Anthropic)
site-research build-catalog     # consolida catalog.json + SQLite/FTS5
site-research search <query>    # busca textual no índice FTS
site-research inspect <path>    # mostra metadados de uma página
site-research stats             # totais, distribuição por tipo, orphans, stale
```

Arquivos: `cmd/site-research/cmd_*.go` (um por subcomando) + `main.go` (root cobra).

### 4.2 Pipeline canônico

```
sitemap.xml.gz
    ↓
┌──────────────────────────────────────────────────────────────┐
│ discover                                                       │
│  • Canonicalização RF-03 (segmento duplicado, sufixo numérico, │
│    copy_of, fragmentos, query params)                          │
│  • Filtra por cfg.scope.prefix                                 │
│  • Exclui padrões Plone (@@, ++theme++)                        │
│  • Saída: 561 URLs in-scope de 573 candidatas (TRE-PI)         │
└──────────────────────────────────────────────────────────────┘
    ↓
┌──────────────────────────────────────────────────────────────┐
│ crawl (incremental)                                            │
│  • httpfetch: rate-limit + jitter + retry + circuit-breaker    │
│  • honra robots.txt, Retry-After (429/503), ETag/If-None-Match │
│  • htmlextract: goquery → title, content, breadcrumb, links,   │
│    documents (PDF/CSV/XLSX), dates, keywords                   │
│  • classify: landing | article | listing | redirect | empty    │
│  • BFS complementar descobre páginas órfãs (não no sitemap)    │
│  • stale_since para páginas que desapareceram do sitemap       │
│  • Re-crawl preserva mini_summary quando content_hash inalterado│
│  • Saída: _index.json (schema v2) por página                   │
└──────────────────────────────────────────────────────────────┘
    ↓
┌──────────────────────────────────────────────────────────────┐
│ summarize                                                      │
│  • LLMProvider (Anthropic claude-haiku-4-5 default)            │
│  • Gera mini_summary de 1–2 linhas por página                  │
│  • Skip se hash inalterado desde o último run                  │
│  • Falhas isoladas não abortam o batch; custo reportado        │
└──────────────────────────────────────────────────────────────┘
    ↓
┌──────────────────────────────────────────────────────────────┐
│ build-catalog                                                  │
│  • Consolida árvore de _index.json em catalog.json             │
│  • (Re)cria catalog.sqlite com índice FTS5                     │
│    (unicode61, remove_diacritics 2) sobre title + mini_summary │
│    + full_text                                                 │
└──────────────────────────────────────────────────────────────┘
    ↓
search / inspect / stats    ← consumidores locais via CLI ou via MCP
```

### 4.3 Robustez do fetcher

`internal/adapters/httpfetch/` implementa:

- `ratelimit.go` — token bucket com **jitter ±ms** aleatório sobre o intervalo (evita bursts síncronos e parece mais humano).
- `retry.go` — retries com backoff; honra `Retry-After` em 429/503 (`honor_retry_after: true` no config).
- `circuit.go` — **circuit breaker**: abre após `max_consecutive_failures` (5), pausa `pause_minutes` (10), aborta após `abort_threshold` (3) falhas pós-retorno.
- `suspicious.go` — detecta páginas-bloqueio (Cloudflare/captcha/Access Denied) por padrões de título + tamanho mínimo de body; quando suspeita, marca e não sobrescreve página válida existente.
- `robots.go` — baixa e respeita `robots.txt`.

Critérios de aceite CA-13/14/15 têm testes de integração específicos (jitter, Retry-After, circuit breaker).

### 4.4 Schema v2 — `_index.json`

Cada página armazena (ver `internal/domain/page.go`):

| Grupo | Campos |
|---|---|
| Identidade | `url`, `canonical_url`, `title`, `description`, `section`, `lang` |
| Hierarquia | `breadcrumb[]`, `path_titles[]`, `links.children[]`, `links.internal[]`, `links.external[]` |
| Classificação | `page_type` (landing/article/listing/redirect/empty), `has_substantive_content` |
| Conteúdo | `content.summary`, `content.full_text`, `content.full_text_hash`, `content.content_hash`, `content.content_length`, `content.keywords_extracted[]` |
| Sumário LLM | `mini_summary.text`, `mini_summary.generated_at`, `mini_summary.model`, `mini_summary.source_hash`, `mini_summary.skipped` |
| Datas | `dates.content_date`, `dates.page_updated_at` |
| Metadata | `depth`, `extracted_at`, `etag`, `last_modified`, `http_status`, `parent_url`, `redirected_from`, `canonical_of`, `stale_since`, `crawler_version`, `discovered_via` (sitemap/link), `is_plone_copy`, `extraction_warnings[]` |
| Anexos | `documents[]` — PDF/CSV/XLSX/DOCX/ODS (URL, título, tipo, tamanho, contexto) — **sem download na Fase 1** |
| Taxonomia | `tags[]` |

### 4.5 Re-crawl incremental

Três mecanismos para evitar re-trabalho e preservar custo LLM:

1. **ETag / If-None-Match** → servidor retorna `304` → recarimba `extracted_at` sem reprocessar.
2. **content_hash** → body mudou mas conteúdo extraído é idêntico → preserva `mini_summary`.
3. **`summarize --force`** permite reabastecimento total quando desejado.

### 4.6 Stale + purge

Páginas que saem do sitemap e não são mais linkadas recebem `metadata.stale_since`. Permanecem no catálogo (ficam visíveis em `stats`). Remoção só com `crawl --purge-stale --confirm` (flag `--confirm` obrigatório como salvaguarda).

### 4.7 Configuração

Arquivo `config.yaml` na raiz do projeto. Seções:

- `scope` — seed URL + prefix
- `sitemap` — URL do sitemap (`.xml` ou `.xml.gz`)
- `crawler` — `rate_limit_per_second`, `jitter_ms`, `request_timeout_seconds`, `max_retries`, `respect_robots_txt`, `honor_retry_after`, `circuit_breaker.*`, `suspicious_response.*`
- `storage` — `data_dir`, `catalog_path`, `sqlite_path`
- `llm` — `provider`, `model`, `endpoint`, `api_key_env`, `concurrency`, `request_timeout_seconds`
- `recrawl` — `stale_retention_days`, `force_resummarize`

Override via `--config <path>` ou `SITE_RESEARCH_CONFIG`.

### 4.8 Números reais (TRE-PI, 2026-04)

- 2.367 URLs no sitemap global
- 573 URLs dentro do escopo de transparência
- 561 páginas crawleadas com sucesso
- 148 páginas com documentos anexos detectados
- 4 órfãs descobertas por BFS
- 23 páginas sem mini_summary (pendentes de summarize)

---

## 5. Componente 2 — MCP Server (`site-research-mcp`)

### 5.1 Responsabilidade

Fachada **fina** e **read-only** sobre o catálogo da Fase 1. Não implementa lógica de busca própria — delega para o FTS5 e para o `fsstore` já populado pelo crawler. Expõe o acervo como ferramentas MCP para que LLMs possam descobrir e inspecionar páginas durante uma conversa.

### 5.2 Transporte e protocolo

- **Transporte**: stdio (JSON-RPC 2.0 delimitado por `\n`).
- **Revisão MCP**: `2025-11-25` (única — sem negociação de fallback). Clientes com revisão diferente recebem erro explícito no `initialize`.
- **Capacidades anunciadas**: apenas `tools: {}`. Não expõe `resources`, `prompts`, `sampling` ou `roots`.
- **Cancelamento**: `notifications/cancelled` com `requestId` cancela o `context.Context` do handler; o resultado volta com `isError: true`.
- **Concorrência**: `tools/call` é despachado em goroutine por request; `sync.WaitGroup` drena handlers in-flight no shutdown; `recover()` captura panics em handlers e responde `isError` em vez de quebrar o protocolo.
- **Stdout é reservado exclusivamente a JSON-RPC** — logs vão para stderr (JSON, `log/slog`). Qualquer byte fora do protocolo em stdout é bug grave.

Arquivos: `internal/mcp/server.go`, `internal/mcp/protocol.go`, `internal/mcp/transport_stdio.go` + testes de contrato (`stdout_contract_test.go`).

### 5.3 Configuração (100% env vars)

Sem arquivo de config. Apenas `SITE_RESEARCH_DATA_DIR` é obrigatória; as
demais são derivadas por convenção e podem ser sobrescritas se necessário.

| Var | Obrigatória | Padrão | Descrição |
|---|---|---|---|
| `SITE_RESEARCH_DATA_DIR` | **sim** | — | Raiz da árvore de `_index.json`. **Única var obrigatória.** |
| `SITE_RESEARCH_CATALOG` | não | `{DATA_DIR}/catalog.json` | Path para `catalog.json`. Override se o arquivo estiver fora de `DATA_DIR`. |
| `SITE_RESEARCH_FTS_DB` | não | `{DATA_DIR}/catalog.sqlite` | Path para `catalog.sqlite`. Override se o arquivo estiver fora de `DATA_DIR`. |
| `SITE_RESEARCH_SCOPE_PREFIX` | não | lido de `catalog.root_url` | Prefixo do escopo (resolve paths relativos em `inspect_page`). Default: campo `root_url` do `catalog.json`. |
| `SITE_RESEARCH_LOG_LEVEL` | não (`info`) | `info` | `debug`/`info`/`warn`/`error` |
| `ANTHROPIC_API_KEY` | condicional | — | Lida mas ignorada na v0.x |

Ausência de `SITE_RESEARCH_DATA_DIR` → **exit 2**. Falhas de validação
(arquivo inexistente, schema version ≠ 2, FTS sem tabela `pages_fts`,
`root_url` vazio sem `SCOPE_PREFIX`) → **exit 1** com log de erro.

Validações de startup em `cmd/site-research-mcp/main.go`:

1. `data_dir` existe e é diretório
2. `catalog.json` existe (path default ou override)
3. `catalog.sqlite` existe (path default ou override)
4. SQLite abre em modo `ro` e `SELECT count(*) FROM pages_fts` funciona
5. `catalog.json` parseia como `domain.Catalog` com `SchemaVersion == 2`
6. `scope_prefix` resolvido: env `SCOPE_PREFIX` ou `catalog.RootURL`; ambos vazios → exit 1

### 5.4 Tools expostas

Três ferramentas read-only, definidas em `internal/tools/registry.go`:

#### `search`
Busca páginas por FTS5.
```json
{
  "query":   "string  (obrigatório, 1..500 chars, PT-BR natural)",
  "limit":   "integer (1..50, default 10)",
  "section": "string  (opcional, filtro case-insensitive, ≤120 chars)"
}
```
Handler em `internal/tools/search.go` → delega para `app.SearchHits` → `internal/adapters/sqlitefts`. Retorna markdown com título, mini-resumo e URL oficial por hit. Quando `section` é informada, o handler busca até 200 candidatos, filtra por seção e aplica `limit` depois.

#### `inspect_page`
Metadados completos de uma página.
```json
{
  "target": "string (obrigatório, URL completa ou path relativo ao scope prefix)"
}
```
Handler em `internal/tools/inspect.go` → delega para `app.Inspect` → `fsstore`. Retorna título, seção, breadcrumb, mini-resumo, `page_type`, `documents[]`, `dates`, URLs de filhos.

#### `catalog_stats`
Estatísticas agregadas (sem argumentos).
Handler em `internal/tools/stats.go`. Retorna total, distribuição por profundidade e `page_type`, top seções por volume, páginas sem mini-resumo, documentos detectados, páginas stale.

### 5.5 Formato de resposta

Toda tool retorna `CallToolResult` com `Content: [{type: "text", text: "<markdown>"}]`. Erros de domínio (query vazia, página não encontrada) não quebram o protocolo — voltam como resposta bem-sucedida com `isError: true` e mensagem em português em markdown.

Formatador em `internal/format/markdown.go`. Renderização humana para o LLM ler — não JSON bruto.

### 5.6 Clientes suportados

Documentado em `cmd/site-research-mcp/README.md`:

- **Claude Desktop** (macOS/Linux/Windows) — bloco `mcpServers` em `claude_desktop_config.json` com `command` + `env`.
- **Claude Code** — `claude mcp add site-research --env ... -- /usr/local/bin/site-research-mcp`.
- **Cowork** — mesmo modelo (comando absoluto + bloco `env`).

### 5.7 Performance

- **Cold start** alvo: < 500 ms (CA-12) em macOS/Linux com catálogo de 573 páginas. Medido ~7 ms em Apple M4 Pro com catálogo de fixture.
- **Binários** (target `-s -w -trimpath -X main.version=...`): < 25 MB por plataforma (orçamento RNF-04).
- **Plataformas** (GoReleaser): darwin_amd64, darwin_arm64, linux_amd64, linux_arm64, windows_amd64. Build local: `CGO_ENABLED=0` em todos os targets (driver SQLite é puro Go via `modernc.org/sqlite`).

### 5.8 Logs

- Tudo em **stderr**, formato JSON (`log/slog`).
- Campos mínimos: `time` (RFC 3339), `level`, `msg`.
- Campos adicionais: `tool`, `query`, `duration_ms`, `hits`, `request_id`, `error`.
- **API keys nunca aparecem nos logs**.

### 5.9 Testes

- Unitários: `internal/mcp/*_test.go` (protocolo, transport, server, contrato de stdout), `internal/tools/*_test.go`, `internal/app/*_test.go`, `internal/adapters/*/` (coverage ~73% em `internal/app`).
- Integração: `cmd/site-research-mcp/integration_test.go` sobe o servidor com fixtures e fala JSON-RPC real.
- Benchmark: `cmd/site-research-mcp/bench_test.go` (cold start).
- `go test ./...` roda sem rede e sem API keys.

---

## 6. Fluxo End-to-End

### 6.1 Bootstrap (uma vez por snapshot do portal)

```bash
export ANTHROPIC_API_KEY="sk-ant-..."

site-research discover                       # → 573 in-scope → 561 após canonical
site-research crawl                          # → ./data/<path>/_index.json
site-research summarize                      # → mini_summary por página
site-research build-catalog                  # → catalog.json + catalog.sqlite
```

### 6.2 Consumo via CLI

```bash
site-research search "diárias" --limit 20 --format json
site-research inspect contabilidade/balancetes --full
site-research stats --format json
```

### 6.3 Consumo via MCP (Claude Desktop / Claude Code / Cowork)

Cliente registra o binário:
```json
{
  "mcpServers": {
    "site-research": {
      "command": "/usr/local/bin/site-research-mcp",
      "env": {
        "SITE_RESEARCH_CATALOG":      "/path/to/data/catalog.json",
        "SITE_RESEARCH_FTS_DB":       "/path/to/data/catalog.sqlite",
        "SITE_RESEARCH_DATA_DIR":     "/path/to/data",
        "SITE_RESEARCH_SCOPE_PREFIX": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"
      }
    }
  }
}
```

Usuário pergunta em linguagem natural; o LLM chama `search`, depois `inspect_page` para detalhar uma URL, e eventualmente `catalog_stats` para se orientar sobre cobertura.

### 6.4 Re-crawl periódico

```bash
site-research crawl                          # incremental — ETag + content_hash
site-research summarize                      # só mexe em páginas cujo hash mudou
site-research build-catalog                  # recria catalog.json + SQLite
site-research crawl --purge-stale --confirm  # (opcional) remove stale > retenção
```

---

## 7. Papel no Projeto Maior (CRISTAL)

Este repo cobre **duas** das quatro fases planejadas:

| Fase | Componente | Status |
|---|---|---|
| **1** | Crawler + catálogo (este repo: `site-research`) | ✅ Implementado |
| **2** | Roteador semântico — seleciona tools relevantes por `mini_summary` | 📝 Spec |
| **3** | Servidor MCP (este repo: `site-research-mcp`) | ✅ Implementado |
| **4** | Consulta estruturada de datasets (extração de PDFs/CSVs, agregações, insights) | 📝 Spec (em `CRISTAL_README.md` e `SPEC_MIDLEWARE_CRISTAL.md`) |

As Fases 2 e 4, conforme spec, preveem um segundo servidor MCP em Python (`cristal_*` tools: `cristal_search`, `cristal_stats`, `cristal_extract_document`, `cristal_analyze`, `cristal_job_status`) com Celery + Redis e extractors de PDF (pdfplumber / pdftotext) e CSV (pandas). **Nenhum desses arquivos existe no repo ainda** — são decisões documentais.

---

## 8. Critérios de Aceite Atendidos (Fase 1)

Da tabela em `README.md`, 15 critérios — todos com status "FEITO":

- CA-01 a CA-04: discover, crawl, BFS de órfãs, canonicalização.
- CA-05: `page_type` acerta ≥80% em amostra de 20.
- CA-06: re-crawl preserva mini_summaries (>95% inalteração).
- CA-07: summarize resiliente a falhas isoladas; custo reportado.
- CA-08 a CA-10: catalog.json + SQLite/FTS5 + `search "diárias"` + inspeção manual de 20 mini-resumos.
- CA-11: `go test ./...` roda sem rede nem API keys.
- CA-12: README documenta instalação, fluxo, schema.
- CA-13 a CA-15: testes de jitter, Retry-After e circuit breaker.

---

## 9. Observações Arquiteturais

**Pontos fortes identificados**:

- Separação clara ports/adapters — adapter SQLite pode ser trocado sem mexer em `internal/app`.
- SQLite CGO-free viabiliza binário estático único por plataforma; distribuição via GitHub Release com GoReleaser.
- `stdout` exclusivo para JSON-RPC é contratado por teste (`stdout_contract_test.go`) — evita regressão comum em servidores MCP.
- `recover()` no handler + `WaitGroup` no loop = protocolo resiliente a panics e shutdown limpo.
- Cache implícito via `content_hash` no crawler economiza custo de LLM em re-runs.
- Cancelamento cooperativo via context propagado do `notifications/cancelled`.

**Limitações atuais (intencionais)**:

- **Sem download de documentos** — `documents[]` lista anexos mas não baixa PDF/CSV/XLSX. Essa responsabilidade é da Fase 4 (ainda não implementada).
- **Sem extração tabular** — nenhum pdfplumber/pandas no repo; apenas a spec descreve.
- **Sem jobs assíncronos** — tudo no servidor MCP é síncrono por design (cold start < 500 ms, handlers rápidos). Os jobs longos aparecerão só na Fase 4 em Python.
- **Single-tenant, single-portal** — escopo configurado via prefixo; não suporta múltiplos portais simultaneamente.
- **LLM só em batch (`summarize`)** — o runtime do MCP não chama LLM; a variável `ANTHROPIC_API_KEY` é lida mas ignorada na v0.x.

---

## 10. Referências no Repositório

- `README.md` — documentação do crawler (Fase 1)
- `BRIEF.md` — especificação formal da Fase 1, schema v2, critérios de aceite
- `PLANO_CRAWLER_CRISTAL.md` — plano de milestones e decisões de design do crawler
- `PLANO_IMPLEMENTACAO_MCP.md` — plano de implementação do servidor MCP
- `MCP_BRIEF.md` — spec do servidor MCP (RF-01..RF-04, RNF-*, CA-*)
- `CHANGES-v2.1.md` — mudanças no schema v2.1
- `cmd/site-research-mcp/README.md` — manual do binário MCP (instalação, env vars, troubleshooting, smoke test)
- `CRISTAL_README.md` + `SPEC_MIDLEWARE_CRISTAL.md` + `PROGRESSO.md` + `RETOMAR_AQUI.md` — documentação **da fase futura** (CRISTAL Python, Fases 2–4)
- `fixtures/sitemap.xml.gz` + `fixtures/_index.json` — ground truth para testes

---

**Versão binários atuais**: `v0.1.0` (dev)
**Schema catálogo**: v2
**Revisão MCP**: 2025-11-25
