# BRIEF — Site Research (Fase 1: Crawler + Catálogo)

**Versão:** 2.1
**Data:** 2026-04-20
**Autor da especificação:** Rosemberg Maia Gomes (Berg)

**Histórico de revisões:**
- **v2.1 (2026-04-20)**: RF-03 fixa política de trailing slash e regra de canônico; **RF-07 ampliada** com jitter no rate limit, honrar `Retry-After`, rate limiter compartilhado, circuit breaker e detecção de resposta suspeita; RF-09 default LLM passa a Anthropic `claude-haiku-4-5`; schema ganha `canonical_of` e `stale_since`; adiciona YAML de exemplo em RNF-03 e referência à API Anthropic; **critérios de aceite passam de 12 para 15** (adicionados CA-13, CA-14, CA-15).
- v2.0 (2026-04-20): primeira redação formal da Fase 1.

## Contexto

Este projeto implementa um sistema de pesquisa site-specific que, dado um site institucional (alvo inicial: portal TRE-PI), permite a usuários descobrir rapidamente páginas relevantes para uma consulta em linguagem natural e receber resumo curto + link direto para o conteúdo oficial.

A arquitetura final terá quatro fases de desenvolvimento:

1. **Crawler + Catálogo** (esta spec)
2. Engine de roteamento (loop programático multi-estágio sobre o catálogo)
3. Servidor MCP como fachada de consumo
4. Consulta estruturada de datasets (CSV/XLSX anexados)

**Esta spec cobre apenas a Fase 1.** O produto final é um **ativo de dados**: uma árvore hierárquica de páginas crawleadas, enriquecida com mini_summaries gerados por LLM, consultável por busca textual. Esse ativo servirá de insumo para as fases seguintes.

A premissa arquitetural do projeto é: **um catálogo hierárquico com mini_summaries bem-feitos é suficiente para roteamento de pesquisa sem RAG, sem embeddings e sem vector store.** Esta fase valida parte dessa premissa produzindo o catálogo e permitindo inspeção manual de sua qualidade.

## Escopo do Crawl

- **Escopo único e inequívoco**: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas` e todas as páginas sob esse prefixo.
- **Volume esperado**: aproximadamente 573 URLs (valor real obtido do sitemap do portal em 2026-04-20). Após canonicalização, o volume final deve ficar entre 400 e 600 páginas.
- **Profundidade máxima observada**: 8 segmentos de path. A maioria (~80%) das páginas está entre 3 e 5 segmentos de profundidade.
- **CMS alvo**: Plone. Padrões específicos do Plone devem ser tratados pelo crawler (detalhado em RF-02 e RF-03).

### Fora do escopo

- Demais seções do portal (`/institucional`, `/legislacao`, `/eleicoes`, `/servicos-eleitorais`, `/servicos-judiciais`, `/partidos`, `/jurisprudencia`, `/comunicacao`)
- Subdomínios do TRE-PI (`jurisprudencia.tre-pi.jus.br`, `sei.tre-pi.jus.br`, `servicos.tre-pi.jus.br`, `eadeje.tre-pi.jus.br`, `portalservidor3.tre-pi.jus.br`, `revista.tre-pi.jus.br`, entre outros) — tratados como links externos
- Download ou processamento de conteúdo de anexos (PDFs, CSVs, XLSX) — apenas URLs serão registradas
- Qualquer lógica de pesquisa, roteamento ou LLM em runtime de usuário
- Exposição via MCP
- Extração de schemas de datasets estruturados
- Renderização de páginas dinâmicas (SPA / JavaScript)
- Autenticação ou conteúdo protegido

## Objetivo da Fase 1

Produzir, de forma repetível e incremental, um catálogo hierárquico navegável do subsite alvo, contendo para cada página:

- Metadados de identificação (URL canônica, título, breadcrumb, hierarquia)
- Conteúdo textual extraído (summary bruto e full_text limpo)
- Classificação de tipo de página e sinais de qualidade de conteúdo
- Datas de publicação e atualização do conteúdo (quando disponíveis)
- Mini_summary gerado por LLM (1-2 linhas, otimizado para roteamento)
- Relações hierárquicas (parent/children explícitos)
- Listagem de anexos detectados (URLs e metadados leves, sem download)

O sistema deve suportar re-crawls incrementais sem regerar mini_summaries de páginas inalteradas.

## Requisitos Funcionais

### RF-01 — Descoberta de URLs via sitemap

O sistema deve, como estratégia primária, descobrir URLs a partir do sitemap global do portal.

- Baixar `https://www.tre-pi.jus.br/sitemap.xml.gz` (comprimido) ou `sitemap.xml` (fallback).
- Parsear o XML conforme especificação sitemaps.org (urlset).
- Filtrar URLs pelo prefixo do escopo configurado.
- Extrair `lastmod` de cada URL para uso como heurística complementar a ETag no re-crawl incremental.
- Se o sitemap não estiver disponível ou estiver vazio para o prefixo, o crawler deve abortar com erro explícito (não é esperado — o sitemap do TRE-PI foi verificado como funcional em 2026-04-20).
- O sitemap é a **fonte primária de verdade** sobre o conjunto de URLs a crawlear.

