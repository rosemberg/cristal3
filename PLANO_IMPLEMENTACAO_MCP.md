# PLAN — Plano de Implementação da Fase 3 (Servidor MCP)

## 0. Confirmação de Entendimento

Absorvidos [`BRIEF.md`](./BRIEF.md) (Fase 1 — entrega o catálogo), [`MCP_BRIEF.md`](./MCP_BRIEF.md) (Fase 3 — esta), `README.md`, estrutura atual do monorepo (`cmd/site-research/`, `internal/{app,adapters,domain,config,logging,canonical,classify}`) e `go.mod` (`github.com/bergmaia/site-research`, Go 1.25.0).

**Escopo único**: binário `site-research-mcp` no mesmo módulo, transporte stdio, três tools read-only (`search`, `inspect_page`, `catalog_stats`). Thin facade sobre `internal/app` (Fase 1) e futura `internal/engine` (Fase 2). **Fora do escopo**: HTTP/SSE, autenticação, tools de escrita, integrações externas, Fase 4.

**Reuso confirmado da Fase 1**:
- `internal/app.Search` — FTS sobre `catalog.sqlite`.
- `internal/app.Inspect` — leitura de `_index.json` via `fsstore`.
- `internal/app.Stats` — agregados sobre `catalog.json` + walk do fsstore.
- `internal/adapters/sqlitefts` — `modernc.org/sqlite` (CGO-free) já em uso.
- `internal/adapters/fsstore` — write-then-rename, leitura idempotente.
- `internal/logging` — `log/slog` handler JSON.
- `internal/config` — **será bypassado** no binário MCP (que lê apenas env vars, não YAML).
- `internal/domain/ports` — `SearchHit`, `Page`, `Catalog` são os tipos de fronteira.

**Diferenças em relação à Fase 1 que exigem cuidado**:

| Aspecto | CLI site-research | site-research-mcp |
|---|---|---|
| Entrada | flags + `config.yaml` | env vars apenas |
| Saída principal | stdout (texto legível) | stdout (**JSON-RPC apenas**) |
| Logs | stderr ou stdout, livre | stderr exclusivo (stdout é sagrado) |
| Ciclo de vida | process-per-command, curto | long-running, uma sessão MCP |
| Erros | imprime e `os.Exit(1)` | `isError: true` em resposta estruturada |
| Concorrência | serial | múltiplos `tools/call` paralelos |

**12 critérios de aceite** (do [`MCP_BRIEF.md`](./MCP_BRIEF.md) §Critérios) entendidos. Cada milestone abaixo mapeia para ao menos um.

**Decisões bloqueadoras** do `MCP_BRIEF.md` — ambas resolvidas em 2026-04-20:
- #1 (biblioteca MCP): **implementação from-scratch**.
- #2 (revisão do protocolo MCP): **`2025-11-25`** (mais recente publicada).

Demais decisões em aberto podem ser resolvidas em paralelo com a implementação. Ver §3.

---

## 1. Plano em Milestones

Proposta: **4 milestones**. Escopo menor que a Fase 1 (não há crawl, extração, LLM, classificação) — apenas protocolo, tradução e formatação. Cada milestone é testável em isolamento.

### M1 — Fundação: binário, env vars, logging, handshake MCP

**Objetivo**: estabelecer o binário `site-research-mcp` que compila, carrega env vars, valida o catálogo no startup, abre transporte stdio e responde corretamente a `initialize` e `tools/list` — mas retorna "not implemented" em `tools/call`.

**Escopo (in)**:
- Implementação from-scratch do protocolo MCP (decisão §3.1). Nenhuma dependência nova em `go.mod`.
- `cmd/site-research-mcp/main.go`:
  - Lê env vars (`SITE_RESEARCH_CATALOG`, `SITE_RESEARCH_FTS_DB`, `SITE_RESEARCH_DATA_DIR`, `SITE_RESEARCH_SCOPE_PREFIX`, `SITE_RESEARCH_LOG_LEVEL`, `ANTHROPIC_API_KEY` — condicional).
  - Exit code 2 + mensagem agregada em stderr se faltar qualquer obrigatória.
  - Valida catálogo: existência + `SELECT count(*) FROM pages_fts` + `catalog.json.schema_version == 2`. Falha → exit 1 com log em stderr.
  - Inicializa `slog` JSON em stderr com nível configurável.
  - Constrói struct `config.Config` compatível com `internal/app.*` a partir das env vars (wrapper local — **não** adicionar suporte a env vars em `internal/config`).
