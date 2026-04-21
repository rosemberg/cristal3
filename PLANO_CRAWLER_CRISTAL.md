# PLAN — Plano de Implementação da Fase 1 (Crawler + Catálogo)

## 0. Confirmação de Entendimento

Absorvidos `README.md`, `BRIEF.md`, `.claude/README.md`, `.claude/settings.json`, `fixtures/README.md`, `fixtures/_index.json` (schema v1) e `fixtures/sitemap.xml.gz`.

**Escopo único**: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas` e descendentes. Sitemap carrega 573 URLs em escopo (valor real, conferido). Fora do escopo: outras seções do portal, subdomínios, download de anexos, SPA, auth.

**Arquitetura**: Go 1.22+, hexagonal — domínio puro sem dependências externas, ports como interfaces, adapters implementam I/O (HTTP, filesystem, SQLite, LLM). CGO desabilitado (restrição herdada de `.claude/settings.json`).

**Diff schema v1 → v2** (o mais relevante do que vi no fixture):

| Campo / característica | v1 (fixture) | v2 (a produzir) |
|---|---|---|
| `$schema` | `page-node-v1` | `page-node-v2` + `schema_version: 2` |
| `canonical_url` | ausente | obrigatório |
| `section` / `path_titles` | ausente | obrigatório |
| `page_type`, `has_substantive_content` | ausente | obrigatório (RF-05) |
| `content.full_text_hash`, `content.content_hash`, `content.content_length`, `content.keywords_extracted` | ausente | obrigatório (RF-04) |
| `mini_summary` | ausente | objeto com `text`, `model`, `source_hash`, `skipped`, `generated_at` (RF-09) |
| `dates.content_date`, `dates.page_updated_at` | ausente | obrigatório (RF-06) |
| `metadata.depth` | valor lixo (`262`) | correto, baseado no path |
| `metadata.etag`, `discovered_via`, `is_plone_copy`, `redirected_from`, `crawler_version`, `fetch_duration_ms`, `extraction_warnings` | ausente | obrigatório |
| `links.*.type` | sempre `"unknown"` | classificado (externo separado de interno-fora-do-escopo) |
| `documents` | sempre `[]` | populado com metadados leves + `context_text` |
| `tags` | sempre `[]` | extraído de Plone quando presente |

**Números reais do sitemap** (validados no fixture): 2.367 URLs totais no portal, 573 em escopo, 96 URLs com padrão `segmento/segmento` duplicado (17%), 159 com sufixo numérico `-N` (28%), 14 com `copy_of` (BRIEF diz 10 — pequena divergência a esclarecer), 0 com `@@` ou `++theme++` em escopo. Distribuição de profundidade bate exatamente com o schema do catalog.json do BRIEF.

**12 critérios de aceite** entendidos — cada milestone abaixo mapeia para pelo menos um.

---

## 1. Plano em Milestones

Proposta: **7 milestones**, da fundação à documentação final. Cada uma é testável em isolamento e tem critério de pronto objetivo.

### M1 — Fundação (domínio, ports, CLI skeleton, config, logging)

**Objetivo**: estabelecer o esqueleto hexagonal com todos os contratos domínio↔adapter definidos e um binário que compila, aceita subcomandos e carrega config YAML, mas ainda não faz I/O real.

**Escopo (in)**:
- `go.mod` no path a ser decidido (ver Decisões em Aberto #1).
- Pacote `internal/domain/` com entidades (`Page`, `Content`, `Metadata`, `Links`, `Document`, `MiniSummary`, `Dates`, `SitemapEntry`, `CatalogEntry`, `CrawlReport`, `SummarizeReport`, `URLRef`) — **sem imports externos**.
- Pacote `internal/domain/ports/` com interfaces: `Fetcher`, `SitemapSource`, `HTMLExtractor`, `URLCanonicalizer`, `PageClassifier`, `PageStore`, `CatalogBuilder`, `SearchIndex`, `LLMProvider`, `Clock`.
- `internal/config/` carrega YAML + resolve env vars por nome (nunca armazena segredos no struct de modo logável).
- `internal/logging/` usando `log/slog` (stdlib) — handler JSON, níveis configuráveis.
- `cmd/site-research/main.go` com subcomandos `discover`, `crawl`, `summarize`, `build-catalog`, `search`, `inspect`, `stats` — todos retornam "not implemented" por enquanto.
- Flag `--config` global.
- `config.yaml.example` commitado na raiz (sem chaves reais).

**Escopo (out)**: qualquer I/O real; extração; LLM.

**Deliverables concretos**:
- `go.mod`, `go.sum`
- `internal/domain/*.go`
- `internal/domain/ports/ports.go`
- `internal/config/config.go` + teste
- `internal/logging/logger.go`
- `cmd/site-research/main.go` + arquivos por subcomando em `cmd/site-research/cmd_*.go`
- `config.yaml.example`

**Dependências**: nenhuma (primeira milestone).

**Critério de pronto**:
- `go build ./...` produz `./bin/site-research`
- `./bin/site-research --help` lista os 7 subcomandos
- `./bin/site-research discover` imprime "not implemented" e retorna exit code não-zero de forma controlada
- `go test ./...` passa (testes do config loader)
- `go vet ./...` limpo

**Complexidade**: baixa.

**Riscos**: divergência posterior entre ports e uso real (refatoração inevitável). Mitigação: desenhar ports pensando nos fluxos de M2–M6, não só no que M1 precisa.

---

### M2 — Descoberta via sitemap + canonicalização de URLs (`discover`)

**Objetivo**: entregar o subcomando `discover` funcional — baixar o sitemap global, filtrar pelo escopo, canonicalizar cada URL e imprimir a lista final. Primeira validação ponta-a-ponta sobre dados reais.

**Escopo (in)**:
- `internal/canonical/canonicalizer.go` implementando TODAS as regras do RF-03:
  - Remoção de fragmentos
  - Remoção de query params de tracking (`utm_*`, `gclid`, `fbclid`, `_ga`, `portal_form_id`)
  - Política de trailing slash (ver Decisão em Aberto #2)
  - Exclusão de URLs com `@@` ou `++theme++`
  - Sinalização (não exclusão) do padrão Plone `segmento/segmento` — retornando pair `{canonical_candidate, original}` para que a deduplicação real seja feita em M4 via `content_hash`
  - Preservação de sufixo numérico `-N`
  - Detecção do padrão `copy_of` (flag `is_plone_copy`)
- `internal/adapters/sitemap/` com fetcher HTTP simples e parser XML (`encoding/xml`, `compress/gzip`).
- Filtro por prefixo configurado no YAML.
- Subcomando `discover` que chama `SitemapSource.Fetch()` → canonicaliza → filtra → imprime uma URL por linha em stdout e um resumo em stderr (total, filtradas, inválidas).
- Flag `--format=text|json` opcional para saída estruturada.
- Teste unitário do canonicalizer com >=20 casos (cada variante do RF-03).
- Teste do parser do sitemap usando `fixtures/sitemap.xml.gz` como ground truth (sem rede).

**Escopo (out)**: crawling de páginas, deduplicação por `content_hash` (isso exige ter o conteúdo).

**Deliverables concretos**:
- `internal/canonical/canonicalizer.go` + `canonicalizer_test.go`
- `internal/adapters/sitemap/fetcher.go`, `parser.go` + testes usando fixture local
- `internal/app/discover.go` (serviço que orquestra o subcomando)
- `cmd/site-research/cmd_discover.go`

**Dependências**: M1 (domain ports, config, CLI skeleton).

**Critério de pronto**:
- `discover` com fixture local (flag `--from-file fixtures/sitemap.xml.gz`) retorna exatamente 573 URLs → **mapeia CA-1** parcialmente (CA-1 exige 400–800 URLs)
- `discover` contra o portal real (quando aprovado) retorna número compatível
- Todos os casos de canonicalização do RF-03 cobertos por teste (CA-4)
- `go test ./internal/canonical/...` e `./internal/adapters/sitemap/...` verdes

**Complexidade**: média.

**Riscos**:
- Política de trailing slash afetando match de prefixo — por isso decisão tem que ser tomada antes de codar (ver Decisões em Aberto).
- Canonicalização agressiva pode excluir URLs válidas — mitigado por testes tabulados contra amostra real.

---

### M3 — Infra de crawl: fetcher HTTP, rate limit, retries, robots, store filesystem

**Objetivo**: construir a camada que busca bytes + headers de uma URL com disciplina de cliente responsável, e o adapter de filesystem que materializa a árvore de páginas, sem ainda extrair conteúdo.

**Escopo (in)**:
- `internal/adapters/httpfetch/client.go`: wrapper de `net/http` com:
  - User-Agent configurável (default do BRIEF)
  - Timeout configurável
  - ETag / If-Modified-Since in/out
  - Header `Cache-Control: no-cache` (RNF-06)
  - Redirect handling (registra `redirected_from`)
- `internal/adapters/httpfetch/ratelimit.go`: limiter global (`golang.org/x/time/rate`) — 1 req/s default, configurável.
- `internal/adapters/httpfetch/retry.go`: backoff exponencial com jitter (base 500ms, factor 2, max 10s, até 3 tentativas), retry apenas em 5xx/429/erros de rede (ver Decisão em Aberto #3).
- `internal/adapters/httpfetch/robots.go`: parser de robots.txt (lib `github.com/temoto/robotstxt`) cacheado por host.
- `internal/adapters/fsstore/store.go`: implementa `PageStore` do domínio:
  - Layout hierárquico espelhando segmentos do path da URL (`./data/<segmento1>/<segmento2>/.../_index.json`)
  - Write-then-rename (escreve `_index.json.tmp` + `os.Rename`) — RNF-05
  - Leitura idempotente
  - Helper para caminhar a árvore.
- Testes com servidor HTTP local (`httptest.NewServer`) cobrindo rate limit, retry, 304, 404, redirect.

**Escopo (out)**: extração de HTML para schema v2 (fica em M4); BFS complementar e incremental (M5).

**Deliverables concretos**:
- `internal/adapters/httpfetch/*.go`
- `internal/adapters/fsstore/store.go`
- Testes com ≥ 80% de cobertura em ambos os pacotes.

**Dependências**: M1 (ports).

**Critério de pronto**:
- Teste de integração: fetcher contra servidor mock com 3 páginas mockadas → grava 3 `_index.json` parciais (apenas com bytes brutos e headers) na árvore.
- Retry real disparado por 503 consecutivos e sucesso no 3ª tentativa.
- 304 NOT MODIFIED resulta em "skip" sem sobrescrever arquivo.
- robots.txt negando `/transparencia-e-prestacao-de-contas/auditoria` → fetcher recusa essa URL.

**Complexidade**: média-alta.

**Riscos**:
- Interação do rate limiter com retries e concorrência precisa ser desenhada com cuidado para não burlar o rate limit durante retries.
- Redirects em cadeia podem cair fora do escopo — decidir: seguir e registrar, ou recusar? Recomendo **seguir até 5x e aplicar filtro de escopo ao final**.

---

### M4 — Extração HTML para schema v2 + classificação + datas + comando `crawl` ponta-a-ponta

**Objetivo**: transformar bytes brutos em um `_index.json` válido segundo schema v2 — extrair title, breadcrumb, conteúdo, hierarquia, anexos, tags, datas, keywords, classificar `page_type` e gravar estrutura final. Subcomando `crawl` totalmente funcional para o caminho feliz.

**Escopo (in)**:
- `internal/adapters/htmlextract/extractor.go`: usa **goquery** (`github.com/PuerkitoBio/goquery`) para extrair todos os campos do RF-04.
- Extratores dedicados:
  - `breadcrumb.go` (Plone: `.breadcrumb`, `ol.breadcrumb`, data-microdata)
  - `content.go` (identificar `#content-core`, `article`, fallback heurístico; remover boilerplate do Plone)
  - `links.go` (separar `children`, `internal`, `external`, segundo definição do RF-04)
  - `documents.go` (detectar PDF/CSV/XLSX/DOCX/ODS por extensão + captura de `context_text` de ~200 chars)
  - `dates.go` (RF-06: meta `DC.date`, `DC.date.created`, `DC.date.modified`, `article:published_time`, `article:modified_time`)
  - `keywords.go` (TF simples sobre `full_text`, stopwords pt-BR)
- `internal/classify/classifier.go`: heurísticas do RF-05 (`landing` / `article` / `listing` / `redirect` / `empty`) e deriva `has_substantive_content`. Pura — entrada é `Page` parcialmente preenchido.
- `internal/app/crawl.go`: orquestrador que consome lista do `discover`, para cada URL chama `Fetcher` → `HTMLExtractor` → `PageClassifier` → `PageStore`. Reporta progresso via logs estruturados.
- `internal/adapters/fsstore/` estendido para gravar schema v2 completo.
- Flag `--dry-run` no `crawl` (simula, não grava).
- Detecção e marcação de `is_plone_copy` (copiada de M2) dentro do extrator para consistência.
- Deduplicação por `content_hash` para o padrão `/a/b/b` vs `/a/b/`: pós-processamento ao final do crawl que varre os arquivos gerados, agrupa por `content_hash`, elege canônico (regra: URL mais curta; ver Decisão em Aberto #4) e preenche `canonical_of` no outro. **Apenas registro de relação** — não deleta nem muda URL canônica no arquivo.
- **Adição de fixtures HTML reais em `fixtures/html/`** (landing, article, listing, empty, listing-com-documentos) + README explicando origem de cada um. (Premissa: usuário vai autorizar download de amostras do portal real em uma sessão específica, usando `curl -sL https://www.tre-pi.jus.br/...` — já permitido em `allow`.)
- Testes de integração com servidor HTTP local servindo os fixtures HTML.

**Escopo (out)**: BFS complementar (descoberta de órfãs) e incremental (fica em M5); mini_summaries (M6); FTS (M7).

**Deliverables concretos**:
- `internal/adapters/htmlextract/*.go` + testes
- `internal/classify/classifier.go` + testes
- `internal/app/crawl.go`
- `cmd/site-research/cmd_crawl.go` com `--dry-run`
- `fixtures/html/*` com README

**Dependências**: M2 (canonicalização), M3 (fetcher + store).

**Critério de pronto**:
- Crawl end-to-end sobre fixtures HTML gera árvore `./data/...` com `_index.json` validando o schema v2 (validador JSON schema simples no teste).
- Hierarquia de `children` coerente com navegação do fixture (CA-2).
- Classificação do `page_type` correta em ≥ 80% da amostra curada de 20 fixtures (CA-5).
- Deduplicação por `content_hash` detecta as duplicatas esperadas em fixtures montados propositalmente.
- `go test ./...` verde.

**Complexidade**: alta. **Maior milestone do plano**. Pode ser dividida em duas se preferir ritmo mais conservador (M4a extração + M4b classificação+crawl).

**Riscos**:
- Plone tem layouts divergentes; a extração de `content-core` pode falhar em listagens ou páginas customizadas. Mitigação: fallback em cascata + `extraction_warnings` registrando cada heurística que não bateu.
- Breadcrumb em Plone tem múltiplas variantes (com/sem microdata). Mitigação: três seletores diferentes + fallback para derivar do path URL.
- Remoção de boilerplate pode remover conteúdo útil por engano. Mitigação: curadoria manual dos fixtures + teste comparando `content_length` extraído vs esperado.

---

### M5 — BFS complementar para órfãs + re-crawl incremental

**Objetivo**: adicionar as duas camadas de robustez sobre o crawl: descoberta de páginas órfãs (linkadas, não no sitemap) e re-crawl incremental com detecção fina de mudanças.

**Escopo (in)**:
- **BFS complementar (RF-02)**:
  - Após o crawl baseado em sitemap, cruza `links.internal` + `links.children` de todos os `_index.json` com a lista visitada.
  - URLs linkadas não visitadas e dentro do escopo → fila de órfãs.
  - Crawl das órfãs com `discovered_via: "link"`.
  - Relatório final com `sitemap_total`, `orphans_found`, `final_crawled`.
- **Re-crawl incremental (RF-08)**:
  - Ao iniciar crawl, lê `_index.json` existentes e constrói índice em memória: `url → (etag, last_modified, content_hash, mini_summary_source_hash)`.
  - Envia `If-None-Match` e `If-Modified-Since` quando presentes.
  - Compara `lastmod` do sitemap com registro local → skip condicional quando possível.
  - HTTP 304 → página marcada inalterada, não reprocessa.
  - `content_hash` igual ao anterior → inalterada, mini_summary preservado.
  - Diferença de `content_hash` ou `lastmod` mais recente → regrava e marca `needs_resummarize: true`.
  - URL presente localmente, ausente do sitemap E 404 em re-crawl → marca `stale_since: <data>` sem deletar. Retenção default 30 dias configurável.
  - Relatório: novas / atualizadas / inalteradas / stale / removidas.
- Subcomando `crawl --purge-stale --confirm` (dois flags exigidos) para deletar entradas cujo `stale_since` ultrapassou a retenção (RNF-05).
- Testes com cenários: sitemap+local alinhados (0 órfãs), sitemap+local divergentes, página removida do site, página modificada, página inalterada (304).

**Escopo (out)**: mini_summaries (M6); FTS (M7).

**Deliverables concretos**:
- `internal/app/orphan.go` (descoberta de órfãs)
- `internal/app/incremental.go` (diff e decisões de skip/refresh)
- `internal/app/stale.go` (marcação e purge)
- Extensões em `cmd_crawl.go`
- Testes de integração com cenários específicos.

**Dependências**: M4 (crawl completo funcionando).

**Critério de pronto**:
- Relatório de órfãs produzido com contagens corretas em cenário sintético (CA-3).
- Re-crawl imediatamente após crawl zero-diff reporta >95% inalteradas e 0 mini_summary regerado (CA-6 — parcial; CA-6 completo só em M6 quando `mini_summary` existir).
- Cenário de página removida → registro fica com `stale_since` preenchido; `--purge-stale --confirm` passado após retenção simulada deleta.
- Suite de testes verde.

**Complexidade**: média-alta.

**Riscos**:
- Armadilha clássica de comparar `lastmod` do sitemap com horário local da máquina — usar UTC consistentemente.
- Servidores Plone podem devolver ETag "fraco" (`W/"..."`) — implementar conforme RFC (comparação forte por padrão para If-None-Match).

---

### M6 — `summarize`: providers de LLM + mini_summaries + contabilidade de custo

**Objetivo**: gerar mini_summaries de 1-2 linhas via LLM configurável, integrado à árvore existente, com tolerância a falhas e contabilidade de tokens.

**Escopo (in)**:
- `internal/domain/ports/` já tem `LLMProvider` desde M1. Agora implementamos.
- `internal/adapters/llm/provider.go`: tipos comuns (`GenerateRequest`, `GenerateResponse`, `UsageMetrics`, `ProviderError`).
- Três adapters:
  - `internal/adapters/llm/gemini.go` — `generativelanguage.googleapis.com` (Gemini API REST, `gemini-2.0-flash` default).
  - `internal/adapters/llm/claude.go` — `api.anthropic.com/v1/messages`.
  - `internal/adapters/llm/openai_compat.go` — endpoint genérico compatível com OpenAI (Ollama, LM Studio, vLLM).
- Seleção por config YAML; chave por env var conforme `api_key_env: GEMINI_API_KEY`.
- **Nunca logar chave**. Redação em qualquer log de debug.
- `internal/app/summarize.go`:
  - Prompt otimizado para roteamento (ver Decisão em Aberto #5) com exemplos few-shot.
  - Distingue `landing` (prompt descritivo de papel navegacional) vs `article`/`listing` (prompt descritivo de conteúdo útil) vs `empty` (skipped: "empty_content").
  - Concorrência configurável (default 3) via worker pool com canal.
  - Falhas isoladas: erro em uma página não aborta pipeline; registra no próprio `_index.json` como `mini_summary.skipped: "llm_error:<código>"` e no relatório.
  - Detecção de `needs_resummarize`: regenera se `source_hash` do full_text atual ≠ `mini_summary.source_hash`.
  - Tokens in/out acumulados → imprime custo estimado ao final (heurística por provider/modelo).
- Teste com mock provider (sem chamar API real) cobrindo: sucesso, rate limit, timeout, erro 500, resposta malformada.
- Fixture de 10 `_index.json` gerados em M4 + snapshot esperado das chamadas feitas ao provider (ordem, prompts, páginas ignoradas).

**Escopo (out)**: catálogo consolidado (M7), busca (M7).

**Deliverables concretos**:
- `internal/adapters/llm/*.go`
- `internal/app/summarize.go`
- `cmd/site-research/cmd_summarize.go`
- `internal/app/prompts/` com templates separados para cada tipo de página.

**Dependências**: M4 (pages com full_text e page_type existem).

**Critério de pronto**:
- Mock provider gera mini_summaries para uma árvore de 20 páginas, com 2 falhas injetadas — pipeline completa, 2 ficam `skipped`, relatório reporta 18 sucessos e 2 falhas (CA-7).
- Re-rodar `summarize` imediatamente não regera os 18 que já têm `source_hash` matching (idempotência).
- Teste de contrato do prompt (snapshot): prompts para `landing` / `article` / `listing` contêm os blocos esperados.
- Custo de tokens é reportado ao final.

**Complexidade**: média (o difícil é o prompt; a infra é relativamente direta).

**Riscos**:
- Qualidade dos mini_summaries dependente de prompt — o BRIEF já lista isso. Mitigação: iterar com amostra real após M4, antes de rodar em escala.
- Gemini/Claude podem mudar formato de resposta; isolar schema no adapter.
- Rate limit do provider: pode exigir throttle independente do HTTP crawler.

---

### M7 — Catálogo consolidado (`build-catalog`), SQLite/FTS, `search`, `inspect`, `stats` + documentação

**Objetivo**: fechar o produto: consolidar a árvore em `catalog.json`, construir SQLite/FTS idempotente, entregar os subcomandos restantes e produzir README de usuário.

**Escopo (in)**:
- `internal/adapters/catalog/builder.go`: walk da árvore `./data/**/_index.json` → `catalog.json` no schema do BRIEF, com `stats.by_depth`, `stats.by_page_type`, `stats.total_pages`.
- `internal/adapters/sqlitefts/` usando **modernc.org/sqlite** (puro Go, compatível com `CGO_ENABLED=0`):
  - Cria DB do zero a cada `build-catalog` (drop+create table + delete do arquivo .sqlite).
  - Tabela FTS5 exata do BRIEF: `pages_fts(path UNINDEXED, url UNINDEXED, title, mini_summary, full_text, section UNINDEXED, page_type UNINDEXED)` com `tokenize = "unicode61 remove_diacritics 2"`.
  - Bulk insert em transação única.
- `internal/app/buildcatalog.go`, `internal/app/search.go`, `internal/app/inspect.go`, `internal/app/stats.go`.
- Subcomandos:
  - `build-catalog`: executa o pipeline de consolidação.
  - `search <query> [--limit N]`: MATCH FTS5 por `title OR mini_summary OR full_text`, ranking bm25, imprime top-N com `title`, `mini_summary`, `url`.
  - `inspect <path|url>`: imprime entrada do catálogo em formato legível (título, path, tipo, children, documentos, mini_summary).
  - `stats`: imprime métricas (total, by_depth, by_page_type, páginas sem mini_summary, total de documentos detectados).
- Teste de busca: executa `search "diárias"` após popular FTS com fixture sintético contendo a palavra em 3 páginas → retorno correto.
- **README de usuário completo na raiz** (substituindo/complementando o atual):
  - Instalação (`go install` / `go build`)
  - Configuração (criar `config.yaml`, env vars de API key)
  - Fluxo completo com exemplo: `discover` → `crawl` → `summarize` → `build-catalog` → `search`
  - Estrutura dos dados produzidos (`./data/**/_index.json`, `catalog.json`, `./data/catalog.sqlite`)
  - Troubleshooting comum.
- Coverage gate 70% no domínio e nos adapters críticos (canonical, classify, htmlextract).

**Escopo (out)**: Fase 2 e posteriores.

**Deliverables concretos**:
- `internal/adapters/catalog/builder.go`
- `internal/adapters/sqlitefts/db.go`, `fts.go`
- `internal/app/{buildcatalog,search,inspect,stats}.go`
- `cmd/site-research/cmd_{build_catalog,search,inspect,stats}.go`
- `README.md` (raiz) atualizado
- Checklist de inspeção manual das 20 páginas (CA-10) — template `.md` para o usuário preencher.

**Dependências**: M4 + M5 + M6 (precisa da árvore completa com mini_summaries).

**Critério de pronto**:
- `build-catalog` produz `catalog.json` consistente e `./data/catalog.sqlite` com FTS populado (CA-8).
- `search "diárias"` retorna resultados com título, mini_summary, URL (CA-9).
- `stats` e `inspect <path>` imprimem saídas legíveis.
- Coverage ≥ 70% nos pacotes críticos.
- `go test ./...` verde sem rede nem API keys (CA-11).
- README cobre instalação, config, fluxo e schema (CA-12).
- Manual inspection template permite verificação de CA-10 (fica pendente da execução pelo usuário — fora do escopo de código).

**Complexidade**: média.

**Riscos**:
- `modernc.org/sqlite` suporta FTS5, mas é preciso verificar build tags corretas (não esquecer que `CGO_ENABLED=0` está imposto).
- `tokenize = "unicode61 remove_diacritics 2"` precisa do valor literal correto — teste com consultas com acento e sem.

---

## 2. Estrutura de Pacotes Go Proposta

```
cristal3/
├── BRIEF.md
├── README.md
├── PLANO_CRAWLER_CRISTAL.md
├── go.mod
├── go.sum
├── config.yaml.example
├── .claude/
│   ├── settings.json
│   └── README.md
├── fixtures/
│   ├── README.md
│   ├── _index.json           # v1, read-only
│   ├── sitemap.xml.gz
│   └── html/                 # adicionado em M4
│       ├── README.md
│       ├── landing.html
│       ├── article.html
│       ├── listing.html
│       ├── empty.html
│       └── listing_com_docs.html
├── cmd/
│   └── site-research/
│       ├── main.go
│       ├── cmd_discover.go
│       ├── cmd_crawl.go
│       ├── cmd_summarize.go
│       ├── cmd_build_catalog.go
│       ├── cmd_search.go
│       ├── cmd_inspect.go
│       └── cmd_stats.go
├── internal/
│   ├── domain/                        # puro, sem imports externos
│   │   ├── page.go                    # Page, Content, Metadata, Links, Document, MiniSummary, Dates, URLRef
│   │   ├── sitemap.go                 # SitemapEntry
│   │   ├── catalog.go                 # CatalogEntry, CatalogStats
│   │   ├── report.go                  # CrawlReport, SummarizeReport
│   │   ├── errors.go
│   │   └── ports/
│   │       └── ports.go               # todas as interfaces
│   ├── canonical/                     # URL canonicalization (pure)
│   │   ├── canonicalizer.go
│   │   └── canonicalizer_test.go
│   ├── classify/                      # page_type heuristic (pure)
│   │   ├── classifier.go
│   │   └── classifier_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── logging/
│   │   └── logger.go
│   ├── adapters/
│   │   ├── httpfetch/                 # net/http + rate + retry + robots
│   │   │   ├── client.go
│   │   │   ├── ratelimit.go
│   │   │   ├── retry.go
│   │   │   └── robots.go
│   │   ├── sitemap/
│   │   │   ├── fetcher.go
│   │   │   └── parser.go
│   │   ├── htmlextract/               # goquery-based extraction
│   │   │   ├── extractor.go
│   │   │   ├── breadcrumb.go
│   │   │   ├── content.go
│   │   │   ├── links.go
│   │   │   ├── documents.go
│   │   │   ├── dates.go
│   │   │   └── keywords.go
│   │   ├── fsstore/
│   │   │   └── store.go
│   │   ├── llm/
│   │   │   ├── provider.go
│   │   │   ├── gemini.go
│   │   │   ├── claude.go
│   │   │   └── openai_compat.go
│   │   ├── catalog/
│   │   │   └── builder.go
│   │   └── sqlitefts/
│   │       ├── db.go
│   │       └── fts.go
│   └── app/                            # application services (orquestração)
│       ├── discover.go
│       ├── crawl.go
│       ├── orphan.go
│       ├── incremental.go
│       ├── stale.go
│       ├── summarize.go
│       ├── prompts/
│       │   ├── landing.txt
│       │   ├── article.txt
│       │   └── listing.txt
│       ├── buildcatalog.go
│       ├── search.go
│       ├── inspect.go
│       └── stats.go
└── data/                               # gerado pelo crawler; gitignored
    └── <hierarquia>/_index.json
