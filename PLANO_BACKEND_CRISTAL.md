# Cristal Backend - Plano de Implementação (MVP Simplificado)

**Projeto**: Cristal REST API  
**Versão**: 0.1.0 (MVP)  
**Linguagem**: Go 1.22+  
**Data**: 2026-04-21  
**Autor**: Rosemberg Maia Gomes

---

## 1. Visão Geral

### 1.1 Objetivo

Implementar uma API REST minimalista em Go que recebe perguntas do usuário e retorna respostas inteligentes usando Claude (Anthropic) + servidores MCP existentes.

**MVP = 1 endpoint + Claude + MCP clients. Nada mais.**

### 1.2 Premissas

- O `data-orchestrator-mcp` (Python) já está funcional
- O `site-research-mcp` (Go) já está funcional
- Infraestrutura Anthropic já existe (`internal/adapters/llm/anthropic.go`)
- **SEM autenticação**
- **SEM sessões/contexto** (cada request é independente)
- **SEM métricas/observabilidade avançada**
- **SEM Docker/K8s** (rodar localmente com `go run`)

### 1.3 Arquitetura Simplificada

```
User → POST /chat → HTTP Server → Orquestrador → Claude API
                                        ↓             (tool use)
                                   MCP Client ←─────────┘
                                        ↓
                            ┌───────────┴───────────┐
                            ↓                       ↓
                   data-orchestrator-mcp    site-research-mcp
                        (Python)                 (Go)
```

**Fluxo:**
1. User envia `POST /chat` com uma pergunta
2. HTTP Server recebe e passa para Orquestrador
3. Orquestrador envia pergunta + ferramentas para Claude API
4. Claude decide usar uma ferramenta MCP (ex: `search_pages`)
5. Orquestrador executa a ferramenta via MCP Client
6. Resultado volta para Claude
7. Claude gera resposta final
8. Retorna resposta ao usuário

---

## 2. Componentes (Apenas o Essencial)

### 2.1 HTTP Server

**1 endpoint apenas:**
- `POST /chat` - Recebe pergunta, retorna resposta

**Framework**: `net/http` da stdlib (sem frameworks pesados)

### 2.2 Orquestrador

**Responsabilidade**: Coordenar Claude + MCP tools

**Fluxo:**
1. Recebe pergunta do usuário
2. Envia para Claude com definição das ferramentas MCP
3. Se Claude pedir tool → executa via MCP Client
4. Retorna resposta final

**System Prompt (simples)**:
```
Você é um assistente para consultas ao portal de transparência do TRE-PI.

Ferramentas disponíveis:
- search_pages: buscar páginas
- get_document: extrair dados de PDF/Excel
- inspect_page: ver detalhes de uma página

Use a ferramenta apropriada e responda em português.
```

### 2.3 MCP Client

**1 cliente apenas** que conecta aos 2 servidores MCP:
- `data-orchestrator-mcp` (Python)
- `site-research-mcp` (Go)

**Simples**: 
- Inicia 1 processo de cada no startup
- Se falhar, retorna erro ao usuário
- SEM pool, SEM restart automático (futuro)

### 2.4 Claude Integration

Reutiliza `internal/adapters/llm/anthropic.go` do projeto existente.

Adiciona suporte a **tool use** (function calling).

---

## 3. Estrutura de Diretórios (Minimalista)

```
cristal-backend/
├── cmd/
│   └── api/
│       └── main.go                    # Entry point
├── internal/
│   ├── server/
│   │   ├── server.go                  # HTTP server
│   │   └── handler.go                 # POST /chat handler
│   ├── orchestrator/
│   │   ├── orchestrator.go            # Lógica principal
│   │   └── tools.go                   # Definição de ferramentas MCP
│   ├── mcp/
│   │   ├── client.go                  # Cliente MCP genérico
│   │   └── types.go                   # Tipos MCP
│   └── llm/
│       ├── claude.go                  # Wrapper Anthropic com tool use
│       └── types.go                   # Message types
├── config.yaml                        # Configuração simples
├── go.mod
└── README.md
```

**Total: ~8 arquivos Go principais**

---

## 4. API - 1 Endpoint

### POST /chat

Envia pergunta, recebe resposta.

**Request**:
```json
{
  "message": "Quais são os balancetes de março de 2025?"
}
```

**Response** (Success):
```json
{
  "response": "Encontrei 3 páginas relacionadas aos balancetes de março de 2025:\n\n1. Balancetes Mensais 2025\n   URL: https://www.tre-pi.jus.br/...\n\n2. Balancete — Março 2025\n   URL: https://...",
  "status": "success"
}
```

**Response** (Error):
```json
{
  "error": "Erro ao buscar dados: MCP server timeout",
  "status": "error"
}
```

**Pronto. Simples assim.**

---

## 5. Integração com Claude (Anthropic)

### 5.1 Tool Use / Function Calling

Claude Sonnet 4.5 suporta "tool use" (function calling). Definimos as ferramentas MCP como tools disponíveis para o modelo.

**Exemplo de Tool Definition**:
```json
{
  "name": "search_pages",
  "description": "Busca páginas no portal de transparência do TRE-PI que respondam a uma consulta",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Consulta em português (ex: 'balancetes março 2025')"
      },
      "limit": {
        "type": "integer",
        "description": "Número máximo de resultados",
        "default": 5
      }
    },
    "required": ["query"]
  }
}
```

### 5.2 Fluxo de Conversação com Tools

1. **User message** → API REST
2. **API** carrega histórico da sessão
3. **API** → Claude API:
   - System prompt
   - Histórico de mensagens
   - User message atual
   - **Tools disponíveis**
