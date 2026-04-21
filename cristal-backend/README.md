# Cristal Backend - REST API

API REST em Go que orquestra um LLM (Anthropic Claude ou Vertex AI Gemini) com servidores MCP para consultas inteligentes ao portal de transparência do TRE-PI.

## Arquitetura

```
Cliente HTTP → POST /chat → Orquestrador → llm.Provider (tool use)
                                ↓              ├─ ClaudeProvider (Anthropic)
                                ↓              └─ VertexProvider (Gemini)
                           MCP Manager
                                ↓
                    ┌───────────┴───────────┐
                    ↓                       ↓
          data-orchestrator-mcp     site-research-mcp
               (Python)                  (Go)
```

## Estrutura

```
cristal-backend/
├── cmd/api/main.go                    # Entry point
├── internal/
│   ├── server/                        # HTTP server
│   ├── orchestrator/                  # Lógica principal
│   ├── mcp/                           # Cliente MCP
│   ├── llm/                           # Provider interface + Claude & Vertex adapters
│   └── config/                        # Configuração
├── config.yaml                        # Configuração
└── README.md
```

## Requisitos

- Go 1.22+
- Python 3.10+ (para data-orchestrator-mcp)
- Binário `site-research-mcp` compilado
- Catálogo gerado (`../data/catalog.json`, etc)
- Para provider Anthropic: API Key (`ANTHROPIC_API_KEY`)
- Para provider Vertex: projeto GCP com a API `aiplatform.googleapis.com` habilitada e Application Default Credentials configuradas

## Configuração

1. **Configurar paths em `config.yaml`**:

```yaml
server:
  port: 8080

mcp:
  data_orchestrator:
    command: python3
    args: ["-m", "src.server"]
    working_dir: ../data-orchestrator-mcp
    timeout: 120s

  site_research:
    command: ../cmd/site-research-mcp/site-research-mcp
    env:
      SITE_RESEARCH_CATALOG: ../data/catalog.json
      SITE_RESEARCH_FTS_DB: ../data/catalog.sqlite
      SITE_RESEARCH_DATA_DIR: ../data
      SITE_RESEARCH_SCOPE_PREFIX: https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas
    timeout: 30s

llm:
  provider: vertex          # anthropic | vertex
  max_tokens: 4096
  temperature: 0.7
  timeout: 60s

anthropic:
  model: claude-sonnet-4-5-20250120

vertex:
  project_id: meu-projeto-gcp
  location: global          # ou us-central1
  model: gemini-3-flash-preview
  # credentials_file: /path/to/service-account.json
```

2. **Autenticação do provider escolhido**:

**Anthropic** (`llm.provider: anthropic`):
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Vertex AI** (`llm.provider: vertex`) — Application Default Credentials:
```bash
# Opção 1: login interativo (dev)
gcloud auth application-default login

# Opção 2: service account (prod)
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Habilitar API no projeto (uma vez)
gcloud services enable aiplatform.googleapis.com --project=$PROJECT_ID
```

## Instalação

```bash
cd cristal-backend
go mod download
go build -o bin/api ./cmd/api
```

## Executar

```bash
# Modo desenvolvimento — Anthropic
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/api

# Modo desenvolvimento — Vertex AI (após gcloud auth application-default login)
go run ./cmd/api

# Produção
./bin/api -config config.yaml
```

## API

### POST /chat

Envia uma pergunta e recebe resposta inteligente.

**Request**:
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Quais são os balancetes de março de 2025?"}'
```

**Response**:
```json
{
  "status": "success",
  "response": "Encontrei 3 páginas relacionadas aos balancetes de março de 2025:\n\n1. Balancetes Mensais 2025\n   URL: https://www.tre-pi.jus.br/..."
}
```

**Error**:
```json
{
  "status": "error",
  "error": "processing error: tool search failed"
}
```

### GET /health

Health check.

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "ok"
}
```

## Como Funciona

1. **User** envia pergunta via POST /chat
2. **Orquestrador** envia pergunta + ferramentas para o `llm.Provider` configurado
3. **LLM** analisa e decide usar uma ferramenta MCP (ex: `search`)
4. **Orquestrador** executa a ferramenta via MCP Manager
5. **MCP Manager** roteia para o servidor correto (Python ou Go)
6. **Resultado** volta para o LLM
7. **LLM** gera resposta final com os dados
8. **Resposta** retorna ao usuário

## Ferramentas Disponíveis

O Claude tem acesso a 7 ferramentas MCP:

### data-orchestrator-mcp (Python)
- `research` - Busca + extração automática de dados
- `get_document` - Extrai dados de PDF/Excel/CSV
- `get_cached` - Consulta cache de dados
- `metrics` - Métricas do sistema

### site-research-mcp (Go)
- `search` - Busca páginas no catálogo
- `inspect_page` - Detalhes de uma página
- `catalog_stats` - Estatísticas do catálogo

## Logs

Logs estruturados em JSON no stderr:

```json
{"time":"2026-04-21T15:30:00Z","level":"INFO","msg":"processing query","query":"balancetes"}
{"time":"2026-04-21T15:30:01Z","level":"INFO","msg":"executing tool","tool":"search"}
{"time":"2026-04-21T15:30:05Z","level":"INFO","msg":"query completed","iterations":2}
```

## Troubleshooting

### MCP server não inicia

```bash
# Verificar paths em config.yaml
# Testar manualmente:
cd ../data-orchestrator-mcp
python3 -m src.server

# Para site-research-mcp:
cd ..
./cmd/site-research-mcp/site-research-mcp
```

### Timeout do LLM

Aumentar `llm.timeout` em `config.yaml`.

### Vertex AI: `could not find default credentials`

Rodar `gcloud auth application-default login` (dev) ou exportar
`GOOGLE_APPLICATION_CREDENTIALS` apontando para uma service account JSON (prod).

### Vertex AI: `PERMISSION_DENIED` / `API not enabled`

Habilitar `aiplatform.googleapis.com` no projeto e garantir que a credencial tem
o papel `roles/aiplatform.user`.

### Tool execution failed

Verificar logs do MCP server (stderr). Possíveis causas:
- Catálogo não encontrado
- Permissões de arquivo
- Python dependencies faltando

## Próximos Passos

Melhorias futuras (fora do MVP):
- [ ] Sessões/contexto de conversação
- [ ] Métricas (Prometheus)
- [ ] Rate limiting
- [ ] Autenticação (JWT)
- [ ] Streaming de respostas (SSE)
- [ ] Docker + docker-compose

## Referências

- [PLANO_BACKEND_CRISTAL.md](../PLANO_BACKEND_CRISTAL.md) - Plano completo
- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Claude Tool Use](https://docs.anthropic.com/claude/docs/tool-use)
- [data-orchestrator-mcp](../data-orchestrator-mcp)
- [site-research-mcp](../cmd/site-research-mcp)