```

**Justificativas das escolhas não-óbvias**:

- **`internal/` para todo o código**: impede consumo externo via `go get` antes de a API estabilizar; típico em Go para aplicações.
- **`internal/domain/ports/` em subpacote**: evita ciclo de import entre entidades e interfaces; apps e adapters importam `ports`, domain não sabe dos adapters.
- **`canonical/` e `classify/` fora de `domain/`**: são puros, mas são "use-case primitives" — poderiam estar em `domain/`, porém mantê-los em pacotes próprios facilita teste isolado e reutilização. Cabe escolher; proposta é separar.
- **`htmlextract/` em adapters**: depende de goquery; não pode ficar em `domain/`.
- **`app/` (camada application service)**: orquestra adapters via ports; cada subcomando tem seu arquivo. Facilita testar os fluxos com mocks dos ports sem tocar rede/disk.
- **`prompts/` como texto plano**: facilita revisão, diff e iteração sem recompilar; pode ser embeddado com `//go:embed`.
- **`cmd_*.go` separados por subcomando**: um arquivo = um comando, reduz conflito de merge.

---

## 3. Decisões Técnicas (Consolidadas em 2026-04-20)

**Status**: todas as decisões resolvidas na sessão de planejamento de 2026-04-20.

Recomendações das seções 3.1 a 3.7, 3.10, 3.13, 3.14, 3.16, 3.17 e 3.18 **confirmadas** sem alteração. Os quatro pontos ⚠️ (3.8, 3.9, 3.11, 3.12) têm resolução inline abaixo.