- `internal/mcp/protocol.go`: tipos JSON-RPC 2.0 + estruturas MCP (`InitializeParams/Result`, `Tool`, `ToolsListResult`, `CallToolParams/Result`, `ContentBlock`, `CancelledNotification`, envelope `Request`/`Response`/`Notification`/`Error`). Apenas `encoding/json` da stdlib. Tipos marcados com `omitempty` onde a spec exige campos opcionais ausentes.
- `internal/mcp/transport_stdio.go`: leitura/escrita de mensagens JSON-RPC em stdin/stdout (delimitação por linha ou Content-Length, conforme a revisão do protocolo fixada em §3.2). Writer protegido por mutex para serializar frames em ambiente concorrente (preparação para M3).
- `internal/mcp/server.go`:
  - Dispatcher de `initialize`, `tools/list`, `tools/call`, `notifications/cancelled`, `shutdown`.
  - Responde `capabilities = { tools: {} }` — **sem** `resources`, `prompts`, `sampling`, `roots`.
  - Devolve erros JSON-RPC `-32601` (method not found), `-32700` (parse error), `-32602` (invalid params) conforme corresponda.
- `internal/tools/registry.go`: enumera as três tools (`search`, `inspect_page`, `catalog_stats`) com `name`, `description`, `inputSchema` — mas handlers retornam erro "not implemented" nesta milestone.
- Embedding dos JSON Schemas das tools via `//go:embed` ou struct literal Go (decidir em M1; recomendação: Go struct literal — mais fácil de testar e refatorar).
- Teste unitário: parser JSON-RPC, dispatcher, `tools/list` produz os três schemas válidos (validação via `encoding/json` mais um sanity check de JSON Schema).
- Teste de integração mínimo: subprocesso com `initialize` → resposta correta; `tools/list` → 3 tools; `tools/call` → `isError: true` com mensagem clara.
- Teste de contrato **stdout limpo**: durante todo o ciclo, stdout contém apenas linhas/frames JSON-RPC parseáveis. Injetar falhas (catálogo ausente, env var faltando) **antes** do startup para confirmar que ainda assim nada vaza para stdout.

**Escopo (out)**:
- Lógica real das tools (M2).
- Formatação markdown (M2).
- Cancelamento propagando via `ctx` (M3 — nesta milestone só o receive da notification).
- GoReleaser, docs de cliente (M4).

**Deliverables concretos**:
- `cmd/site-research-mcp/main.go`
- `internal/mcp/{protocol,server,transport_stdio}.go` + testes
- `internal/tools/registry.go` + teste de schemas
- `go.mod`/`go.sum` atualizados
- Build script: `go build -o bin/site-research-mcp ./cmd/site-research-mcp`

**Dependências**: Fase 1 concluída (já está).

**Critério de pronto**:
- `go build ./cmd/site-research-mcp` produz binário.
- Subprocesso recebe `initialize` e responde com `capabilities = { tools: {} }` + `serverInfo` com versão embedada via `-ldflags` → **CA-2** atendido.
- `tools/list` retorna exatamente 3 tools com `inputSchema` JSON Schema válido → **CA-3** atendido.
- Env var obrigatória ausente: exit 2 + stderr agregada → **CA-8** atendido.
- Catálogo ausente/corrompido no startup: exit 1 + stderr descritivo → **CA-7** atendido.
- Teste confirma zero bytes espúrios em stdout → **CA-10** parcialmente (completar em M3 sob stress).
- `go test ./internal/mcp/... ./internal/tools/...` verde sem rede.

**Complexidade**: média. A superfície é pequena, mas exige disciplina no transporte stdio (é onde o `fmt.Println` acidental quebra tudo).

**Riscos**:
- Versão do protocolo MCP (2024-11-05, 2025-03-26, ou mais recente): fixar em M1 e documentar. Ver §3.2. Como a implementação é from-scratch, um erro de interpretação da spec é nosso — mitigação: validar contra um cliente real (Claude Desktop) em M4 e rodar a suite de conformidade MCP, se houver, durante M1.
- Disciplina no transporte stdio: qualquer `fmt.Println` residual em pacotes transitivos (`internal/app`, `internal/adapters/*`) vaza para stdout e corrompe o protocolo. Mitigação: teste de contrato do stdout nesta milestone, antes de qualquer handler real.
- Integração da `Config` sintetizada a partir de env vars com `internal/app.*`: as assinaturas esperam `*config.Config` — verificar que o struct preenchido apenas nos campos usados pelas funções de interesse (`Scope.Prefix`, `Storage.*`) é suficiente. Se exigir mais, adaptar.

---

### M2 — Handlers das três tools + formatação markdown

**Objetivo**: entregar os três handlers funcionais, delegando a `internal/app` e renderizando markdown otimizado para LLM. Ao final desta milestone, um cliente MCP consegue buscar, inspecionar e consultar estatísticas do catálogo.

