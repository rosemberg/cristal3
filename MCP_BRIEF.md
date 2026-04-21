# MCP_BRIEF — Site Research (Fase 3: Servidor MCP)

**Versão:** 1.0
**Data:** 2026-04-20
**Autor da especificação:** Rosemberg Maia Gomes (Berg)
**Projeto relacionado:** `site-research` (monorepo). Consumidor direto dos artefatos produzidos pela Fase 1 ([`BRIEF.md`](./BRIEF.md)) e pela Fase 2 (engine de roteamento).

**Histórico de revisões:**
- v1.0 (2026-04-20): primeira redação formal da Fase 3.

## Contexto

O projeto `site-research` está estruturado em quatro fases (ver [`BRIEF.md`](./BRIEF.md) §Contexto):

1. **Crawler + Catálogo** — concluída ([`BRIEF.md`](./BRIEF.md)).
2. **Engine de roteamento** — loop programático multi-estágio sobre o catálogo.
3. **Servidor MCP** — *escopo desta spec*.
4. Consulta estruturada de datasets.

Esta Fase 3 expõe o ativo de dados e a engine de roteamento (Fases 1 e 2) a clientes MCP — Claude Desktop, Claude Code, Cowork, e qualquer outro cliente compatível com o Model Context Protocol — via um servidor local operando sobre transporte stdio.

**Esta spec cobre apenas a Fase 3.** O servidor MCP é uma **fachada fina** (thin facade): não implementa lógica de negócio, não adiciona novos tipos de consulta, não modifica o catálogo. Traduz requisições MCP em chamadas aos pacotes internos do monorepo e formata as respostas em markdown otimizado para consumo por LLM.

A premissa arquitetural é: **o catálogo e a engine, já validados por testes e inspeção manual nas Fases 1 e 2, são o único caminho correto para responder qualquer consulta**. O servidor MCP é apenas um adaptador de protocolo; qualquer lógica nova pertence à engine, não ao servidor.

## Princípios Norteadores

- **Thin facade**. O servidor MCP não contém lógica de busca, classificação ou formatação semântica. Traduz protocolo em chamadas à engine e formata saída em markdown.
- **Read-only**. Nenhuma tool efetua escrita. Operações mutativas (`crawl`, `summarize`, `build-catalog`) permanecem no CLI `site-research`, fora do MCP.
- **Respostas otimizadas para LLM, não para humanos**. As tools devolvem markdown com estrutura previsível, URLs sempre explícitas, sem cores/ANSI/tabelas ASCII largas. O cliente MCP é um LLM que reapresenta ao usuário final.
- **Rastreabilidade total**. Toda resposta inclui URLs canônicas das páginas citadas — o usuário pode sempre conferir a fonte oficial.
- **Stdio apenas nesta versão**. Sem HTTP+SSE, sem Streamable HTTP, sem autenticação. O servidor é lançado como subprocesso pelo cliente.
- **Configuração por env var, não por arquivo**. Clientes MCP passam env vars no launch; exigir arquivo de config complica o onboarding.
- **Falhas explícitas e visíveis**. Catálogo ausente, FTS corrompido, API key faltando — todos resultam em erro claro no startup (exit ≠ 0) ou em mensagem de erro MCP estruturada, nunca em respostas vazias ou silêncio.
- **Stdout é sagrado**. Apenas JSON-RPC trafega por stdout; qualquer log, progresso ou diagnóstico vai para stderr.

## Escopo

### Dentro

- Binário `site-research-mcp` no monorepo (`cmd/site-research-mcp/`).
- Transporte stdio implementando JSON-RPC 2.0 conforme especificação MCP.
- Três tools read-only: `search`, `inspect_page`, `catalog_stats`.
- Handshake MCP (`initialize`), listagem (`tools/list`), invocação (`tools/call`) e cancelamento (`notifications/cancelled`).
- Carregamento e validação do catálogo no startup.
- Formatação de respostas em markdown.
- Configuração via env vars.
- Logs estruturados em stderr (JSON).
- Testes unitários e de integração com cliente MCP mock.
- Exemplos de configuração para Claude Desktop e Claude Code.

### Fora