As decisões que afetam a SPEC (RF-03 trailing slash, RF-03 canônico, RF-09 provider default, schema do `_index.json`, YAML de exemplo, referências) estão propostas como atualização **BRIEF v2.0 → v2.1** em [`CHANGES-v2.1.md`](./CHANGES-v2.1.md), **aguardando aprovação antes de aplicar em `BRIEF.md`**.

### 3.1 Biblioteca de parsing HTML
- **goquery** (`github.com/PuerkitoBio/goquery`): seletores CSS, API jQuery-like, maduro, amplamente usado em scraping Go.
- `golang.org/x/net/html`: stdlib-adjacent, lower-level, requer mais código manual para breadcrumb/seletores.
- **Recomendação: goquery**. Trade-off: uma dependência a mais (via `golang.org/x/net/html` internamente), em troca de ~40-60% menos código nos extratores.

### 3.2 Driver SQLite
- `modernc.org/sqlite`: puro Go, compatível com `CGO_ENABLED=0` (requerimento do seu `.claude/settings.json`). FTS5 suportado.
- `mattn/go-sqlite3`: CGo, mais rápido, mas **incompatível com CGO_ENABLED=0**.
- **Recomendação: modernc.org/sqlite (obrigatório)**. Trade-off: binário maior (~5MB) e ~20-30% mais lento em workloads intensivos, mas atende os ~600 registros + FTS deste projeto sem stress.

