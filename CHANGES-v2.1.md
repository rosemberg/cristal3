# BRIEF — Proposta de atualização v2.0 → v2.1

**Status**: proposto, aguardando aprovação do autor da SPEC (Berg).
**Data da proposta**: 2026-04-20.
**Não aplicar sem aprovação explícita.**

## Motivação

Incorporar decisões tomadas na sessão de planejamento da Fase 1:

- **RF-03**: fixar política de trailing slash e regra de eleição de canônico em duplicatas.
- **RF-07**: ampliar políticas de crawl com cinco proteções adicionais contra bloqueio, rate limiting passivo e "lixo silencioso" (captcha/WAF mascarado) — jitter no rate limit, honrar `Retry-After`, garantia de rate limiter compartilhado, circuit breaker e detecção de resposta suspeita.
- **RF-09**: trocar provider default de LLM de Gemini para Anthropic (modelo `claude-haiku-4-5`).
- **Schema `_index.json`**: tornar explícitos os campos `canonical_of` e `stale_since` no metadata; atualizar modelo exemplificado em `mini_summary`.
- **RNF-03**: adicionar YAML de exemplo concreto, incluindo os novos parâmetros de RF-07.
- **Critérios de aceite**: adicionar **CA-13, CA-14, CA-15** cobrindo jitter, `Retry-After` e circuit breaker.
- **Referências**: adicionar link para a documentação da API Anthropic.

Nenhuma das mudanças altera o escopo, o volume esperado de URLs ou a arquitetura hexagonal. Os 12 critérios de aceite originais são preservados literalmente; **três novos critérios são adicionados**, elevando o total para 15.

---

## Diff proposto

### § 1 — Cabeçalho (linhas 1-4)

**Antes:**
```
# BRIEF — Site Research (Fase 1: Crawler + Catálogo)

**Versão:** 2.0
**Data:** 2026-04-20
**Autor da especificação:** Rosemberg Maia Gomes (Berg)
```

**Depois:**
```
# BRIEF — Site Research (Fase 1: Crawler + Catálogo)

**Versão:** 2.1
**Data:** 2026-04-20
**Autor da especificação:** Rosemberg Maia Gomes (Berg)

**Histórico de revisões:**
- **v2.1 (2026-04-20)**: RF-03 fixa política de trailing slash e regra de canônico; **RF-07 ampliada** com jitter no rate limit, honrar `Retry-After`, rate limiter compartilhado, circuit breaker e detecção de resposta suspeita; RF-09 default LLM passa a Anthropic `claude-haiku-4-5`; schema ganha `canonical_of` e `stale_since`; adiciona YAML de exemplo em RNF-03 e referência à API Anthropic; **critérios de aceite passam de 12 para 15** (adicionados CA-13, CA-14, CA-15).
- v2.0 (2026-04-20): primeira redação formal da Fase 1.
```

---

### § 2 — RF-03 · trailing slash

**Antes (linha 82):**
```
- Preservar trailing slash **consistentemente** (escolher uma política: com ou sem trailing slash, aplicada a todas as URLs).
```

**Depois:**
```
- **Sem trailing slash**, exceto quando o path for a raiz do domínio (`/`). Regra aplicada a todas as URLs antes do filtro de escopo e antes do registro em disco. Exemplos canônicos:
  - `/a/b/` → `/a/b`
  - `/a/b`  → `/a/b`
  - `/`     → `/`
  - `/a/b?utm_source=x#frag` → `/a/b`
```

Justificativa (informativa, não vai para o BRIEF): nenhuma das 573 URLs do sitemap do TRE-PI em 2026-04-20 usa trailing slash em folhas; normalizar para "sem trailing" garante comparação exata com a fonte de verdade.

---

### § 3 — RF-03 · eleição de canônico em duplicatas

**Antes (linha 83):**
```
- **Detectar e tratar o padrão Plone "segmento/segmento" duplicado no final**: URLs como `/a/b/b` são equivalentes a `/a/b/` (o segundo é a página default da pasta). Decisão de canonicalização: preservar a forma como aparece no sitemap, mas detectar duplicatas entre as duas formas via `content_hash` e marcar uma como `canonical_of` a outra.
```

**Depois:**
```
- **Detectar e tratar o padrão Plone "segmento/segmento" duplicado no final**: URLs como `/a/b/b` são equivalentes a `/a/b` (o segundo é a página default da pasta). Ambas são preservadas e crawleadas; a deduplicação é registrada como relação, não como exclusão. Procedimento:
  1. Crawlear todas as URLs do sitemap normalmente.
  2. Após extração do conteúdo, comparar `content_hash` entre pares `/a/b` e `/a/b/b`.
  3. Ao detectar colisão, marcar a URL **mais curta** como canônica e preencher o campo `metadata.canonical_of` da mais longa com a URL canônica.
  4. Ambas permanecem no catálogo; `build-catalog` prefere a canônica em listagens agregadas (`child_count`, hierarquia de `section`).
