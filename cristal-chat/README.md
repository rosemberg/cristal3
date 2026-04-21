# Cristal Chat

Chat de linha de comando em Go que integra com o data-orchestrator-mcp via protocolo MCP.

## Status

**Milestone 1.1: MCP Client Básico** - ✅ COMPLETO

## Estrutura

```
cristal-chat/
├── cmd/
│   └── cristal/          # Entry point (futuro)
├── internal/
│   ├── mcp/
│   │   ├── types.go      # Tipos MCP e JSON-RPC 2.0
│   │   ├── client.go     # Cliente MCP via stdio
│   │   └── client_test.go # Testes de integração
│   ├── chat/             # Sessão e histórico (futuro)
│   ├── ui/               # REPL e formatação (futuro)
│   └── config/           # Configuração (futuro)
└── testdata/             # Mock data para testes
```

## Milestone 1.1: MCP Client Básico

### Implementado

- ✅ Estrutura de diretórios do projeto
- ✅ `internal/mcp/types.go`: Tipos JSON-RPC 2.0 e MCP
  - Request, Response, RPCError
  - InitializeRequest/Result, Tool, CallToolRequest/Result
  - ContentItem, ClientInfo, ServerInfo
- ✅ `internal/mcp/client.go`: Cliente MCP genérico
  - NewClient: Inicia processo Python via exec
  - Initialize: Handshake MCP
  - ListTools: Lista ferramentas disponíveis
  - CallTool: Executa ferramentas
  - Close: Encerra processo
  - readLoop/stderrLoop: Goroutines para I/O
- ✅ `internal/mcp/client_test.go`: Testes de integração
  - TestClientInitialize: Valida handshake
  - TestListTools: Verifica 4 tools
  - TestCallToolMetrics: Testa execução
- ✅ `go.mod`: Módulo Go 1.26.2

### Critérios de Aceite (CA-1.1 a CA-1.6)

- ✅ CA-1.1: Cliente conecta ao data-orchestrator-mcp via stdio sem erros
- ✅ CA-1.2: Handshake MCP (initialize) completa com sucesso
- ✅ CA-1.3: ListTools retorna exatamente 4 tools (research, get_cached, get_document, metrics)
- ✅ CA-1.4: CallTool("metrics", nil) retorna dados válidos
- ✅ CA-1.5: Close() encerra processo sem órfãos
- ✅ CA-1.6: `go test ./internal/mcp/...` passa com sucesso

## Testando

```bash
cd cristal-chat
go test -v ./internal/mcp/...
```

### Pré-requisitos

- Go 1.22+
- data-orchestrator-mcp instalado em `../data-orchestrator-mcp`
- Python 3.10+ com venv configurado

### Output esperado

```
=== RUN   TestClientInitialize
INFO MCP initialized server=data-orchestrator version=1.27.0
--- PASS: TestClientInitialize (2.01s)
=== RUN   TestListTools
INFO MCP initialized server=data-orchestrator version=1.27.0
--- PASS: TestListTools (2.01s)
=== RUN   TestCallToolMetrics
INFO MCP initialized server=data-orchestrator version=1.27.0
--- PASS: TestCallToolMetrics (2.01s)
PASS
ok  	github.com/bergmaia/cristal-chat/internal/mcp	6.394s
```

## Correções Realizadas

### Fix no data-orchestrator-mcp

Durante os testes, identificamos que o servidor Python estava retornando dicionários simples em `list_tools()`, mas a biblioteca MCP Python espera objetos `Tool`. Corrigimos o servidor:

**Antes:**
```python
@server.list_tools()
async def list_tools():
    return [{"name": "research", "description": "...", ...}]
```

**Depois:**
```python
from mcp.types import Tool

@server.list_tools()
async def list_tools():
    return [Tool(name="research", description="...", ...)]
```

Esta correção garante compatibilidade com a versão mais recente da biblioteca MCP Python.

## Próximos Passos

### Milestone 1.2: REPL Mínimo
- [ ] Loop interativo de input/output
- [ ] Comandos especiais: `/help`, `/quit`, `/tools`
- [ ] Query passthrough para `research`
- [ ] Prompt visual simples

## Referências

- [PLANO_CHAT_CRISTAL.md](../PLANO_CHAT_CRISTAL.md) - Plano completo de implementação
- [data-orchestrator-mcp](../data-orchestrator-mcp) - Servidor MCP Python
- [MCP Specification](https://spec.modelcontextprotocol.io/) - Protocolo MCP