### 3.3 Logging estruturado
- `log/slog` (stdlib, Go 1.21+): JSON handler nativo, níveis, contexto, zero dependência.
- `zerolog` / `zap`: mais rápidos em alto throughput, API menos idiomática.
- **Recomendação: `log/slog`**. Trade-off: perf não é gargalo aqui; stdlib é o padrão preferido quando possível.

### 3.4 Cliente HTTP
- `net/http` stdlib com wrappers próprios para rate/retry/robots.
- `gocolly/colly`: framework de crawling; traz muito além do necessário (scheduler, DB de visitados etc.).
- **Recomendação: `net/http` + wrappers próprios**. Trade-off: escrevemos ~300 linhas extras, mas mantemos controle fino e testes simples.

### 3.5 Formato de config
- **YAML** (BRIEF mandata — RNF-03). Biblioteca: `gopkg.in/yaml.v3`.
- Sem escolha em aberto aqui — já está definido.

### 3.6 Framework CLI
- **`spf13/cobra`**: subcomandos idiomáticos, help automático, amplamente usado em Go.
- stdlib `flag` com despacho manual: mais leve, sem dependência, mas ergonomia pior para 7 subcomandos.
- **Recomendação: cobra**. Trade-off: +3 deps transitivas, em troca de UX e manutenção melhores.