- HTTP, SSE, Streamable HTTP ou qualquer transporte que não seja stdio.
- Autenticação, autorização, multi-tenancy.
- Capacidades MCP fora de `tools`: `resources`, `prompts`, `sampling`, `roots` — todas desabilitadas nesta versão.
- Tools de escrita: `crawl`, `summarize`, `build-catalog` permanecem no CLI, fora do MCP.
- Extração de schemas de datasets (Fase 4).
- Telemetria remota, métricas exportadas a sistemas externos.
- Distribuição via package manager (Homebrew, apt, chocolatey) — apenas binário cross-platform nesta versão.
- Interface gráfica, TUI interativa.

## Arquitetura de Alto Nível

```
┌─────────────────────┐                ┌──────────────────────────────────────┐
│  Cliente MCP        │                │  site-research-mcp (subprocesso)     │
│                     │                │                                       │
│  - Claude Desktop   │   stdio        │  ┌────────────────────────────────┐  │
│  - Claude Code      │◄── JSON-RPC ──►│  │  cmd/site-research-mcp (main)  │  │
│  - Cowork / outros  │                │  └─────────────┬──────────────────┘  │
└─────────────────────┘                │                │                       │
                                       │  ┌─────────────▼──────────────────┐  │
                                       │  │  internal/mcp                  │  │
                                       │  │  (protocolo, transporte stdio) │  │
                                       │  └─────────────┬──────────────────┘  │
                                       │                │                       │
                                       │  ┌─────────────▼──────────────────┐  │
                                       │  │  internal/tools                │  │
                                       │  │  (search, inspect_page,        │  │
                                       │  │   catalog_stats — thin facade) │  │
                                       │  └─────────────┬──────────────────┘  │
                                       │                │                       │
                                       │  ┌─────────────▼──────────────────┐  │
                                       │  │  internal/format               │  │
                                       │  │  (markdown rendering)          │  │
                                       │  └─────────────┬──────────────────┘  │
                                       │                │                       │
                                       │  ┌─────────────▼──────────────────┐  │
                                       │  │  internal/engine   (Fase 2)    │  │
                                       │  │  internal/app      (Fase 1)    │  │
                                       │  │  internal/adapters (Fase 1)    │  │
                                       │  └─────────────┬──────────────────┘  │
                                       └────────────────┼──────────────────────┘
                                                        │
                                                        ▼
                                       ┌──────────────────────────────────────┐
                                       │  Ativo de dados (produzido na Fase 1):│
                                       │  - catalog.json                       │
                                       │  - catalog.sqlite (FTS5)              │
                                       │  - árvore de _index.json              │
                                       └──────────────────────────────────────┘
```

Observações:

- O servidor é **in-process**: `internal/engine` e adapters da Fase 1 são importados como pacotes Go do próprio módulo. Não há IPC, não há HTTP interno.
- O ativo de dados é acessado em modo somente-leitura. O servidor não invoca `crawl`, `summarize` ou `build-catalog`.
- Uma instância do servidor atende um único cliente MCP por vez (modelo subprocesso); múltiplos clientes = múltiplos subprocessos independentes.

## Requisitos Funcionais

### RF-01 — Protocolo MCP (initialize, tools/list, tools/call, cancelamento)

O servidor implementa JSON-RPC 2.0 sobre stdio conforme a especificação MCP vigente (ver §Decisões em Aberto quanto à revisão do protocolo adotada).

- **`initialize`**: recebe `protocolVersion` e `clientInfo` do cliente; responde com `serverInfo` (`name: "site-research-mcp"`, `version: <semver do binário>`) e `capabilities` contendo **apenas** `tools: {}`. `resources`, `prompts`, `sampling` e `roots` não são anunciados.
- **`tools/list`**: retorna exatamente três tools (RF-02, RF-03, RF-04) com `name`, `description` (em português, otimizada para LLM) e `inputSchema` em JSON Schema.
- **`tools/call`**: despacha para o handler correspondente. Erros de argumento (campos obrigatórios ausentes, tipos errados) retornam `isError: true` com mensagem MCP estruturada, não panic/exit.
- **`notifications/cancelled`**: quando o cliente envia cancelamento referindo-se a um `requestId` em andamento, o servidor propaga o cancelamento via `context.Context` para a tool em execução. Buscas longas (FTS, walk de fsstore) devem checar `ctx.Err()` em pontos apropriados e abortar limpamente.
- **Mensagens desconhecidas**: responder com erro JSON-RPC `-32601 Method not found`, sem crashar.
- **Parsing inválido**: responder com erro JSON-RPC `-32700 Parse error` e continuar operando (não derrubar a conexão).