**Escopo (in)**:
- `internal/format/markdown.go`:
  - `RenderSearchHits(query string, hits []ports.SearchHit, limit int, totalFound int) string`
  - `RenderPage(page *domain.Page) string`
  - `RenderStats(report StatsReport) string` (usar struct intermediário — a função `internal/app.Stats` hoje escreve direto em `io.Writer` texto/JSON; precisa de refatoração mínima — ver §3.4).
  - Helpers: truncamento com "… (+ N mais)", formatação de URLs explícitas, tabelas ≤ 4 colunas, mini_summary truncado em 500 chars.
  - Sem emojis, sem ANSI, sem caracteres de controle. Cabeçalho `#`, seções `##`, rodapé `---` + linha em itálico.
- `internal/tools/search.go`:
  - Valida args via `inputSchema` (obrigatório `query`; opcionais `limit` default 10 max 50, `section` opcional).
  - Chama `internal/app.Search` (FTS direto) **ou** `internal/engine.Search` quando Fase 2 estiver pronta (§3.3 decide). Nesta implementação inicial, usa `internal/app.Search`.
  - Filtro `section`: pós-filtragem sobre hits retornados pelo FTS. Se a Fase 2 trouxer suporte nativo, migrar.
  - Em zero hits: renderiza seção "Nenhum resultado" com sugestão.
  - `isError: true` apenas em falha real (sqlite corrompido em runtime, panic recuperável); query sem resultado é sucesso.
- `internal/tools/inspect.go`:
  - Valida args (`target` obrigatório).
  - Chama `internal/app.Inspect` adaptado para retornar o `*domain.Page` em vez de escrever em `io.Writer` — **ou** captura a saída com um buffer se a refatoração for adiada (§3.4).
  - Renderiza página inteira em markdown via `format.RenderPage`.
  - Página não encontrada: `isError: true` com mensagem "Página não encontrada no catálogo: <url>. Use `search` para descobrir URLs válidas.".
  - Directory traversal: rejeitar `target` com `..` fora do escopo antes de passar ao app.
- `internal/tools/stats.go`:
  - Sem args.
  - Chama `internal/app.Stats` retornando struct estruturado (refatoração em §3.4) ou captura via buffer JSON e re-parseia.
  - Renderiza markdown via `format.RenderStats`.
- Atualiza `internal/tools/registry.go` para despachar cada tool ao seu handler.
- Respostas seguem formato MCP: `content: [{ type: "text", text: "<markdown>" }]`.
- SQLite aberto em modo `?mode=ro` (read-only) — verificar em `internal/adapters/sqlitefts`; se já abre assim, apenas confirmar; se não, adicionar override configurável.
- Testes unitários dos renderers com golden files (`testdata/search_*.md`, `testdata/inspect_*.md`, `testdata/stats_*.md`).
- Testes dos handlers com mocks do app layer (fácil — funções puras aceitam `*config.Config`).
- Teste end-to-end com fixtures reais: catálogo pré-gerado de teste em `fixtures/mcp_catalog/`, processo MCP em subprocess, queries reais via cliente mock. **CA-4, CA-5, CA-6** atendidos aqui.

**Escopo (out)**:
- Cancelamento propagado via `ctx` (M3).
- Concorrência de `tools/call` (M3).
- GoReleaser, docs de cliente final (M4).

**Deliverables concretos**:
- `internal/format/markdown.go` + `markdown_test.go` + `testdata/*.md`
- `internal/tools/{search,inspect,stats}.go` + testes
- Refatoração mínima em `internal/app/{search,inspect,stats}.go` para expor caminhos retornando struct (ver §3.4).
- `fixtures/mcp_catalog/` com `catalog.json` + `catalog.sqlite` + árvore `_index.json` pré-gerados para teste.

**Dependências**: M1. Fase 1 em estado release (já está).

**Critério de pronto**:
- `tools/call search` com query `"balancetes"` sobre fixture retorna markdown com ≥ 1 hit, formato conforme §RF-02 do `MCP_BRIEF.md` → **CA-4**.
- `tools/call inspect_page` com path relativo retorna markdown correto; target inexistente retorna `isError` → **CA-5**.
- `tools/call catalog_stats` retorna markdown com todas as seções do §RF-04 → **CA-6**.
- Golden files reproduzíveis: alterar output obriga update explícito dos testdata.
- Cobertura ≥ 75% em `internal/format` e `internal/tools`.
- `go test ./...` verde sem rede.

**Complexidade**: média. A parte delicada é o markdown consistente; testes com golden files resolvem.

**Riscos**:
- Refatoração de `internal/app.Inspect`/`.Stats` para retornar struct pode gerar churn no CLI existente. Mitigação: expor **novas** funções `InspectPage` / `GetStats` retornando struct, mantendo as antigas intactas (`Inspect`/`Stats` continuam escrevendo em `io.Writer`). CLI não muda. Ver §3.4.
- Filtro `section` em `search`: `ports.SearchHit` precisa ter `Section` populado. Verificar; adicionar se faltar (mudança pequena em `internal/adapters/sqlitefts`).
- Golden files ficam frágeis a mudanças cosméticas. Mitigação: normalizar timestamps no renderer (usar `GeneratedAt` do input, não `time.Now()`).