### 3.7 Biblioteca robots.txt
- `github.com/temoto/robotstxt`: simples, bem-testado, zero deps além de stdlib.
- `github.com/jimsmart/grobotstxt`: alternativa razoável.
- **Recomendação: temoto/robotstxt**.

### 3.8 Política de trailing slash na canonicalização  ✅ **DECIDIDO**

**Decisão**: opção **(a)** — sem trailing slash, exceto na raiz do domínio (`/`).

Regra de canonicalização (ordem de aplicação, a ser codificada em M2):
1. Remove fragmento (`#...`).
2. Remove query params de tracking: `utm_*`, `gclid`, `fbclid`, `_ga`, `portal_form_id`.
3. Se path termina com `/` **E** path ≠ `/`, remove o trailing slash.

Casos de teste mandatórios:
- `/a/b/` → `/a/b`
- `/a/b`  → `/a/b`
- `/`     → `/`
- `/a/b?utm_source=x#frag` → `/a/b`

Justificativa: nenhuma das 573 URLs do sitemap usa trailing slash; normalizar para "sem trailing" garante comparação exata com a fonte de verdade.

**Reflete-se em BRIEF v2.1** → ver `CHANGES-v2.1.md` § 2.

### 3.9 Eleição de canônico para padrão `segmento/segmento`  ✅ **DECIDIDO**