### RF-02 — Crawl com validação de cobertura por BFS complementar

Complementarmente ao sitemap, o crawler deve detectar páginas acessíveis por links internos que não apareçam no sitemap.

- Durante a extração de conteúdo de cada página, coletar todos os links internos que caem dentro do escopo.
- Ao final do crawl baseado em sitemap, comparar a lista de URLs visitadas com a lista de URLs linkadas e detectar "órfãs" (linkadas mas não no sitemap).
- Crawlear as órfãs dentro do escopo, marcando-as no metadata como `discovered_via: "link"` (versus `discovered_via: "sitemap"`).
- Produzir relatório ao final indicando: total do sitemap, total de órfãs encontradas, total final crawleado.

### RF-03 — Canonicalização de URLs

Antes de crawlear ou registrar uma URL, aplicar regras de canonicalização determinísticas:

- Remover fragmentos (`#...`) sempre.
- Remover query params conhecidos de tracking: `utm_*`, `gclid`, `fbclid`, `_ga`, `portal_form_id`.
- **Sem trailing slash**, exceto quando o path for a raiz do domínio (`/`). Regra aplicada a todas as URLs antes do filtro de escopo e antes do registro em disco. Exemplos canônicos:
  - `/a/b/` → `/a/b`
  - `/a/b`  → `/a/b`
  - `/`     → `/`
  - `/a/b?utm_source=x#frag` → `/a/b`
- **Detectar e tratar o padrão Plone "segmento/segmento" duplicado no final**: URLs como `/a/b/b` são equivalentes a `/a/b` (o segundo é a página default da pasta). Ambas são preservadas e crawleadas; a deduplicação é registrada como relação, não como exclusão. Procedimento:
  1. Crawlear todas as URLs do sitemap normalmente.
  2. Após extração do conteúdo, comparar `content_hash` entre pares `/a/b` e `/a/b/b`.
  3. Ao detectar colisão, marcar a URL **mais curta** como canônica e preencher o campo `metadata.canonical_of` da mais longa com a URL canônica.
  4. Ambas permanecem no catálogo; `build-catalog` prefere a canônica em listagens agregadas (`child_count`, hierarquia de `section`).
- **Preservar sufixos numéricos `-N`** (ex: `/justica-em-numeros-2019`, `/justica-em-numeros-2025-1`): são conteúdos distintos (diferentes exercícios/edições), não duplicatas.
- **Páginas com `copy_of` no path**: crawlear normalmente, mas registrar flag `is_plone_copy: true` no metadata para revisão manual posterior.
- Excluir URLs que contenham `@@` (views do Plone) ou `++theme++` (assets de tema), caso apareçam.
- Ao detectar redirecionamento HTTP, usar a URL final como canônica e registrar a original em `redirected_from`.

### RF-04 — Extração de conteúdo HTML

Para cada página HTML crawleada, extrair:

- `title`: título principal da página (tag `<title>` ou `<h1>` principal)
- `description`: meta description
- `breadcrumb`: trilha de navegação como lista de `{title, url}`
- `path_titles`: array só com os títulos do breadcrumb (ex: `["Transparência", "Contabilidade", "Balancetes"]`)
- `section`: primeiro segmento do breadcrumb após a raiz do escopo (ex: "Contabilidade")
- `lang`: língua detectada (default "pt-BR")
- `content.summary`: primeiros parágrafos ou lead text (até ~500 chars)
- `content.full_text`: corpo textual limpo, sem menu, footer, scripts ou boilerplate
- `content.full_text_hash`: SHA-256 do `full_text`
- `content.content_hash`: SHA-256 de `title + description + full_text` (detecção de duplicatas via canonicalização)
- `content.content_length`: comprimento em caracteres de `full_text`
- `content.keywords_extracted`: 5-10 termos-chave extraídos por heurística simples (TF sobre o próprio texto, filtrando stopwords em pt-BR) ou das tags/categorias Plone
- `links.children`: páginas filhas diretas na hierarquia do site (dentro do escopo)
- `links.internal`: outros links internos dentro do escopo (não hierárquicos)
- `links.external`: todos os demais links — incluindo páginas de outras seções do TRE-PI fora do escopo, subdomínios do TRE-PI e domínios externos
- `documents`: lista de `{title, url, type, size_bytes, detected_from, context_text}` para anexos detectados (PDFs, CSVs, XLSX, DOCX, ODS). **Apenas metadados; nenhum download realizado nesta fase.** `context_text` deve conter o texto HTML ao redor do link (janela de ~200 chars) para uso futuro na Fase 4.
- `tags`: tags/categorias extraídas da página quando disponíveis

### RF-05 — Classificação de tipo de página

Cada página deve ser classificada em um dos tipos, via heurística determinística executada no crawler:

- `landing`: página de seção com predominância de menu/navegação, pouco conteúdo próprio (ex: `content_length < 500` e ratio links/texto > threshold)
- `article`: página com conteúdo substantivo próprio
- `listing`: página cujo conteúdo principal é uma lista de documentos ou links (ex: lista de portarias, relatórios)
- `redirect`: página que redireciona para outra
- `empty`: página sem conteúdo extraível útil

Derivar também a flag `has_substantive_content: bool` (verdadeiro para `article` e `listing`).

### RF-06 — Extração de datas do conteúdo

Extrair datas do conteúdo (não apenas datas HTTP) para uso em consultas temporais futuras:

- `dates.content_date`: data de publicação extraída do próprio conteúdo (meta tags Plone `DC.date`, `DC.date.created`, `article:published_time`, ou padrão textual da página).
- `dates.page_updated_at`: data de última atualização do conteúdo, extraída de `DC.date.modified` ou `article:modified_time`. Pode coincidir com o `lastmod` do sitemap; preferir o do conteúdo quando disponível.
- Se nenhuma data for extraível do conteúdo, registrar `null` (não fabricar).

### RF-07 — Respeito a políticas de crawl

- Respeitar `robots.txt` do domínio.
- Rate limiting configurável (default: 1 requisição/segundo), aplicado via **token bucket global à execução do crawler** — todos os workers, atuais ou adicionados no futuro, compartilham o mesmo limiter. Sobre o intervalo base, aplicar **jitter aleatório de ±200ms** (configurável via `crawler.jitter_ms`) para que os intervalos reais entre requisições não sejam mecânicos. O jitter também se aplica dentro de retries para evitar retomadas sincronizadas.
- User-Agent identificável e configurável (default: `"TRE-PI-Research-Crawler/0.1 (+contact: cotdi@tre-pi.jus.br)"`).
- Timeout por requisição configurável (default: 30s).
- Retry com backoff exponencial em erros transientes (até 3 tentativas; falhas finais são registradas no relatório, não abortam o crawl).
- **Honrar o cabeçalho `Retry-After`** em respostas HTTP 429 e 503 (configurável via `crawler.honor_retry_after`, default `true`):
  - Quando presente, usar o valor indicado (segundos ou data HTTP RFC 7231) em vez do backoff exponencial padrão.
  - Se o valor for **maior que 60s**, logar em nível `warn` e **pausar o crawl inteiro** (não apenas a URL em questão) até o momento indicado.
  - Quando ausente, aplicar backoff exponencial normal da política de retry.