---

### M3 — Cancelamento, concorrência, robustez do transporte

**Objetivo**: blindar o servidor contra cenários reais de operação: cliente cancela uma busca longa, envia várias `tools/call` em paralelo, envia mensagens malformadas, envia `shutdown`.

**Escopo (in)**:
- **Cancelamento** (§RF-01 do `MCP_BRIEF.md`):
  - Servidor mantém map `requestID → context.CancelFunc` protegido por mutex.
  - `tools/call` executa em goroutine com context cancelável.
  - `notifications/cancelled` chama `cancel()` do requestID referenciado.
  - Handlers honram `ctx.Err()` em pontos naturais: antes da query FTS, antes do walk do fsstore, no laço de agregação de documentos.
  - Resposta de cancelamento: erro JSON-RPC estruturado com código apropriado (ver spec MCP vigente — provavelmente erro customizado, não `-32800`).
- **Concorrência**:
  - Handlers stateless; SQLite em `?mode=ro` suporta leitores concorrentes.
  - Teste: disparar 10 `tools/call` simultâneos, verificar que todos respondem sem corrupção.
  - Serializar apenas a escrita no transporte stdio (um mutex ao redor do writer) — o protocolo JSON-RPC permite respostas em qualquer ordem desde que cada frame seja íntegro.
- **Robustez do transporte**:
  - Mensagem com JSON inválido → `-32700 Parse error`, conexão permanece viva.
  - Método desconhecido → `-32601 Method not found`.
  - Params mal tipados → `-32602 Invalid params`.
  - EOF no stdin → `shutdown` limpo (flush de respostas pendentes + exit 0).
  - Panic em handler → recover, log em stderr nível `error`, resposta `isError: true` ao cliente, servidor continua.
- **Teste de stress de stdout**: cliente mock envia 100 chamadas intercaladas (metade válidas, metade com argumentos inválidos) em paralelo; verificar que stdout contém apenas frames JSON-RPC parseáveis (um parser rigoroso consome tudo sem erro residual). **Completa CA-10**.
- **Teste de cancelamento** (`CA-11`): handler de `search` é substituído por um mock que bloqueia em `select { case <-ctx.Done(): ...; case <-time.After(2s): ... }`; cliente envia call + cancel 100ms depois; verificar resposta de cancelamento em ≤ 150ms.
- **Benchmark de cold start** (`CA-12`): `go test -bench=BenchmarkColdStart` que mede tempo entre `exec.Command.Start()` e primeira resposta a `initialize`. Registrar resultado em `cmd/site-research-mcp/README.md`.

**Escopo (out)**:
- GoReleaser (M4).
- Docs de cliente externo (M4).

**Deliverables concretos**:
- Evolução de `internal/mcp/server.go` com cancellation map + panic recovery.
- `internal/mcp/server_test.go` com cenários de concorrência/cancelamento.
- `internal/mcp/stdout_contract_test.go` (teste de contrato do stdout sob estresse).
- `cmd/site-research-mcp/bench_test.go` com cold start benchmark.

**Dependências**: M1 + M2.

**Critério de pronto**:
- **CA-10** atendido integralmente (contrato do stdout sob 100 calls intercaladas).
- **CA-11** atendido (cancelamento observado em ≤ 100ms após `notifications/cancelled`).
- **CA-12** atendido (cold start < 500ms medido e documentado).
- Coverage ≥ 75% em `internal/mcp`.
- Nenhum flaky em 10 execuções consecutivas de `go test -race ./...`.
- `go vet`, `go test -race` limpos.

**Complexidade**: média-alta. Concorrência + contrato binário de stdio exigem testes cuidadosos.

**Riscos**:
- `go test -race` pode expor bugs latentes no mutex do writer ou no map de cancelamentos. Vale planejar 1-2 dias só para esse debug se aparecer.
- Bibliotecas MCP de terceiros podem já fazer concorrência internamente de forma que conflite com o nosso modelo — avaliar em M1 e ajustar aqui se necessário.
- Cancelamento em `internal/app.Stats` precisa passar `ctx` adiante até o walk do fsstore; hoje o código já aceita `ctx` — verificar que o walk checa `ctx.Err()`.

---

### M4 — Distribuição, documentação, release

**Objetivo**: entregar o binário empacotado para os clientes MCP reais (Claude Desktop, Claude Code, Cowork), com documentação de instalação e troubleshooting pronta. Fase encerrada ao ponto de instalação em máquina nova em < 5 minutos.