4. **Claude** analisa e retorna:
   - Texto de resposta OU
   - `tool_use` block (ex: "preciso usar search_pages")
5. Se `tool_use`:
   - **API** executa ferramenta MCP correspondente
   - **API** → Claude API novamente com resultado da tool
   - **Claude** → resposta final com dados da tool
6. **API** retorna resposta ao usuário

### 5.3 Configuração

```yaml
# config.yaml
llm:
  provider: anthropic
  model: claude-sonnet-4-5-20250120  # ou claude-haiku-4-5 (mais rápido/barato)
  endpoint: https://api.anthropic.com
  api_key_env: ANTHROPIC_API_KEY
  max_tokens: 4096
  temperature: 0.7
  timeout_seconds: 60
  
  # Controle de custos
  max_input_tokens_per_request: 100000  # ~$0.30
  max_output_tokens_per_request: 4096   # ~$0.06
  
  # Caching (Prompt Caching)
  enable_caching: true
  cache_system_prompt: true
```

**Custo Estimado** (Claude Sonnet 4.5):
- Input: $3.00 / 1M tokens
- Output: $15.00 / 1M tokens
- Cached input: $0.30 / 1M tokens (10x mais barato)

Para 1000 mensagens/dia com média de 1k tokens input e 500 tokens output:
- Input: 1M tokens → $3.00 (ou $0.30 com cache)
- Output: 500k tokens → $7.50
- **Total: ~$10.50/dia** ($7.80 com cache)

---

## 6. Roadmap - MVP em 3-4 dias

### Dia 1: Setup + MCP Client
- [ ] Estrutura de diretórios
- [ ] `go.mod` com dependências mínimas
- [ ] Copiar `internal/mcp/` do `cristal-chat`
- [ ] Cliente MCP conecta aos 2 servidores

### Dia 2: Claude Integration
- [ ] Wrapper para `internal/adapters/llm/anthropic.go`
- [ ] Adicionar suporte a tool use
- [ ] Tipos de mensagens + ferramentas
- [ ] Teste: Claude chama tool mock e retorna resposta

### Dia 3: Orquestrador + HTTP
- [ ] Orquestrador que coordena Claude + MCP
- [ ] HTTP server com 1 endpoint `POST /chat`
- [ ] Handler que chama orquestrador
- [ ] Teste end-to-end

### Dia 4: Testes + Docs
- [ ] Testes de integração
- [ ] README com instruções de uso
- [ ] Config YAML simples
- [ ] Rodar e validar funcionamento

**Pronto para uso!**

---

## 7. Configuração (Simples)

```yaml
# config.yaml
server:
  port: 8080

mcp:
  data_orchestrator:
    python: /usr/bin/python3
    dir: ../data-orchestrator-mcp
  
  site_research:
    binary: ../site-research-mcp/site-research-mcp
    catalog: ../data/catalog.json
    fts_db: ../data/catalog.sqlite
    data_dir: ../data

anthropic:
  api_key_env: ANTHROPIC_API_KEY
  model: claude-sonnet-4-5-20250120
  max_tokens: 4096
```

---

## 8. Critérios de Aceite (MVP)

### CA-1: Servidor Inicia
- [ ] `go run ./cmd/api` inicia sem erro
- [ ] Processos MCP conectam corretamente
- [ ] Porta 8080 disponível

### CA-2: Chat Funciona - Busca
- [ ] `POST /chat` com "balancetes março 2025"
- [ ] Claude usa `search_pages` via MCP
- [ ] Response contém lista de páginas
- [ ] URLs válidas presentes

### CA-3: Chat Funciona - Documento
- [ ] Pergunta sobre PDF específico
- [ ] Claude usa `get_document` via MCP
- [ ] Response contém dados extraídos

### CA-4: Erros são Claros
- [ ] MCP falha → erro descritivo
- [ ] Claude timeout → erro descritivo
- [ ] JSON inválido → 400 Bad Request

### CA-5: Testes Passam
- [ ] `go test ./...` passa
- [ ] Pelo menos 1 teste E2E funciona

---

## 9. Dependências (Mínimas)

```go
// go.mod
module github.com/bergmaia/cristal-backend

go 1.22

require (
    gopkg.in/yaml.v3 v3.0.1        // Config
    // Reutilizar:
    // ../site-research/internal/adapters/llm (Anthropic)
    // ../cristal-chat/internal/mcp (MCP client)
)
```

**Usar stdlib quando possível**:
- `net/http` para server
- `encoding/json` para JSON
- `log/slog` para logs

---

## 10. Limitações do MVP

⚠️ **Este MVP NÃO tem:**
- Autenticação
- Rate limiting
- Sessões/contexto
- Métricas
- HTTPS
- Docker
- Testes de carga
- Observabilidade avançada

**É apenas para validar a arquitetura básica.**

---

## 11. Próximos Passos (Futuro)

**Depois do MVP funcionar:**
- [ ] Adicionar sessões/contexto
- [ ] Métricas básicas
- [ ] Rate limiting
- [ ] Autenticação simples
- [ ] Docker
- [ ] Streaming (SSE)

---

## 12. Referências

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Claude API Documentation](https://docs.anthropic.com/claude/reference/messages_post)
- [Claude Tool Use Guide](https://docs.anthropic.com/claude/docs/tool-use)
- [Go Chi Router](https://github.com/go-chi/chi)
- [Projeto site-research](./README.md)
- [Plano Chat Cristal](./PLANO_CHAT_CRISTAL.md)
- [MCP Brief](./MCP_BRIEF.md)

---

## Changelog

- **2026-04-21**: Versão inicial do plano (v0.1.0)