### RF-02 — Tool `search`

**Descrição (exposta no `tools/list`, escrita para LLM)**:

> Busca páginas no catálogo do portal de transparência do TRE-PI que respondam a uma consulta em linguagem natural. Use quando o usuário quiser descobrir conteúdo oficial sobre um tópico (ex: "balancetes de março", "diárias de servidores", "contratos vigentes"). Retorna top-N páginas com título, mini-resumo e URL oficial. **Não** use para consultar dados tabulares dentro de anexos — este servidor apenas localiza páginas; o usuário deve abrir o PDF/XLSX pela URL retornada.

**Input schema (JSON Schema)**:

```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Consulta em português, em linguagem natural (ex: 'balancetes 2025', 'diárias magistrados'). Obrigatório.",
      "minLength": 1,
      "maxLength": 500
    },
    "limit": {
      "type": "integer",
      "description": "Número máximo de resultados. Default 10, máximo 50.",
      "minimum": 1,
      "maximum": 50,
      "default": 10
    },
    "section": {
      "type": "string",
      "description": "Filtra resultados à seção indicada (ex: 'Contabilidade', 'Recursos Humanos'). Opcional.",
      "maxLength": 120
    }
  },
  "required": ["query"],
  "additionalProperties": false
}
```

**Comportamento**:

1. Encaminhar a query à engine de roteamento (Fase 2). Se o binário for distribuído antes da Fase 2 estar pronta, o handler chama `internal/app.Search` (FTS direto) — ver §Decisões em Aberto.
2. Aplicar `limit` e, quando presente, o filtro `section`.
3. Formatar resultado em markdown (RF-08).
4. Em caso de zero hits, retornar uma seção "Nenhum resultado" explicando que a consulta não encontrou páginas e sugerindo reformulação — **nunca** retornar string vazia.

**Exemplo de resposta (markdown)**:

```markdown
# Resultados para: "balancetes 2025"

Foram encontradas **7 páginas** no catálogo (exibindo top 5).

## 1. Balancetes Mensais 2025
Relatórios contábeis mensais do exercício de 2025, consolidando receitas, despesas e disponibilidades do TRE-PI.
**Seção:** Contabilidade › Balancetes
**URL:** https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/contabilidade/balancetes/balancetes-2025

## 2. Balancete — Março 2025
Demonstrativo contábil do mês de março de 2025 com execução orçamentária detalhada.
**Seção:** Contabilidade › Balancetes › 2025
**URL:** https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/contabilidade/balancetes/balancetes-2025/marco

...

---
_Fonte: catálogo site-research (gerado em 2026-04-20). Para inspecionar uma página específica, use a ferramenta `inspect_page`._
```

### RF-03 — Tool `inspect_page`

**Descrição (exposta no `tools/list`)**:

> Retorna metadados completos de uma única página do catálogo: título, seção, breadcrumb, mini-resumo, tipo de página, documentos anexos listados, datas de publicação/atualização, e URLs de páginas filhas. Use quando o usuário perguntar detalhes sobre uma página específica retornada anteriormente por `search`, ou quando precisar entender a estrutura hierárquica ao redor de uma página. Aceita URL completa ou path relativo ao escopo (ex: "contabilidade/balancetes").

**Input schema**:

```json
{
  "type": "object",
  "properties": {
    "target": {
      "type": "string",
      "description": "URL completa (https://...) ou path relativo ao escopo do catálogo (ex: 'contabilidade/balancetes'). Obrigatório.",
      "minLength": 1,
      "maxLength": 500
    }
  },
  "required": ["target"],
  "additionalProperties": false
}
```

**Comportamento**:

1. Delegar a `internal/app.Inspect` (com `Full=false`) para resolução de URL e leitura do `_index.json`.
2. Converter a estrutura `domain.Page` em markdown (RF-08), reaproveitando os campos apresentados pelo CLI `inspect` mas formatando em seções markdown.
3. Se a página não existir, retornar `isError: true` com mensagem "Página não encontrada no catálogo: <url>. Use `search` para descobrir URLs válidas.".
4. Incluir sempre URLs explícitas de páginas filhas (até 10, truncando com nota "(+ N mais)") e documentos anexos (até 10).

**Exemplo de resposta (trecho)**:

```markdown
# Balancetes

**URL:** https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/contabilidade/balancetes
**Seção:** Contabilidade
**Tipo:** landing
**Profundidade:** 2

## Breadcrumb
Transparência e Prestação de Contas › Contabilidade › Balancetes

## Mini-resumo
Índice dos balancetes contábeis do TRE-PI, organizados por exercício.

## Páginas filhas (5)
- **Balancetes 2025** — https://www.tre-pi.jus.br/.../balancetes/balancetes-2025
- **Balancetes 2024** — https://www.tre-pi.jus.br/.../balancetes/balancetes-2024
- **Balancetes 2023** — https://www.tre-pi.jus.br/.../balancetes/balancetes-2023
- ...

## Documentos anexos (2)
- **Balancete Março 2026** (pdf) — https://www.tre-pi.jus.br/.../balancete-marco-2026.pdf
- **Consolidado 2025** (xlsx) — https://www.tre-pi.jus.br/.../consolidado-2025.xlsx

## Metadados
- **Extraído em:** 2026-04-20 18:00:00 UTC
- **Atualizado em:** 2026-04-01 (extraído do conteúdo)
- **Descoberto via:** sitemap
- **Versão do crawler:** 0.1.0
```

### RF-04 — Tool `catalog_stats`

**Descrição (exposta no `tools/list`)**:

> Retorna estatísticas agregadas sobre o catálogo: total de páginas, distribuição por profundidade hierárquica, distribuição por tipo (landing/article/listing/empty), top seções por volume, páginas sem mini-resumo, documentos anexos detectados, e páginas marcadas como stale. Use quando o usuário quiser uma visão geral do tamanho e cobertura do conteúdo indexado antes de formular buscas mais específicas.

**Input schema**:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

**Comportamento**:

1. Delegar a `internal/app.Stats` (formato interno estruturado).
2. Formatar em markdown (RF-08) com seções "Totais", "Por profundidade", "Por tipo de página", "Top seções".
3. Incluir `generated_at` e `root_url` do catálogo no cabeçalho da resposta.

**Exemplo de resposta (trecho)**:

```markdown
# Catálogo site-research — estatísticas

**Raiz:** https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas
**Schema:** v2
**Gerado em:** 2026-04-20 18:00:00 UTC

## Totais
- **Páginas:** 573
- **Sem mini-resumo:** 12
- **Com documentos anexos:** 187
- **Total de documentos listados:** 1.423
- **Páginas stale:** 3

## Por profundidade
| Profundidade | Páginas |
| ---: | ---: |
| 2 | 17 |
| 3 | 80 |
| 4 | 253 |
| ... | ... |

## Por tipo de página
- **article:** 380
- **landing:** 120
- **listing:** 50
- **empty:** 23

## Top seções (por contagem)
1. Recursos Humanos e Remuneração — 63
2. Estratégia — 51
3. ...
```

### RF-05 — Carregamento e validação do catálogo no startup

No `initialize`, o servidor valida o ambiente antes de anunciar capacidades:

- Verificar existência de `SITE_RESEARCH_CATALOG` (arquivo), `SITE_RESEARCH_FTS_DB` (arquivo), `SITE_RESEARCH_DATA_DIR` (diretório).
- Abrir o FTS5 e executar uma query trivial (`SELECT count(*) FROM pages_fts`) para confirmar que o banco está íntegro e a tabela existe.
- Ler `catalog.json`, parseá-lo e verificar `schema_version == 2` (valor atual — ver [`BRIEF.md`](./BRIEF.md) §Estrutura de Dados).
- Se qualquer uma dessas verificações falhar: logar a falha em stderr com nível `error`, responder ao `initialize` com erro JSON-RPC estruturado (`serverInfo` + erro em `result` não é suficiente; usar `error` no envelope) e **encerrar o processo com exit code 1**.
- Validações bem-sucedidas são logadas em stderr com nível `info` (contagem de páginas, tamanho do FTS em bytes, versão do schema).

### RF-06 — Configuração via env vars

O servidor lê **exclusivamente** env vars no startup. Não há flag `--config`, não há leitura de YAML.