**Decisão**: opção **(a) — shorter wins**. Ambas as URLs são crawleadas e preservadas.

Procedimento:
1. Crawlear todas as URLs do sitemap normalmente (M4).
2. Ao gerar `content_hash`, detectar colisões entre pares `/a/b` vs `/a/b/b`.
3. A URL mais curta é marcada como canônica; a mais longa recebe `metadata.canonical_of = "<url-canônica>"`.
4. `build-catalog` (M7) prefere a canônica em listagens agregadas (`child_count`, hierarquia de `section`).

**Reflete-se em BRIEF v2.1** → ver `CHANGES-v2.1.md` §§ 3 e 5 (schema ganha `canonical_of` no metadata).

### 3.10 Estratégia de retry
Recomendação:
- Backoff exponencial com **full jitter** (estilo AWS), base 500ms, factor 2, ceiling 10s.
- Max 3 tentativas (RF-07).
- Retry apenas em: erros de rede (DNS, timeout, conn reset), HTTP 429, HTTP 5xx.
- Não retry em: 4xx (exceto 429), redirects problemáticos, robots.txt disallow.
- Jitter previne "thundering herd" se múltiplas URLs falharem simultaneamente.

**Aceita?** Se preferir equal-jitter ou decorrelated-jitter, ajustar.