**Escopo (in)**:
- `.goreleaser.yaml` na raiz: build para `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`. Target: binário < 25 MB, CGO_ENABLED=0.
- Versão embedada via `-ldflags "-X main.version={{.Version}}"` e exposta em `serverInfo.version` no `initialize`.
- `cmd/site-research-mcp/README.md`:
  - Instalação (GitHub Release → download → `chmod +x` → mover para PATH).
  - Env vars com exemplos (copy-paste).
  - Bloco JSON pronto para `claude_desktop_config.json` no macOS, Linux, Windows.
  - Comando `claude mcp add` pronto para Claude Code.
  - Seção para Cowork (esboço — atualizar quando a integração for testada).
  - Troubleshooting:
    - Catálogo não encontrado → como gerar (apontar para fluxo da Fase 1).
    - Env var faltando → mensagem esperada.
    - FTS corrompido → como reconstruir (`site-research build-catalog`).
    - Tool responde vazio → checar `section` e reformular query.
    - Cold start lento → checar filesystem (NFS, iCloud Drive).
  - Resultado da medição de cold start da M3.
- Atualização do `README.md` raiz com nova seção "Fase 3 — MCP server" apontando para `cmd/site-research-mcp/README.md` e para `MCP_BRIEF.md`.
- Checklist de smoke test manual (em `cmd/site-research-mcp/README.md`):
  1. Gerar catálogo via Fase 1.
  2. Configurar Claude Desktop com o bloco JSON.
  3. Reiniciar Claude Desktop.
  4. Enviar query "liste as tools disponíveis" → verificar que as 3 tools aparecem.
  5. Pedir "busque sobre balancetes" → verificar chamada a `search`.
  6. Pedir "inspecione a página X" → verificar chamada a `inspect_page`.
  7. Pedir "quantas páginas tem o catálogo" → verificar chamada a `catalog_stats`.
- Tag de release `v0.1.0-mcp` (ou schema de versionamento a decidir — §3.5).
- `go test ./...` gate no CI (ver §3.6 se entra no PR ou fica para depois).
- Validação manual com **pelo menos um cliente MCP real** (Claude Desktop ou Claude Code) antes de declarar release pronta.

**Escopo (out)**:
- Homebrew tap, distribuição em package managers — decisão adiada para versão seguinte se houver demanda.
- Assinatura de binários (codesign macOS, Authenticode Windows) — idem.
- Telemetria local de uso — ficou como decisão em aberto #8 do `MCP_BRIEF.md`, não essencial para v1.

**Deliverables concretos**:
- `.goreleaser.yaml`
- `cmd/site-research-mcp/README.md`
- Atualização do `README.md` raiz
- Tag git de release
- Binários publicados em GitHub Release (ou equivalente — §3.5)

**Dependências**: M1 + M2 + M3.

**Critério de pronto**:
- **CA-1** atendido: `goreleaser build --snapshot --clean` produz os 5 binários sem erro.
- **CA-12** confirmado com cold start medido e documentado.
- Instalação em máquina nova (macOS arm64) seguindo o README leva < 5 min até primeira query respondida por Claude Desktop.
- Smoke test manual executado e registrado.
- Todos os 12 critérios de aceite do `MCP_BRIEF.md` atendidos.

**Complexidade**: baixa-média. O trabalho é mais de empacotamento e docs do que de código.

**Riscos**:
- GoReleaser exige GitHub Actions ou runner dedicado para automação de release. Se optarmos por release manual na v1, registrar os comandos exatos no README para reprodutibilidade.
- macOS pode bloquear binário não assinado ("can't be opened because Apple cannot check it for malicious software") — documentar workaround (`xattr -d com.apple.quarantine`) ou priorizar assinatura.
- Cowork: sem acesso ao cliente para testar, ficará com documentação "best effort" até alguém validar. Não bloqueia release.

---

## 2. Estrutura de Pacotes Go Proposta (delta)

Adições à árvore existente (não altera nada da Fase 1):

```
cristal3/
├── MCP_BRIEF.md                          # (existente)
├── PLANO_IMPLEMENTACAO_MCP.md            # (este arquivo)
├── .goreleaser.yaml                      # NOVO (M4)
├── cmd/
│   ├── site-research/                    # (existente, Fase 1 — intocado)
│   └── site-research-mcp/                # NOVO
│       ├── main.go
│       ├── README.md                     # (M4)
│       └── bench_test.go                 # (M3)
├── internal/
│   ├── mcp/                              # NOVO — protocolo + transporte
│   │   ├── protocol.go
│   │   ├── server.go
│   │   ├── transport_stdio.go
│   │   ├── server_test.go
│   │   └── stdout_contract_test.go
│   ├── tools/                            # NOVO — handlers (thin facade)
│   │   ├── registry.go
│   │   ├── search.go
│   │   ├── inspect.go
│   │   ├── stats.go
│   │   └── *_test.go
│   ├── format/                           # NOVO — rendering markdown
│   │   ├── markdown.go
│   │   ├── markdown_test.go
│   │   └── testdata/
│   │       ├── search_basic.md
│   │       ├── search_empty.md
│   │       ├── inspect_page.md
│   │       └── stats.md
│   ├── app/                              # (existente — extensão mínima em search.go/inspect.go/stats.go)
│   ├── adapters/                         # (existente — sqlitefts ganha flag mode=ro se necessário)
│   ├── domain/                           # (intocado)
│   ├── config/                           # (intocado — MCP não usa)
│   └── logging/                          # (intocado — reusado)
└── fixtures/
    └── mcp_catalog/                      # NOVO (M2) — catálogo pré-gerado para testes
        ├── README.md
        ├── catalog.json
        ├── catalog.sqlite
        └── data/                         # árvore _index.json mínima
```