- **Circuit breaker** sobre falhas consecutivas (erros de rede, 5xx não resolvidos após retry, respostas suspeitas — ver bullet seguinte):
  - Após `N` falhas consecutivas (default `N=5`, configurável via `crawler.circuit_breaker.max_consecutive_failures`), pausar o crawl por `M` minutos (default `M=10`).
  - Ao retomar, se as próximas `K` requisições falharem (default `K=3`, configurável via `abort_threshold`), **abortar** o crawl e emitir relatório completo: URLs já processadas, URLs pendentes, último erro, histograma de códigos HTTP observados.
  - Qualquer sucesso reseta o contador de falhas consecutivas.
  - O estado do breaker é logado em cada transição (fechado → aberto → meio-aberto → fechado/abort).
- **Detecção de resposta suspeita** aplicada após o download do HTML e antes da extração. A resposta é considerada suspeita se qualquer uma das heurísticas abaixo dispara:
  - `<title>` contém um dos padrões configurados em `crawler.suspicious_response.block_title_patterns` (default: "Access Denied", "Forbidden", "Captcha", "Cloudflare", "Just a moment"; match case-insensitive).
  - `content-length` do body é inferior a `crawler.suspicious_response.min_body_bytes` (default 500) **E** a URL tem registro local anterior com conteúdo de tamanho normal.
  - Presença de headers típicos de WAF em resposta que não bate com fluxo normal do Plone — ex.: `cf-ray`, `x-sucuri-id` sem a resposta apresentar um `X-Generator: Plone` ou marcadores equivalentes.

  Ao detectar suspeita:
  1. **Não gravar `_index.json`** daquela URL (preserva o último registro válido se houver).
  2. Logar em nível `warn` com a URL, motivo da suspeita e amostra de até 200 bytes do body (cookies/headers sensíveis devem ser redactados).
  3. Contabilizar a ocorrência como **falha** para o contador do circuit breaker.
  4. A URL permanece elegível para re-crawl no próximo ciclo; **não** é marcada como `stale`.

### RF-08 — Re-crawl incremental

- Ao re-crawlear, enviar `If-None-Match` (ETag) e `If-Modified-Since` quando disponíveis.
- Usar também o `lastmod` do sitemap como sinal: páginas com `lastmod` igual ou anterior ao registro local são candidatas a skip condicional.
- Páginas retornando HTTP 304 não são reprocessadas nem têm mini_summary regerado.
- Páginas com conteúdo alterado (detectado por ETag diferente, `content_hash` diferente ou `lastmod` posterior) são marcadas para regeneração de mini_summary.
- Páginas removidas do site (presentes no catálogo anterior, ausentes no sitemap atual E retornando 404 em re-crawl) são marcadas como `stale` mas **não deletadas** imediatamente (retenção configurável em dias; default: 30).
- Re-crawl produz relatório com contagem de páginas novas, atualizadas, inalteradas, marcadas stale e removidas.

### RF-09 — Geração de mini_summaries

Comando separado (`summarize`) lê o catálogo bruto e produz mini_summaries para páginas que ainda não têm um ou que foram marcadas para regeneração.