### 3.11 Provider default de LLM  ✅ **DECIDIDO (muda a SPEC)**

**Decisão**: provider default passa de Gemini para **Anthropic**, modelo `claude-haiku-4-5`.

Configuração:
- Provider: `anthropic`
- Modelo: `claude-haiku-4-5` (alias; o snapshot concreto é resolvido pelo adapter)
- Endpoint: `https://api.anthropic.com` (oficial)
- Env var da chave: `ANTHROPIC_API_KEY`
- Concorrência default: 3 requisições simultâneas
- Timeout por requisição: 60s
- Retry: reaproveita a política do adapter HTTP (backoff exponencial + jitter total, base 500ms, max 10s, até 3 tentativas, em 429/5xx/timeouts)

Providers alternativos mantidos via interface: Gemini, OpenAI-compatible.

Prompt a ser calibrado iterativamente em M6 com amostra real (CA-10).

**Impacto na SPEC**: altera RF-09, schema example (`mini_summary.model`) e referências. Ver `CHANGES-v2.1.md` §§ 4, 6, 7, 9.

### 3.12 Módulo Go path  ✅ **DECIDIDO**

**Decisão**: `github.com/bergmaia/site-research`.

Migração futura para repositório institucional do TRE-PI será tratada em commit separado, sem impacto na arquitetura interna (imports via `internal/`).

### 3.13 Progress reporting durante operações longas
- Log JSON periódico a cada N páginas (ex: "processed 50/573").
- Barra TTY (`schollz/progressbar`) se stdout é TTY, logs se não.
- **Recomendação: logs periódicos + flag `--progress-bar` opcional**. Logs são JSON-compatíveis e não poluem output script-friendly.

### 3.14 Quando rodar a deduplicação por `content_hash`
- **Durante `crawl`** (final, em passe pós-fetch): garante que `_index.json` já sai com `canonical_of` preenchido. Recomendado.
- Durante `build-catalog`: mais simples, mas obriga reabrir todos os arquivos mais de uma vez.
- **Recomendação: durante `crawl`**. Custo extra no fim do crawl é baixo (uma varredura + agrupamento em memória).

### 3.15 Prompt exato para mini_summary
Rascunho proposto (pode iterar em M6):

> "Você está catalogando páginas do portal de transparência do TRE-PI. Gere **UMA** descrição objetiva de 1-2 linhas (máximo 180 caracteres) do conteúdo abaixo, voltada para um usuário que quer decidir se essa página responde à pergunta dele. **Não comece com 'Esta página...'**. Descreva o CONTEÚDO, não o fato de ser uma página. Se for uma página de índice de subseções, diga explicitamente que é um índice e quais subtemas lista."