```

---

### § 4 — RF-09 · provider default e parâmetros

**Antes (linha 152):**
```
- Provider de LLM selecionado por configuração, com suporte a múltiplos providers via interface. Default: Gemini. Outros providers suportados: Claude, OpenAI-compatible endpoint (Ollama, LM Studio, vLLM).
```

**Depois:**
```
- Provider de LLM selecionado por configuração, com suporte a múltiplos providers via interface. **Default: Anthropic (modelo `claude-haiku-4-5`), endpoint oficial `https://api.anthropic.com`, chave via env var `ANTHROPIC_API_KEY`.** Outros providers suportados: Gemini, OpenAI-compatible endpoint (Ollama, LM Studio, vLLM).
```

**Antes (linha 160):**
```
- Processamento em batch com concorrência configurável (default: 3 requisições simultâneas).
- Log de custo acumulado (tokens input/output) ao final da execução.
```

**Depois (insere duas linhas entre as existentes):**
```
- Processamento em batch com concorrência configurável (default: 3 requisições simultâneas).
- Timeout por requisição LLM configurável (default: 60s).
- Política de retry do adapter LLM reaproveita a do adapter HTTP: backoff exponencial com jitter total (base 500ms, max 10s), até 3 tentativas, aplicável a HTTP 429/5xx e a timeouts/erros de rede.
- Log de custo acumulado (tokens input/output) ao final da execução.
```

---

### § 5 — Schema `_index.json` v2 · `metadata.canonical_of` e `metadata.stale_since`

**Antes (linhas 275-289):**
```json
  "metadata": {
    "depth": 1,
    "extracted_at": "2026-04-20T18:00:00Z",
    "last_modified": "Wed, 01 Apr 2026 10:00:00 GMT",
    "etag": "...",
    "http_status": 200,
    "content_type": "text/html; charset=utf-8",
    "parent_url": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas",
    "redirected_from": null,
    "fetch_duration_ms": 245,
    "crawler_version": "0.1.0",
    "discovered_via": "sitemap",
    "is_plone_copy": false,
    "extraction_warnings": []
  },
```

**Depois:**
```json
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
```

**Notas semânticas**:
- `canonical_of`: `null` quando a página É a canônica; preenchido com a URL canônica quando a página é duplicata detectada pelo procedimento de RF-03.
- `stale_since`: `null` quando a página está ativa; preenchido com timestamp ISO-8601 quando RF-08 detecta ausência no sitemap + 404 em re-crawl. Retenção default 30 dias controla purge com `--confirm`.

---

### § 6 — Schema `_index.json` v2 · modelo exemplificado no `mini_summary`

**Antes (linhas 264-270):**
```json
  "mini_summary": {
    "text": "Relatórios contábeis, balancetes mensais e demonstrações financeiras do TRE-PI.",
    "generated_at": "2026-04-20T18:00:00Z",
    "model": "gemini-2.0-flash",
    "source_hash": "sha256:...",
    "skipped": null
  },
```

**Depois:**
```json
  "mini_summary": {
    "text": "Relatórios contábeis, balancetes mensais e demonstrações financeiras do TRE-PI.",
    "generated_at": "2026-04-20T18:00:00Z",
    "model": "claude-haiku-4-5",
    "source_hash": "sha256:...",
    "skipped": null
  },
```

---

### § 7 — RNF-03 · adicionar YAML de exemplo concreto

**Após** o bloco atual de RNF-03 (após a linha "Valores sensíveis (API keys) apenas via variáveis de ambiente. O config file pode referenciar env vars por nome."), **adicionar nova subseção**:

```
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
  provider:    "anthropic"            # anthropic | gemini | openai_compat
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
```

---

### § 8 — Seção "Riscos e Decisões em Aberto"

**Antes (linha 379):**
```
- **Padrão `segmento/segmento` duplicado**: 96 URLs (17% do escopo) apresentam esse padrão. Decisão tomada: preservar URLs como aparecem no sitemap, detectar duplicatas via `content_hash` e registrar relação `canonical_of`.
```

**Depois:**
```
- **Padrão `segmento/segmento` duplicado**: 96 URLs (17% do escopo) apresentam esse padrão. Decisão tomada (v2.1): preservar todas as URLs; ao detectar colisão de `content_hash` entre pares `/a/b` vs `/a/b/b`, a URL **mais curta é canônica** e a mais longa recebe `metadata.canonical_of`. Ver RF-03.
```

---

### § 9 — Referências

**Antes (linhas 395-401):**
```
## Referências