- Provider de LLM selecionado por configuração, com suporte a múltiplos providers via interface. **Default: Anthropic (modelo `claude-haiku-4-5`), endpoint oficial `https://api.anthropic.com`, chave via env var `ANTHROPIC_API_KEY`.** Outros providers suportados: Gemini, OpenAI-compatible endpoint (Ollama, LM Studio, vLLM).
- Configuração de provider via arquivo de config + variáveis de ambiente para chaves de API (nunca no config file).
- Prompt otimizado para produzir 1-2 linhas descritivas focadas no conteúdo útil da página, voltado para roteamento. Prompt deve desencorajar sumários genéricos do tipo "Esta página fala sobre...".
- Páginas classificadas como `empty` não recebem mini_summary (registram flag `skipped: "empty_content"`).
- Páginas classificadas como `landing` recebem mini_summary especial descrevendo o papel navegacional ("Índice das subseções de X").
- Mini_summaries armazenados dentro do próprio `_index.json` da página.
- `source_hash` (hash do `full_text` usado na geração) armazenado junto, permitindo detectar quando regenerar.
- Falhas de LLM (timeout, rate limit, erro) não abortam o pipeline — são registradas e páginas afetadas ficam sem mini_summary (reprocessáveis).
- Processamento em batch com concorrência configurável (default: 3 requisições simultâneas).
- Timeout por requisição LLM configurável (default: 60s).
- Política de retry do adapter LLM reaproveita a do adapter HTTP: backoff exponencial com jitter total (base 500ms, max 10s), até 3 tentativas, aplicável a HTTP 429/5xx e a timeouts/erros de rede.
- Log de custo acumulado (tokens input/output) ao final da execução.

### RF-10 — Catálogo consolidado e FTS

Comando `build-catalog` consolida a árvore de `_index.json` em:

- `catalog.json`: visão plana de todas as páginas com campos compactos (`path`, `url`, `title`, `depth`, `parent`, `section`, `page_type`, `has_substantive_content`, `mini_summary`, `child_count`, `has_docs`, `content_date`).
- Banco SQLite com tabela FTS5 indexando `title`, `mini_summary` e `full_text`.
- Banco SQLite reconstruído do zero a cada `build-catalog` (idempotente).
- Tokenização `unicode61 remove_diacritics 2` para busca em português insensível a acentos.

### RF-11 — CLI

Interface de linha de comando com os seguintes subcomandos:

- `discover`: baixa sitemap, filtra pelo prefixo, imprime lista de URLs no escopo (não crawlea). Útil para validação prévia.
- `crawl`: executa crawl (inicial ou incremental) a partir da lista derivada do sitemap. Suporta flag `--dry-run` para apenas simular sem gravar.
- `summarize`: gera mini_summaries para páginas pendentes.
- `build-catalog`: consolida árvore em `catalog.json` e SQLite/FTS.
- `inspect <path|url>`: imprime representação legível de uma entrada do catálogo.
- `search <query>`: busca textual via FTS, retorna top-N com título, mini_summary e URL.
- `stats`: imprime métricas do catálogo (total de páginas, distribuição por profundidade, distribuição por `page_type`, páginas sem mini_summary, anexos detectados).

Todos os comandos aceitam flag `--config` apontando para arquivo de configuração.

## Requisitos Não-Funcionais

### RNF-01 — Linguagem e arquitetura

- Implementação em Go (1.22+).
- Arquitetura hexagonal: domínio puro (entidades, ports) isolado de adapters (HTTP client, filesystem, LLM clients, SQLite).
- Princípios SOLID aplicados: cada adapter implementa uma interface do domínio; troca de provider LLM é plug-and-play.
- Zero dependência de banco externo (SQLite embedded).

### RNF-02 — Observabilidade

- Log estruturado (JSON) com níveis configuráveis.
- Progress reporting em operações longas (crawl, summarize) via barra de progresso ou logs periódicos.
- Métricas básicas expostas ao final de cada comando (duração, páginas processadas, erros, custo LLM quando aplicável).

### RNF-03 — Configuração

Arquivo de configuração YAML com:

- Seed URL e prefixo permitido do escopo
- URL do sitemap (com default derivado do domínio da seed)
- Rate limit, timeout, retries
- User-Agent
- Caminho de armazenamento
- Configuração de LLM (provider, modelo, endpoint, referência a env var da chave)
- Parâmetros de re-crawl (retenção de stale em dias, forçar regeneração)

Valores sensíveis (API keys) apenas via variáveis de ambiente. O config file pode referenciar env vars por nome.