Exemplo few-shot (landing) + exemplo few-shot (article) + input real.

**Precisa iteração manual com 5-10 páginas reais antes do rollout**. Essa calibração fica prevista em M6.

### 3.16 Retenção default de páginas stale
BRIEF diz default 30 dias — tomo como decidido.

### 3.17 Validação de schema do `_index.json`
Propor ou não: validador JSON Schema gerado a partir do BRIEF, rodado em teste? Aumenta confiança contra drift de schema. **Recomendação: sim, um schema JSON simples + `github.com/santhosh-tekuri/jsonschema` em teste**. Mas é opcional — não bloqueia.

### 3.18 Estrutura do sitemap
O sitemap real carrega `<loc>` + `<lastmod>` (confirmado pelas amostras). Não há `<changefreq>` nem `<priority>` — ignoraremos. Sem decisão em aberto.

---

## 4. Mapeamento Milestones × Critérios de Aceite

| Critério | Milestone |
|---|---|
| CA-1 (discover retorna 400-800 URLs) | M2 |
| CA-2 (crawl produz árvore hierárquica) | M4 |
| CA-3 (BFS detecta órfãs com relatório) | M5 |
| CA-4 (canonicalização trata todos os casos) | M2 + M4 (dedup content_hash) |
| CA-5 (page_type ≥80% correto) | M4 |
| CA-6 (re-crawl >95% inalteradas) | M5 + M6 (mini_summary não regera) |
| CA-7 (summarize gera mini_summaries, falhas isoladas, custo) | M6 |
| CA-8 (build-catalog produz catalog.json + SQLite/FTS) | M7 |
| CA-9 (search "diárias") | M7 |
| CA-10 (inspeção manual de 20 mini_summaries) | execução humana pós-M6 |
| CA-11 (go test ./... passa) | transversal, gate de M7 |
| CA-12 (README documenta tudo) | M7 |

---

## 5. Riscos Transversais

1. **Drift de HTML Plone**: se o portal mudar estrutura, extratores quebram. Mitigação: `extraction_warnings` + versionamento (`crawler_version`, `schema_version`) + fixtures datados.
2. **CGo desabilitado**: já tratado via modernc.org/sqlite. **Nenhuma lib em cadeia pode exigir CGo**. Checar em M1 com `go build -v ./...` e vigilância em M7.
3. **Rate limit x re-crawl**: 573 URLs a 1 req/s = ~10 minutos por crawl full. Aceitável para a Fase 1, mas re-crawls frequentes no CI são inviáveis. Testes devem sempre usar HTTP mock local.
4. **API keys em testes**: proibidas por CA-11. Todos os testes de LLM usam mock provider.
5. **`depth` no schema v1 está incorreto (262)**: o crawler novo computa do zero a partir do path URL — não reutilizar esse valor.
6. **`copy_of` count**: BRIEF diz 10, amostra real mostra 14. Pode ser definição diferente (ex: case-sensitive, `copy_of_` vs `copyN_of_`). Flag para esclarecer em M2 antes de fechar teste.

---

## 6. Próximos Passos

**Estado atual** (2026-04-20):
- ✅ Todas as decisões técnicas resolvidas (seção 3).
- ✅ Proteções adicionais em RF-07 (jitter, `Retry-After`, rate limiter compartilhado, circuit breaker, detecção de resposta suspeita) integradas ao diff v2.1 como §§ 10–13, com três novos critérios de aceite (CA-13, CA-14, CA-15).
- ⏳ Diff de atualização da SPEC (v2.0 → v2.1) proposto em [`CHANGES-v2.1.md`](./CHANGES-v2.1.md); aguardando aprovação.
- ✅ Itens deferidos para fases posteriores documentados em § 6.1 abaixo.

**Fluxo aprovado**:

1. ✅ Decisões técnicas tomadas.
2. ⏳ Proposta de diff v2.0 → v2.1 (em `CHANGES-v2.1.md`).
3. ⏳ Aguardar aprovação do diff.
4. ⏳ Aplicar diff em `BRIEF.md`.
5. ⏳ Iniciar M1, **reportando a cada task concluída** (setup do módulo → estrutura de pacotes → domain types → primeiros testes) antes de prosseguir.
6. ⏳ **Não avançar para M2 sem aprovação explícita.**

### 6.1 Itens deferidos para fases posteriores (não implementar na M1)

Documentados aqui para não se perderem; não entram no escopo da Fase 1 de código.

- **Janela de crawl por horário (`allowed_hours`)**: permitir configurar janelas de tempo em que o crawler pode rodar (ex.: `02:00-06:00 America/Fortaleza`), útil para evitar horário de pico do portal. Vai para config YAML numa seção dedicada quando implementado.
- **Cache HTTP opcional para desenvolvimento (`--use-cache`)**: armazenar respostas HTTP em `./data/.http-cache/` para acelerar iteração local sem bater no servidor; invalidado manualmente ou por TTL. Não faz parte do pipeline de produção.
- **Detecção de ciclos no BFS complementar**: já temos um set de URLs visitadas na orquestração do crawl (M4/M5); garantir que o BFS usa o mesmo set para não re-visitar. Implementação trivial quando o BFS for escrito (M5), mas vale registrar como requisito explícito para não esquecer.

---

**Próxima ação aguardando aprovação**: aplicação do diff v2.1 em `BRIEF.md` (atualmente em `CHANGES-v2.1.md`, com as proteções RF-07 já integradas).