- Portal TRE-PI: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`
- Sitemap do portal: `https://www.tre-pi.jus.br/sitemap.xml.gz`
- Especificação sitemaps.org: `https://www.sitemaps.org/protocol.html`
- Plone CMS (gerenciador do portal): estrutura de breadcrumb, metadados e tags seguem convenções Plone
- Estrutura existente de `_index.json` do crawler anterior (formato evoluído para schema v2 nesta spec)
```

**Depois:**
```
## Referências

- Portal TRE-PI: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`
- Sitemap do portal: `https://www.tre-pi.jus.br/sitemap.xml.gz`
- Especificação sitemaps.org: `https://www.sitemaps.org/protocol.html`
- Plone CMS (gerenciador do portal): estrutura de breadcrumb, metadados e tags seguem convenções Plone
- API Anthropic Messages (provider LLM default): `https://docs.claude.com/en/api/messages`
- Estrutura existente de `_index.json` do crawler anterior (formato evoluído para schema v2 nesta spec)
```

---

### § 10 — RF-07 · jitter no rate limiting + rate limiter compartilhado

**Antes (RF-07, bullet atual de rate limit):**
```
- Rate limiting configurável (default: 1 requisição/segundo).
```

**Depois:**
```
- Rate limiting configurável (default: 1 requisição/segundo), aplicado via **token bucket global à execução do crawler** — todos os workers, atuais ou adicionados no futuro, compartilham o mesmo limiter. Sobre o intervalo base, aplicar **jitter aleatório de ±200ms** (configurável via `crawler.jitter_ms`) para que os intervalos reais entre requisições não sejam mecânicos. O jitter também se aplica dentro de retries para evitar retomadas sincronizadas.
```

---

### § 11 — RF-07 · honrar `Retry-After`

**Adicionar nova bullet em RF-07, após a bullet de retry:**
```
- **Honrar o cabeçalho `Retry-After`** em respostas HTTP 429 e 503 (configurável via `crawler.honor_retry_after`, default `true`):
  - Quando presente, usar o valor indicado (segundos ou data HTTP RFC 7231) em vez do backoff exponencial padrão.
  - Se o valor for **maior que 60s**, logar em nível `warn` e **pausar o crawl inteiro** (não apenas a URL em questão) até o momento indicado.
  - Quando ausente, aplicar backoff exponencial normal da política de retry.
```

---

### § 12 — RF-07 · circuit breaker

**Adicionar nova bullet em RF-07:**
```
- **Circuit breaker** sobre falhas consecutivas (erros de rede, 5xx não resolvidos após retry, respostas suspeitas — ver bullet seguinte):
  - Após `N` falhas consecutivas (default `N=5`, configurável via `crawler.circuit_breaker.max_consecutive_failures`), pausar o crawl por `M` minutos (default `M=10`).
  - Ao retomar, se as próximas `K` requisições falharem (default `K=3`, configurável via `abort_threshold`), **abortar** o crawl e emitir relatório completo: URLs já processadas, URLs pendentes, último erro, histograma de códigos HTTP observados.
  - Qualquer sucesso reseta o contador de falhas consecutivas.
  - O estado do breaker é logado em cada transição (fechado → aberto → meio-aberto → fechado/abort).
```

---

### § 13 — RF-07 · detecção de resposta suspeita

**Adicionar nova bullet em RF-07:**
```
- **Detecção de resposta suspeita** aplicada após o download do HTML e antes da extração. A resposta é considerada suspeita se qualquer uma das heurísticas abaixo dispara:
  - `<title>` contém um dos padrões configurados em `crawler.suspicious_response.block_title_patterns` (default: "Access Denied", "Forbidden", "Captcha", "Cloudflare", "Just a moment"; match case-insensitive).
  - `content-length` do body é inferior a `crawler.suspicious_response.min_body_bytes` (default 500) **E** a URL tem registro local anterior com conteúdo de tamanho normal.
  - Presença de headers típicos de WAF em resposta que não bate com fluxo normal do Plone — ex.: `cf-ray`, `x-sucuri-id` sem a resposta apresentar um `X-Generator: Plone` ou marcadores equivalentes.

  Ao detectar suspeita:
  1. **Não gravar `_index.json`** daquela URL (preserva o último registro válido se houver).
  2. Logar em nível `warn` com a URL, motivo da suspeita e amostra de até 200 bytes do body (cookies/headers sensíveis devem ser redactados).
  3. Contabilizar a ocorrência como **falha** para o contador do circuit breaker.
  4. A URL permanece elegível para re-crawl no próximo ciclo; **não** é marcada como `stale`.