| Variável | Obrigatória | Default | Descrição |
| --- | --- | --- | --- |
| `SITE_RESEARCH_CATALOG` | sim | — | Caminho absoluto para `catalog.json`. |
| `SITE_RESEARCH_FTS_DB` | sim | — | Caminho absoluto para `catalog.sqlite`. |
| `SITE_RESEARCH_DATA_DIR` | sim | — | Caminho absoluto para o diretório raiz do fsstore (árvore de `_index.json`). |
| `SITE_RESEARCH_SCOPE_PREFIX` | sim | — | Prefixo de escopo do catálogo (ex: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`). Usado pelo `inspect_page` para resolver paths relativos. |
| `ANTHROPIC_API_KEY` | condicional | — | Obrigatória **apenas** se alguma tool invocar a engine em modo que exija LLM; se a engine da Fase 2 não precisar de LLM em runtime de consulta, esta variável é opcional. Ver §Decisões em Aberto. |
| `SITE_RESEARCH_LOG_LEVEL` | não | `info` | `debug` \| `info` \| `warn` \| `error`. |

Ausência de qualquer variável obrigatória: exit code 2 com mensagem explícita em stderr listando quais variáveis faltam. **Nunca** substituir por valores default inventados.

### RF-07 — Logs estruturados em stderr

- Todos os logs em stderr, formato JSON (`log/slog` com handler JSON).
- **Stdout é reservado exclusivamente a JSON-RPC.** Qualquer escrita acidental em stdout (ex: `fmt.Println` de debug, panic message) corrompe o protocolo — o código deve evitar isso; o teste de integração deve verificar.
- Campos mínimos por entrada: `time` (RFC 3339), `level`, `msg`, mais campos específicos (`tool`, `query`, `duration_ms`, `hits`, `request_id`, `error`).
- Em `debug`, logar payload de `tools/call` (redactando campos sensíveis, embora nesta versão não haja campos sensíveis de input).
- **Nunca** logar conteúdo completo de páginas ou mini-resumos em níveis `info`/`warn` — apenas URLs e métricas (mesma política do [`BRIEF.md`](./BRIEF.md) RNF-06).

### RF-08 — Formatação de respostas em markdown para LLM

Regras gerais aplicáveis a todas as tools:

- Cabeçalho `# ` com título descritivo da resposta.
- URLs **sempre** renderizadas explicitamente (não usar `[texto](url)`) — o LLM cliente pode querer citar a URL literal ao usuário.
- Seções com `## ` para blocos lógicos (resultados, metadados, filhos).
- Listas com `- ` ou numeradas (`1.`, `2.`) conforme ordenação faz sentido semântico.
- Tabelas markdown apenas para dados tabulares curtos (≤ 10 linhas, ≤ 4 colunas).
- Truncar strings longas (mini-resumo > 500 chars, listas > 10 itens) com sufixo "… (+ N mais)".
- Sem emojis, sem cores ANSI, sem caracteres de controle.
- Rodapé opcional com `---` + linha em itálico indicando fonte e próximos passos (ex: "use `inspect_page` para detalhes").
- Em erros de tool (`isError: true`): resposta em markdown começando com `**Erro:** <mensagem>` seguida de sugestão de próxima ação.

## Requisitos Não-Funcionais

### RNF-01 — Stack tecnológica

- **Linguagem**: Go 1.25+ (compatível com o módulo atual — ver `go.mod`).
- **Biblioteca MCP**: **implementação from-scratch** do protocolo JSON-RPC 2.0 sobre stdio (decidido em 2026-04-20). Superfície usada é pequena — `initialize`, `tools/list`, `tools/call`, `notifications/cancelled`, `shutdown` — estimada em ~200 linhas. Justificativas: (a) zero dependências novas → binário menor e menos superfície de CVE; (b) controle total sobre escrita em stdout (contrato do transporte é crítico neste projeto); (c) tipos de protocolo ficam versionáveis com o código, facilitando evolução junto da spec MCP.
- **SQLite**: reusar `modernc.org/sqlite` já presente no projeto (driver Go puro, sem CGO, garantindo cross-compile sem cadeia de toolchain C).
- **Logs**: `log/slog` da stdlib (já em uso em `internal/logging/`).
- **JSON-RPC**: `encoding/json` da stdlib. Não introduzir framework de RPC.
- **Zero novas dependências** fora da biblioteca MCP escolhida (quando decidida).

### RNF-02 — Arquitetura e SOLID

- Arquitetura hexagonal preservada: `internal/mcp` (porta/adapter de protocolo), `internal/tools` (handlers = casos de uso), `internal/format` (formatação), tudo dependendo de abstrações já definidas em `internal/domain/ports` e funções existentes de `internal/app`/`internal/engine`.
- Os handlers de tool não conhecem o transporte; recebem contexto + args tipados e retornam markdown + erro. `internal/mcp` faz a tradução bidirecional JSON ↔ args tipados.
- Trocar stdio por HTTP no futuro (Fase 3.1) deve ser isolado a `internal/mcp/transport_*.go` sem tocar em handlers.

### RNF-03 — Testabilidade

- Handlers de tool testáveis em unidade sem stdio: invocação direta da função, assertivas sobre markdown produzido.
- Teste de integração end-to-end: processo `site-research-mcp` rodando em subprocesso com fixtures do catálogo (`fixtures/`), cliente MCP mock escrevendo/lendo JSON-RPC em pipes.
- Teste explícito de que **nenhum byte** é escrito em stdout fora de JSON-RPC válido (regressão contra `fmt.Println` acidental).
- Teste de cancelamento: disparar `notifications/cancelled` durante uma busca e verificar que o handler honra o `ctx`.
- Suite `go test ./...` executável sem rede, sem API keys, sem dependências externas.
- Cobertura mínima de 75% em `internal/mcp`, `internal/tools`, `internal/format`.

### RNF-04 — Distribuição

- Binário estático único `site-research-mcp` (cross-compilável via `GOOS`/`GOARCH`) para `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`.
- Release via GoReleaser (estratégia exata em aberto — §Decisões em Aberto).
- Sem dependência de biblioteca dinâmica em runtime (graças a `modernc.org/sqlite` CGO-free).
- Tamanho alvo do binário: < 25 MB comprimido.

### RNF-05 — Desempenho

- **Cold start < 500ms** em macOS/Linux com catálogo de 573 páginas (tempo entre invocação do processo e resposta a `initialize`).
- `search` com query típica: P95 < 200ms (FTS sobre 573 páginas é trivial).
- `inspect_page`: P95 < 100ms (leitura de um `_index.json` + conversão).
- `catalog_stats`: P95 < 500ms (leitura de `catalog.json` + walk opcional do fsstore para contagem de documentos, já implementado em `internal/app.Stats`).
- Uso de memória RSS em regime: < 80 MB.

### RNF-06 — Documentação

- `README.md` da raiz ganha nova seção "Fase 3 — MCP server" com link para este `MCP_BRIEF.md`.
- Documentação dedicada em `cmd/site-research-mcp/README.md` cobrindo: env vars, exemplos de configuração em Claude Desktop / Claude Code, troubleshooting (catálogo ausente, FTS corrompido), lista de tools.
- Cada tool tem sua descrição e schema documentados nesta spec e no código (comentários godoc).

### RNF-07 — Segurança e conformidade

- Nenhuma tool efetua escrita no catálogo, no fsstore ou no FTS. Os handlers abrem o SQLite em modo read-only (`?mode=ro`) e o fsstore apenas em leitura.
- Nenhuma tool aceita caminhos arbitrários como input — `inspect_page` resolve contra `SITE_RESEARCH_SCOPE_PREFIX` e rejeita paths que escapem do escopo (directory traversal).
- API keys (`ANTHROPIC_API_KEY`) são lidas do ambiente e **nunca** logadas.
- Logs não expõem conteúdo completo de páginas (mesma política do [`BRIEF.md`](./BRIEF.md) RNF-06).
- Limites de input (`maxLength` nos schemas) previnem payloads patológicos.

## Estrutura de Diretórios (delta sobre o monorepo)

Apenas o delta adicionado pela Fase 3 é listado; a estrutura de Fase 1/2 permanece intacta.

```
./
├── cmd/
│   ├── site-research/             # (existente, Fase 1)
│   └── site-research-mcp/         # NOVO
│       ├── main.go                # entrypoint: lê env, valida, inicia servidor MCP
│       └── README.md              # instalação e configuração em clientes MCP
├── internal/
│   ├── mcp/                       # NOVO — protocolo e transporte
│   │   ├── server.go              # orquestra lifecycle (initialize / tools/* / cancelled)
│   │   ├── transport_stdio.go     # leitura/escrita JSON-RPC em stdin/stdout
│   │   ├── protocol.go            # tipos JSON-RPC + MCP (initialize, tool call, etc.)
│   │   └── *_test.go
│   ├── tools/                     # NOVO — handlers dos 3 tools (thin facade)
│   │   ├── search.go              # delega a internal/engine (ou internal/app.Search)
│   │   ├── inspect.go             # delega a internal/app.Inspect
│   │   ├── stats.go               # delega a internal/app.Stats
│   │   └── *_test.go
│   ├── format/                    # NOVO — rendering markdown
│   │   ├── markdown.go            # helpers: renderSearchHits, renderPage, renderStats
│   │   └── markdown_test.go
│   ├── app/                       # (existente, Fase 1) — reusado in-process
│   ├── engine/                    # (Fase 2) — reusado in-process quando disponível
│   ├── adapters/                  # (existente, Fase 1) — reusado in-process
│   ├── domain/                    # (existente, Fase 1)
│   └── config/                    # (existente, Fase 1)
└── MCP_BRIEF.md                   # NOVO — esta spec
```

## Exemplos de Configuração em Clientes MCP

### Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json` no macOS)

```json
{
  "mcpServers": {
    "site-research": {
      "command": "/usr/local/bin/site-research-mcp",
      "env": {
        "SITE_RESEARCH_CATALOG": "/Users/berg/site-research/data/catalog.json",
        "SITE_RESEARCH_FTS_DB": "/Users/berg/site-research/data/catalog.sqlite",
        "SITE_RESEARCH_DATA_DIR": "/Users/berg/site-research/data",
        "SITE_RESEARCH_SCOPE_PREFIX": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas",
        "SITE_RESEARCH_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Claude Code (CLI)

```bash
claude mcp add site-research \
  --env SITE_RESEARCH_CATALOG=/Users/berg/site-research/data/catalog.json \
  --env SITE_RESEARCH_FTS_DB=/Users/berg/site-research/data/catalog.sqlite \
  --env SITE_RESEARCH_DATA_DIR=/Users/berg/site-research/data \
  --env SITE_RESEARCH_SCOPE_PREFIX=https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas \
  -- /usr/local/bin/site-research-mcp
```

### Cowork (esboço — sintaxe exata a confirmar)

Mesma estrutura dos exemplos acima: comando absoluto para o binário + bloco de env vars. Documentar forma canônica em `cmd/site-research-mcp/README.md` quando a integração for testada.

## Critérios de Aceite

A Fase 3 está completa quando todos os critérios abaixo forem atendidos:

1. Binário `site-research-mcp` compila e roda em `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, produzido por `goreleaser build --snapshot --clean` sem erros.
2. Handshake MCP: ao receber `initialize`, o servidor responde com `protocolVersion`, `serverInfo` (name + version), e `capabilities` contendo **exclusivamente** `tools: {}`. Teste de integração verifica o payload exato.
3. `tools/list` retorna exatamente três tools (`search`, `inspect_page`, `catalog_stats`) com `inputSchema` em JSON Schema válido (validado por `github.com/xeipuuv/gojsonschema` ou equivalente no teste).
4. `tools/call` para `search` com query realista (ex: `"balancetes 2025"`) retorna `content` com `type: "text"` contendo markdown formatado conforme RF-02, com ≥ 1 hit quando o catálogo de fixtures cobre o termo.
5. `tools/call` para `inspect_page` com path relativo (ex: `"contabilidade/balancetes"`) retorna markdown com metadados da página. Target inexistente retorna `isError: true` com mensagem explícita.
6. `tools/call` para `catalog_stats` sem argumentos retorna markdown com todas as seções definidas em RF-04 (totais, por profundidade, por tipo, top seções).
7. Catálogo ausente no startup (ex: `SITE_RESEARCH_CATALOG` aponta para arquivo inexistente): processo encerra com exit code 1, mensagem de erro em stderr citando a variável e o caminho, e **nada** em stdout.
8. Env var obrigatória ausente: exit code 2, stderr lista todas as variáveis faltantes em mensagem única.
9. `go test ./internal/mcp/... ./internal/tools/... ./internal/format/...` passa sem rede e sem API keys. Cobertura ≥ 75% em cada pacote.
10. Teste específico verifica que, durante a vida do processo, **stdout contém apenas linhas de JSON-RPC parseáveis** — nenhum byte espúrio, mesmo sob erro de tool, argumentos inválidos, ou catálogo corrompido em runtime.
11. Teste de cancelamento: cliente mock envia `notifications/cancelled` durante um `tools/call` longo simulado; handler observa cancelamento via `ctx.Done()` em ≤ 100ms e responde com erro de cancelamento em vez de resposta completa.
12. Cold start medido (tempo entre `exec` do binário e resposta a `initialize`) é < 500ms em macOS arm64 com o catálogo de 573 páginas da Fase 1 pré-gerado. Medição documentada em `cmd/site-research-mcp/README.md`.

## Decisões em Aberto

1. **Biblioteca MCP Go**. ✅ **DECIDIDO (2026-04-20)**: implementação from-scratch do protocolo JSON-RPC 2.0 sobre stdio, sem dependência externa de MCP. Rationale em §RNF-01.

2. **Revisão do protocolo MCP adotada**. ✅ **DECIDIDO (2026-04-20)**: revisão **`2025-11-25`** (mais recente publicada na data da decisão). Suporte a múltiplas revisões via negociação no `initialize` **não** está incluído nesta v1; o servidor aceita apenas `2025-11-25` no campo `protocolVersion` do `initialize`. Clientes com revisão distinta recebem erro explícito.

3. **Estratégia de release**. GoReleaser publicando em GitHub Releases é suficiente? Publicar em Homebrew tap institucional? Assinar binários (codesign no macOS, assinatura Authenticode no Windows)? Necessidade real depende da política de distribuição interna do TRE-PI.

4. **Integração com Fase 2 (engine)**. A engine da Fase 2 pode não estar pronta quando a Fase 3 começar. Duas opções: (a) MCP server inicia com `search` delegando ao FTS direto (via `internal/app.Search`) e é atualizado quando a engine aterrissa; (b) Fase 3 é bloqueada até Fase 2 estar pronta. Preferência inicial é (a) — entrega valor antes e a substituição do backend é local ao handler. Confirmar com o proponente.

5. **`ANTHROPIC_API_KEY` em runtime de consulta**. Se a engine da Fase 2 usar LLM no caminho da consulta (roteamento, reranking), a API key é obrigatória no startup do MCP. Se a Fase 2 mantiver LLM apenas no `summarize` (offline), a variável é ignorada no MCP. Resposta depende da spec da Fase 2.

6. **Concorrência**. Múltiplos `tools/call` em paralelo no mesmo subprocesso: servir concorrentemente (goroutine por request) ou serializar? Proposta: concorrente (cada handler é read-only, SQLite em modo `ro` suporta leituras concorrentes). Confirmar ausência de estado compartilhado mutável entre handlers.

7. **Filtro `section` em `search`**. Incluído no schema deste brief como parâmetro opcional. A engine de Fase 2 suporta esse filtro nativamente? Se não, implementar via pós-filtragem dos hits no handler. Decidir se vale a pena para v1 ou adiar para v1.1.

8. **Telemetria local**. Manter contador simples de chamadas por tool (em memória) e expor no log ao receber `shutdown`? Útil para diagnóstico de uso em piloto institucional. Não crítico para v1.

9. **Versionamento do binário**. Semver puro (`0.1.0` na primeira release) ou calendar versioning? Como o binário declara a versão no `serverInfo`? Proposta: embedar via `-ldflags "-X main.version=..."` no build do GoReleaser.

10. **Política de evolução de schemas de tool**. Qualquer mudança breaking no `inputSchema` de uma tool é uma mudança incompatível do servidor MCP. Documentar política: bump de major do binário em mudanças breaking; clientes antigos continuam funcionando ao congelar a versão do binário.

## Referências

- [`BRIEF.md`](./BRIEF.md) — Fase 1 (crawler + catálogo): schema v2 do `_index.json`, `catalog.json`, estrutura SQLite/FTS, dados empíricos de referência.
- (Futuro) `ENGINE_BRIEF.md` — spec da Fase 2 (engine de roteamento), a ser escrita antes ou em paralelo a esta.
- Especificação Model Context Protocol: `https://modelcontextprotocol.io/specification`
- JSON-RPC 2.0: `https://www.jsonrpc.org/specification`
- JSON Schema (Draft 2020-12): `https://json-schema.org/specification.html`
- Biblioteca candidata `mark3labs/mcp-go`: `https://github.com/mark3labs/mcp-go`
- SDK oficial Go (quando publicado): `https://github.com/modelcontextprotocol/go-sdk`
- GoReleaser: `https://goreleaser.com`
- Driver SQLite CGO-free: `modernc.org/sqlite` (já no `go.mod` do projeto)