**Justificativas não-óbvias**:

- **Separação `internal/mcp` vs `internal/tools`**: `mcp` conhece JSON-RPC e tipos de protocolo; `tools` conhece domínio. Essa fronteira permite, no futuro, trocar stdio por HTTP/SSE em `internal/mcp/transport_*.go` sem tocar handlers.
- **`internal/format` separado**: os renderers são stateless e testáveis isoladamente com golden files; misturar com `tools/` empurraria dependência de formatação para dentro dos handlers.
- **Fixture de catálogo em `fixtures/mcp_catalog/`**: evita recrawl para rodar testes. Ficará versionado (pequeno, alguns MB).
- **MCP não usa `internal/config`**: o carregador YAML da Fase 1 é overkill para 6 env vars. Duplicar ~20 linhas de leitura de env é melhor que forçar o binário MCP a carregar dependências YAML e validação hierárquica.
- **Não há pacote `internal/engine`** nesta árvore: ele é entregue pela Fase 2. Quando aterrissar, `internal/tools/search.go` passa a importá-lo — mudança local.

---

## 3. Decisões Técnicas

### 3.1 Biblioteca MCP ✅ **DECIDIDO (2026-04-20)**

**Decisão**: **implementação from-scratch** do protocolo JSON-RPC 2.0 sobre stdio. Nenhuma dependência externa de MCP em `go.mod`.

Superfície a implementar:
- `initialize` (request/response) com `protocolVersion`, `clientInfo`, `serverInfo`, `capabilities`.
- `tools/list` (request/response) com array de tools + `inputSchema`.
- `tools/call` (request/response) com `name`, `arguments`, `content: [{type: "text", text}]`, `isError`.
- `notifications/cancelled` (notification) com `requestId` e `reason`.
- `shutdown` (request/response, trivial) + encerramento no EOF de stdin.
- Envelope JSON-RPC 2.0 (`jsonrpc: "2.0"`, `id`, `method`, `params`, `result`, `error`).
- Códigos de erro padrão JSON-RPC: `-32700` (parse), `-32600` (invalid request), `-32601` (method not found), `-32602` (invalid params), `-32603` (internal error).

Justificativas:
- **Zero dependência nova**: mantém `go.mod` enxuto (apenas o que a Fase 1 já traz).
- **Binário menor**: estimativa < 20 MB comprimido.
- **Controle total do contrato de stdout**: crítico neste projeto. Nenhuma biblioteca transitiva pode escrever em stdout por acidente.
- **Superfície pequena**: ~200 linhas de Go para o protocolo, trivialmente testáveis.
- **Versionável com o código**: migração entre revisões do protocolo MCP é editar o nosso próprio `protocol.go`, sem aguardar upstream.

Risco aceito: evoluir junto da spec MCP requer manutenção própria. Mitigação: isolar tipos por revisão em `internal/mcp/protocol.go` e, se futuramente houver necessidade de suportar múltiplas revisões, bifurcar em `protocol_<revisão>.go`.

### 3.2 Revisão do protocolo MCP adotada ✅ **DECIDIDO (2026-04-20)**

**Decisão**: revisão **`2025-11-25`** (mais recente publicada na data da decisão, confirmada em `https://modelcontextprotocol.io/specification/2025-11-25`).

Implicações para a implementação:
- Campo `protocolVersion` no `initialize` é `"2025-11-25"` (literal fixo, declarado como constante em `internal/mcp/protocol.go`).
- Se o cliente enviar `protocolVersion` diferente, o servidor responde com erro explícito (mensagem descrevendo qual revisão é suportada) e encerra a sessão. Não há negociação fallback.
- A delimitação de mensagens em stdio e a semântica de `tools/call`, `notifications/cancelled` e `shutdown` seguem o texto da revisão `2025-11-25`. Qualquer ambiguidade é resolvida consultando o texto oficial, não implementações de referência.
- Qualquer capability introduzida nessa revisão que não seja explicitamente suportada (ex: `resources`, `prompts`, `sampling`, `roots`, `elicitation` se existir) **não** é anunciada em `capabilities` — omissão é a forma correta de declarar não-suporte.

**Multi-revisão**: não suportada nesta v1. Se no futuro for necessário aceitar clientes em revisão anterior ou posterior, bifurcar em `internal/mcp/protocol_<revisão>.go` com dispatcher por versão.