#### Exemplo de `config.yaml`

```yaml
scope:
  seed_url: "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"
  prefix:   "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"

sitemap:
  url: "https://www.tre-pi.jus.br/sitemap.xml.gz"

crawler:
  user_agent: "TRE-PI-Research-Crawler/0.1 (+contact: cotdi@tre-pi.jus.br)"
  rate_limit_per_second: 1.0
  jitter_ms: 200                        # ±ms aleatórios sobre o intervalo do rate limit
  request_timeout_seconds: 30
  max_retries: 3
  respect_robots_txt: true
  honor_retry_after: true               # 429/503 com Retry-After sobrescrevem backoff

  circuit_breaker:
    max_consecutive_failures: 5         # abre o breaker após N falhas seguidas
    pause_minutes: 10                   # duração da pausa quando abre
    abort_threshold: 3                  # falhas consecutivas pós-retorno que disparam abort

  suspicious_response:
    min_body_bytes: 500                 # body menor que isto + histórico com conteúdo ⇒ suspeita
    block_title_patterns:
      - "Access Denied"
      - "Forbidden"
      - "Captcha"
      - "Cloudflare"
      - "Just a moment"

storage:
  data_dir:     "./data"
  catalog_path: "./data/catalog.json"
  sqlite_path:  "./data/catalog.sqlite"

llm:
  provider:    "anthropic"              # anthropic | gemini | openai_compat
  model:       "claude-haiku-4-5"
  endpoint:    "https://api.anthropic.com"
  api_key_env: "ANTHROPIC_API_KEY"
  concurrency: 3
  request_timeout_seconds: 60

recrawl:
  stale_retention_days: 30
  force_resummarize:    false
```

API keys **nunca** ficam neste arquivo — são referenciadas por nome de variável de ambiente via `api_key_env` e lidas em runtime pelo adapter.

### RNF-04 — Testabilidade

- Domínio testável sem rede, sem filesystem, sem LLM (mocks das interfaces).
- Testes de integração com servidor HTTP local servindo fixtures de páginas Plone reais (amostra representativa do TRE-PI).
- Fixtures de páginas TRE-PI (HTML estático, sitemap) commitadas no repositório.
- Suite de testes executável com `go test ./...` sem dependências externas ou API keys.
- Cobertura mínima de 70% no domínio e nos adapters críticos (canonicalização, classificação, extração).

### RNF-05 — Idempotência e segurança de dados

- Re-rodar qualquer comando não deve corromper estado existente.
- Escritas em arquivos usam write-then-rename para evitar corrupção em caso de crash.
- Operações destrutivas (ex: purge de páginas stale) requerem flag explícita `--confirm`.

### RNF-06 — Conformidade com boas práticas institucionais

- Logs não devem expor conteúdo completo das páginas em níveis INFO/WARN (apenas URLs e métricas).
- O crawler não deve efetuar cache em proxies públicos (usar header `Cache-Control: no-cache` no request).
- Código e configurações de exemplo não devem conter chaves de API reais.

## Estrutura de Dados

### Schema do `_index.json` por página (schema v2)