```

---

### § 14 — Critérios de aceite · adicionar CA-13, CA-14, CA-15

**Antes** (final da lista de critérios, após item 12):
```
12. README documenta instalação, configuração, fluxo completo (discover → crawl → summarize → build-catalog → search) e estrutura dos dados produzidos.
```

**Depois (preserva item 12, adiciona três itens):**
```
12. README documenta instalação, configuração, fluxo completo (discover → crawl → summarize → build-catalog → search) e estrutura dos dados produzidos.
13. Teste unitário do rate limiter verifica que o jitter é **efetivamente aplicado**: a variância dos intervalos entre requisições em uma amostra de 30 chamadas sucessivas é estatisticamente não-nula e compatível com a faixa `±jitter_ms` configurada.
14. Teste de integração simula respostas HTTP 429 e 503 com cabeçalho `Retry-After` em ambas as formas (segundos e data HTTP) e verifica que o crawler aguarda o valor indicado antes da próxima requisição. Para `Retry-After > 60s`, verifica (a) log em `warn` e (b) pausa efetiva do crawl inteiro até o instante alvo.
15. Teste de integração simula sequência de 5xx consecutivos e verifica (a) abertura do circuit breaker após `N` falhas, (b) pausa de duração `M` minutos (tempo simulado), (c) abort com relatório estruturado após `K` falhas consecutivas no retorno. Teste separado verifica que uma resposta suspeita detectada incrementa o contador do breaker e não grava `_index.json`.
```

---

## Resumo de impactos

| § | Seção do BRIEF | Natureza da mudança |
|---|---|---|
| 1 | Cabeçalho | Versão 2.0 → 2.1 + histórico de revisões |
| 2 | RF-03 | Fixa política de trailing slash com exemplos canônicos |
| 3 | RF-03 | Explicita procedimento de deduplicação e regra *shorter-wins* |
| 4 | RF-09 | Provider default: Gemini → Anthropic `claude-haiku-4-5`; adiciona timeout e retry explícitos |
| 5 | Schema | Adiciona `metadata.canonical_of` e `metadata.stale_since` |
| 6 | Schema | Atualiza `mini_summary.model` no exemplo |
| 7 | RNF-03 | Adiciona YAML de exemplo concreto (já inclui parâmetros de RF-07 de §§ 10–13) |
| 8 | Riscos | Atualiza decisão do padrão duplicado com regra shorter-wins |
| 9 | Referências | Adiciona doc da API Anthropic Messages |
| 10 | RF-07 | Jitter no rate limiting; explicita que o limiter é global/compartilhado |
| 11 | RF-07 | Honrar `Retry-After` em 429/503; pausa do crawl inteiro para valores > 60s |
| 12 | RF-07 | Circuit breaker: abre após N falhas, pausa M minutos, aborta após K falhas pós-retorno |
| 13 | RF-07 | Detecção de resposta suspeita (título de bloqueio, body < 500B, headers WAF); não grava `_index.json` |
| 14 | Critérios de aceite | Adiciona CA-13 (jitter), CA-14 (Retry-After) e CA-15 (circuit breaker + resposta suspeita) |

---

## Itens NÃO alterados (para registro)

- Critérios de aceite **1–12 permanecem literalmente** como em v2.0; adicionados **CA-13, CA-14, CA-15** cobrindo as proteções ampliadas de RF-07 (total agora: **15** critérios).
- Escopo (573 URLs em `/transparencia-e-prestacao-de-contas`) — inalterado.
- Arquitetura hexagonal (Go 1.22+, SQLite embedded, zero dependências externas) — inalterada.
- RF-01, RF-02, RF-04, RF-05, RF-06, RF-08, RF-10, RF-11 — inalterados. **RF-07 ampliada** com cinco novas proteções (§§ 10–13).
- RNF-01, RNF-02, RNF-04, RNF-05, RNF-06 — inalterados.
- Dados empíricos de referência (573 URLs, 96 duplicados, 159 numéricos, 10 `copy_of`) — inalterados; a contagem de 10 x 14 observada na amostra pode ser esclarecida em M2 sem alterar a SPEC.

---

## Itens deferidos para fases posteriores (NÃO vão para v2.1)

Registrados em [`PLANO_CRAWLER_CRISTAL.md`](./PLANO_CRAWLER_CRISTAL.md) § 6.1 para não se perderem. Não alteram esta revisão da SPEC:

- **Janela de crawl configurável por horário** (`allowed_hours`): permitir restringir o crawl a um intervalo horário (ex.: apenas madrugada).
- **Cache HTTP opcional para desenvolvimento** (`--use-cache`): respostas HTTP salvas em `./data/.http-cache/` para iteração local sem bater no servidor.
- **Detecção de ciclos no BFS complementar**: garantir que o BFS de M5 reusa o set de URLs visitadas do crawl principal.