**Risco residual**: se o ecossistema migrar rápido para uma revisão posterior (ex: `2026-xx-xx`) durante a implementação, avaliar se vale a pena migrar antes do release da v1. Checar antes de M4.

### 3.3 Integração com Fase 2 (engine) ⚠️ **depende de cronograma**

Se a Fase 2 (engine de roteamento) não estiver pronta quando M2 começar, o handler `search` delega diretamente a `internal/app.Search` (FTS direto). Quando a engine aterrissar, a mudança no handler é local (troca da função chamada).

**Recomendação**: prosseguir com `internal/app.Search` em M2 e registrar issue/tarefa para migrar quando `internal/engine` estiver estável. Não bloquear Fase 3 aguardando Fase 2.

### 3.4 Refatoração mínima em `internal/app`

`internal/app.Inspect` e `internal/app.Stats` hoje escrevem direto em `io.Writer` (texto ou JSON), o que não é ideal para compor com o renderer de markdown.

**Opções**:
- **(a)** Expor funções novas (`InspectPage`, `GetStats`) que retornam `*domain.Page` e `StatsReport` respectivamente, mantendo as antigas intactas.
- **(b)** Refatorar as antigas para `Get*` + thin wrapper que escreve em writer. Risco: churn no CLI.
- **(c)** Usar `bytes.Buffer` + `--format=json` + reparse. Funciona, mas feio.

**Recomendação**: **(a)**. Churn zero no CLI, compõe limpamente com `internal/tools`. `internal/app.Inspect` já tem lógica que pode ser extraída sem mudar comportamento visível.

### 3.5 Estratégia de release

Opções: GitHub Release com GoReleaser automatizado (recomendado), tag + upload manual, Homebrew tap, pacote `.deb`/`.rpm`.

**Recomendação para v1**: **GitHub Release + GoReleaser snapshot manual** (operador roda `goreleaser release --clean` localmente após `git tag`). Automação via GitHub Actions fica para v1.1 quando o CI do projeto for definido. Homebrew/.deb adiados até haver demanda.

### 3.6 CI/CD do monorepo

Não existe CI configurado hoje (inferido da ausência de `.github/workflows/`). Decisão em aberto: adicionar CI na Fase 3 ou deixar para trabalho separado?

**Recomendação**: fora do escopo desta fase. `go test ./...` rodado localmente como gate. Um workflow GitHub Actions mínimo (`go test + go vet + goreleaser build --snapshot`) fica para trabalho posterior, documentado como item deferido em §5.

### 3.7 Concorrência de `tools/call`

**Decisão**: concorrente. Todos os handlers são read-only e stateless. SQLite em modo `?mode=ro` suporta leitores paralelos. Escritas no transporte serializadas via mutex.

### 3.8 Filtro `section` no `search`

**Decisão**: incluir no v1 via pós-filtragem. Quando a engine da Fase 2 trouxer filtro nativo, migrar. Justificativa: custo baixo (~10 linhas no handler), ganho real (usuário pode limitar busca a "Contabilidade", "Recursos Humanos" etc.).

### 3.9 Versionamento do binário

**Decisão**: semver puro. v0.1.0 na primeira release. Breaking changes em `inputSchema` das tools ou em `capabilities` bumpam major. Versão embedada via `-ldflags` e exposta em `serverInfo.version`.

### 3.10 Telemetria local

Fora do escopo v1. Reconsiderar se piloto institucional demandar métricas de uso. Decisão em aberto #8 do `MCP_BRIEF.md` fica registrada.

### 3.11 SQLite em modo read-only

**Decisão**: abrir com DSN `file:<path>?mode=ro&_pragma=query_only(1)`. Verificar em M1 que `internal/adapters/sqlitefts.Open` aceita override de DSN; se não, adicionar opção na struct `Options`.

### 3.12 Logs: conteúdo em debug

**Decisão**: em `debug`, logar `requestID`, método, nome da tool, argumentos (não há campos sensíveis nos inputs desta v1, mas redação proativa de chaves com nomes como `*token*`, `*secret*`, `*key*` via helper).

### 3.13 Cobertura-alvo

**Decisão**: ≥ 75% em `internal/mcp`, `internal/tools`, `internal/format`. Abaixo do gate da Fase 1 em alguns pacotes, mas compatível com a natureza mais declarativa do código MCP (muito schema, pouca lógica). Gate aferido via `go test -cover ./...`.

---

## 4. Mapeamento Milestones × Critérios de Aceite