```json
{
  "$schema": "page-node-v2",
  "schema_version": 2,
  "url": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/contabilidade",
  "canonical_url": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/contabilidade",
  "title": "Contabilidade",
  "description": "Informações contábeis e demonstrativos financeiros do TRE-PI.",
  "section": "Contabilidade",
  "breadcrumb": [
    {"title": "Transparência e Prestação de Contas", "url": "..."},
    {"title": "Contabilidade", "url": "..."}
  ],
  "path_titles": ["Transparência e Prestação de Contas", "Contabilidade"],
  "lang": "pt-BR",
  "page_type": "landing",
  "has_substantive_content": true,
  "content": {
    "summary": "...",
    "full_text": "...",
    "full_text_hash": "sha256:...",
    "content_hash": "sha256:...",
    "content_length": 1535,
    "keywords_extracted": ["contabilidade", "balancete", "demonstrativo", "financeiro"]
  },
  "mini_summary": {
    "text": "Relatórios contábeis, balancetes mensais e demonstrações financeiras do TRE-PI.",
    "generated_at": "2026-04-20T18:00:00Z",
    "model": "claude-haiku-4-5",
    "source_hash": "sha256:...",
    "skipped": null
  },
  "dates": {
    "content_date": "2026-03-15",
    "page_updated_at": "2026-04-01T10:00:00Z"
  },
  "metadata": {
    "depth": 1,
    "extracted_at": "2026-04-20T18:00:00Z",
    "last_modified": "Wed, 01 Apr 2026 10:00:00 GMT",
    "etag": "...",
    "http_status": 200,
    "content_type": "text/html; charset=utf-8",
    "parent_url": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas",
    "redirected_from": null,
    "canonical_of": null,
    "fetch_duration_ms": 245,
    "crawler_version": "0.1.0",
    "discovered_via": "sitemap",
    "is_plone_copy": false,
    "stale_since": null,
    "extraction_warnings": []
  },
  "links": {
    "children": [
      {"title": "Balancetes", "url": "...", "local_path": "balancetes/_index.json"}
    ],
    "internal": [],
    "external": []
  },
  "documents": [
    {
      "title": "Balancete Março 2026",
      "url": "...",
      "type": "pdf",
      "size_bytes": null,
      "detected_from": "link_href",
      "context_text": "..."
    }
  ],
  "tags": []
}
```

### Schema do `catalog.json` consolidado

```json
{
  "generated_at": "2026-04-20T18:00:00Z",
  "root_url": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas",
  "schema_version": 2,
  "stats": {
    "total_pages": 573,
    "by_depth": {"2": 17, "3": 80, "4": 253, "5": 160, "6": 54, "7": 7, "8": 1},
    "by_page_type": {"landing": 120, "article": 380, "listing": 50, "empty": 23}
  },
  "entries": [
    {
      "path": "contabilidade",
      "url": "...",
      "title": "Contabilidade",
      "depth": 1,
      "parent": "",
      "section": "Contabilidade",
      "page_type": "landing",
      "has_substantive_content": true,
      "mini_summary": "...",
      "child_count": 5,
      "has_docs": true,
      "content_date": "2026-03-15"
    }
  ]
}
```

### Schema SQLite/FTS

```sql
CREATE VIRTUAL TABLE pages_fts USING fts5(
  path UNINDEXED,
  url UNINDEXED,
  title,
  mini_summary,
  full_text,
  section UNINDEXED,
  page_type UNINDEXED,
  tokenize = "unicode61 remove_diacritics 2"
);
```

## Critérios de Aceite

A Fase 1 está completa quando todos os critérios abaixo forem atendidos:

1. `discover` executa com sucesso, baixa o sitemap global, filtra pelo prefixo de escopo e imprime entre 400 e 800 URLs.
2. `crawl` executado sobre as URLs descobertas produz árvore de `_index.json` no filesystem, com hierarquia correta espelhando a navegação do site.
3. BFS complementar detecta eventuais URLs órfãs (linkadas mas não no sitemap) e produz relatório final com contagens claras.
4. Canonicalização trata corretamente: URLs com segmento duplicado (`/a/b/b`), sufixos numéricos (`-1`, `-2`), páginas `copy_of`, fragmentos e query params de tracking. Testes específicos cobrem cada caso.
5. `page_type` classifica corretamente ao menos 80% de uma amostra de 20 páginas validada manualmente.
6. Re-crawl executado em seguida, sem mudanças no site, reporta alta taxa de inalteração (>95%) e não regera mini_summaries dessas páginas.
7. `summarize` gera mini_summaries de 1-2 linhas descritivas para toda a árvore, com falhas isoladas não abortando o pipeline, e com custo total reportado ao final.
8. `build-catalog` produz `catalog.json` consistente e banco SQLite com FTS populado.
9. `search "diárias"` retorna resultados relevantes do catálogo com título, mini_summary e URL.
10. Inspeção manual de 20 entradas aleatórias do catálogo confirma que os mini_summaries descrevem fielmente o conteúdo das páginas.
11. Suite `go test ./...` passa sem dependências de rede ou API keys.
12. README documenta instalação, configuração, fluxo completo (discover → crawl → summarize → build-catalog → search) e estrutura dos dados produzidos.
13. Teste unitário do rate limiter verifica que o jitter é **efetivamente aplicado**: a variância dos intervalos entre requisições em uma amostra de 30 chamadas sucessivas é estatisticamente não-nula e compatível com a faixa `±jitter_ms` configurada.
14. Teste de integração simula respostas HTTP 429 e 503 com cabeçalho `Retry-After` em ambas as formas (segundos e data HTTP) e verifica que o crawler aguarda o valor indicado antes da próxima requisição. Para `Retry-After > 60s`, verifica (a) log em `warn` e (b) pausa efetiva do crawl inteiro até o instante alvo.
15. Teste de integração simula sequência de 5xx consecutivos e verifica (a) abertura do circuit breaker após `N` falhas, (b) pausa de duração `M` minutos (tempo simulado), (c) abort com relatório estruturado após `K` falhas consecutivas no retorno. Teste separado verifica que uma resposta suspeita detectada incrementa o contador do breaker e não grava `_index.json`.