| Critério | Milestone |
|---|---|
| CA-1 (binário cross-compila 5 plataformas) | M4 |
| CA-2 (initialize com capabilities corretas) | M1 |
| CA-3 (tools/list com 3 schemas válidos) | M1 |
| CA-4 (tools/call search retorna markdown) | M2 |
| CA-5 (tools/call inspect_page + erro para target inexistente) | M2 |
| CA-6 (tools/call catalog_stats com todas as seções) | M2 |
| CA-7 (catálogo ausente → exit 1 + stderr) | M1 |
| CA-8 (env var obrigatória ausente → exit 2 + stderr agregada) | M1 |
| CA-9 (cobertura ≥ 75% e testes sem rede/API key) | transversal, gate em M2 e M3 |
| CA-10 (stdout contém apenas JSON-RPC, mesmo sob erro) | M1 (básico) + M3 (stress) |
| CA-11 (cancelamento propagado em ≤ 100ms) | M3 |
| CA-12 (cold start < 500ms medido) | M3 (medição) + M4 (documentação) |

---

## 5. Riscos Transversais

1. **Desvio de interpretação da spec MCP**: com implementação from-scratch, um erro de leitura da spec fica no nosso código e só aparece ao integrar com cliente real. Mitigação: validar com Claude Desktop em M4 **antes** de release; considerar rodar contra o inspector oficial (`@modelcontextprotocol/inspector`) se disponível; isolar tipos em `internal/mcp/protocol.go` para facilitar ajuste.
2. **Vazamento acidental em stdout**: `fmt.Println` debug, panic stack trace, biblioteca que loga em stdout por default. Mitigação: teste de contrato dedicado (M1 + M3), política de code review explícita.
3. **Drift de protocolo MCP entre revisões**: spec ainda evolui. Mitigação: fixar revisão, documentar, isolar tipos de protocolo em `internal/mcp/protocol.go` para migração futura.
4. **Diferença entre spec e implementação nos clientes**: Claude Desktop/Code podem divergir sutilmente da spec vigente. Mitigação: smoke test manual em M4 com pelo menos um cliente real antes de declarar release pronta.
5. **Catálogo grande**: se a Fase 2 ampliar o escopo (crawl multi-site), o cold start e walk do fsstore degradam. Monitorar; se passar de 500ms, otimizar (lazy-load do catalog.json, cachear estatísticas).
6. **Testes de integração frágeis em Windows**: pipes e subprocess no Windows têm particularidades (`\r\n`, encoding). Mitigação: rodar `go test -race ./...` em Windows como parte do smoke test de M4. Alternativa: skippar testes de subprocess no Windows com build tag se inviável.
7. **Filesystem remoto (iCloud Drive, NFS, Dropbox)**: usuários podem apontar `SITE_RESEARCH_CATALOG` para pasta sincronizada → latência alta + lock contention no SQLite. Mitigação: documentar no README que se recomenda pasta local.

---

## 6. Itens Deferidos (não implementar na Fase 3)

- **CI/CD completo** (GitHub Actions com matrix de plataformas, release automatizada).
- **Assinatura de binários** (codesign macOS, Authenticode Windows, notarization).
- **Distribuição em package managers** (Homebrew tap, apt, chocolatey).
- **Telemetria local** (contadores por tool, histograma de duração, expor em `shutdown`).
- **Transporte HTTP/SSE** e **Streamable HTTP**.
- **Capabilities MCP adicionais**: `resources` (expor catalog.json como recurso), `prompts` (prompts pré-definidos para roteamento), `sampling` (servidor pede LLM ao cliente). Revisitar após estabilização da v1.
- **Autenticação** (quando transporte HTTP entrar).
- **Multi-revisão do protocolo MCP** (suporte simultâneo a múltiplas revisões via negociação).
- **Tool `summarize_search`**: executar uma busca e sintetizar os top-N resultados em parágrafo único via LLM local. Adiado até haver demanda explícita.
- **Integração Cowork testada end-to-end**: documentação "best effort" na v1; validação real requer acesso ao cliente.

---

## 7. Próximos Passos

**Estado atual** (2026-04-20):
- ✅ `MCP_BRIEF.md` escrito e aguardando revisão final.
- ✅ Este plano escrito e aguardando revisão.
- ✅ Decisão §3.1 resolvida: **implementação from-scratch**.
- ✅ Decisão §3.2 resolvida: **revisão `2025-11-25`**.
- ⏳ Revisão final do `MCP_BRIEF.md` e deste plano pelo proponente.

**Fluxo proposto**:

1. Revisar `MCP_BRIEF.md` (spec) e aprovar ou pedir ajustes.
2. Revisar este plano e aprovar ou pedir ajustes.
3. Resolver ou aceitar recomendações provisórias de §3.3–§3.13.
4. Iniciar M1, **reportando a cada task concluída** (skeleton → env vars → startup validation → handshake → tools/list vazio → teste de contrato stdout) antes de prosseguir.
5. **Não avançar para M2 sem aprovação explícita** da conclusão de M1.
6. Mesmo gate entre M2→M3 e M3→M4.

**Próxima ação aguardando aprovação**: revisão final de spec + plano e autorização para iniciar M1.