## Riscos e Decisões em Aberto

- **Qualidade dos mini_summaries**: dependente do prompt e do modelo. Mitigação: iterar em prompt engineering com amostra de páginas reais antes de rodar no catálogo completo; validar com inspeção manual de amostra.
- **Variação de estrutura HTML em Plone**: páginas podem ter layouts diferentes (especialmente entre `landing`, `article` e `listing`). Mitigação: testar extração em amostra diversa; suportar fallback de extração genérica; registrar `extraction_warnings` quando heurísticas falham.
- **Páginas `copy_of` ambíguas**: podem ser conteúdo legítimo ou lixo residual. Decisão: crawlear tudo, sinalizar com flag, deixar decisão de purge para revisão manual.
- **Padrão `segmento/segmento` duplicado**: 96 URLs (17% do escopo) apresentam esse padrão. Decisão tomada (v2.1): preservar todas as URLs; ao detectar colisão de `content_hash` entre pares `/a/b` vs `/a/b/b`, a URL **mais curta é canônica** e a mais longa recebe `metadata.canonical_of`. Ver RF-03.
- **URLs com sufixo numérico**: 159 URLs apresentam esse padrão. Decisão tomada: preservar como URLs distintas, não canonicalizar entre elas.
- **Evolução do Plone/TSE**: o portal segue padrões do TSE e pode mudar estrutura. Mitigação: `crawler_version` e `schema_version` permitem identificar dados antigos e regerar seletivamente.

## Dados Reais de Referência

Observações empíricas obtidas do sitemap do portal em 2026-04-20, que devem guiar testes e calibração:

- Total de URLs no sitemap global do portal: 2.367
- URLs dentro do escopo de `/transparencia-e-prestacao-de-contas`: 573
- Subseções mais densas (top 5 por contagem de páginas): Recursos Humanos e Remuneração (63), Estratégia (51), Comissões Permanentes e Técnicas (47), Justiça em Números (37), Governança de TI (34)
- Profundidade máxima observada: 8 segmentos de path
- URLs com padrão `segmento/segmento` duplicado: 96 (17% do escopo)
- URLs com sufixo numérico `-N`: 159 (28% do escopo)
- URLs com `copy_of` no path: 10

## Referências

- Portal TRE-PI: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`
- Sitemap do portal: `https://www.tre-pi.jus.br/sitemap.xml.gz`
- Especificação sitemaps.org: `https://www.sitemaps.org/protocol.html`
- Plone CMS (gerenciador do portal): estrutura de breadcrumb, metadados e tags seguem convenções Plone
- API Anthropic Messages (provider LLM default): `https://docs.claude.com/en/api/messages`
- Estrutura existente de `_index.json` do crawler anterior (formato evoluído para schema v2 nesta spec)
