# Guia de Instalação e Execução

## Pré-requisitos

### 1. Verificar Go instalado
```bash
go version
# Deve ser >= 1.22
```

### 2. Verificar Python instalado (para data-orchestrator-mcp)
```bash
python3 --version
# Deve ser >= 3.10
```

### 3. Verificar data-orchestrator-mcp
```bash
cd ../data-orchestrator-mcp
ls src/server.py
# Deve existir
```

### 4. Verificar site-research-mcp compilado
```bash
cd ../cmd/site-research-mcp
ls site-research-mcp
# Deve existir o binário
```

Se não existir:
```bash
cd ../
go build -o cmd/site-research-mcp/site-research-mcp ./cmd/site-research-mcp
```

### 5. Verificar catálogo gerado
```bash
cd ../data
ls catalog.json catalog.sqlite
# Ambos devem existir
```

Se não existir, gerar o catálogo primeiro:
```bash
cd ..
./cmd/site-research/site-research discover -config config.yaml
./cmd/site-research/site-research crawl -config config.yaml
./cmd/site-research/site-research build-catalog -config config.yaml
```

## Instalação

### 1. Entrar no diretório
```bash
cd cristal-backend
```

### 2. Download de dependências
```bash
go mod download
```

### 3. Compilar
```bash
go build -o bin/api ./cmd/api
```

## Configuração

### 1. Editar config.yaml

Ajustar paths conforme sua estrutura:

```yaml
server:
  port: 8080

mcp:
  data_orchestrator:
    command: python3
    args: ["-m", "src.server"]
    working_dir: ../data-orchestrator-mcp  # AJUSTAR SE NECESSÁRIO
    timeout: 120s

  site_research:
    command: ../cmd/site-research-mcp/site-research-mcp  # AJUSTAR SE NECESSÁRIO
    args: []
    env:
      SITE_RESEARCH_CATALOG: ../data/catalog.json
      SITE_RESEARCH_FTS_DB: ../data/catalog.sqlite
      SITE_RESEARCH_DATA_DIR: ../data
      SITE_RESEARCH_SCOPE_PREFIX: https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas
    timeout: 30s

anthropic:
  model: claude-sonnet-4-5-20250120
  max_tokens: 4096
  temperature: 0.7
```

### 2. Configurar API Key da Anthropic

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

Ou adicionar ao ~/.bashrc / ~/.zshrc:
```bash
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.bashrc
source ~/.bashrc
```

## Executar

### Modo Desenvolvimento
```bash
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/api
```

### Modo Produção
```bash
ANTHROPIC_API_KEY=sk-ant-... ./bin/api -config config.yaml
```

## Testar

### Terminal 1: Iniciar servidor
```bash
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/api
```

### Terminal 2: Fazer requisições
```bash
# Health check
curl http://localhost:8080/health

# Chat
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "O que você pode fazer?"}'

# Busca
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Busque páginas sobre balancetes de 2025"}'
```

### Ou usar script de teste
```bash
# Terminal 1: servidor rodando
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/api

# Terminal 2: testes
./test.sh
```

## Troubleshooting

### Erro: "ANTHROPIC_API_KEY environment variable is required"
```bash
# Verificar se está definida
echo $ANTHROPIC_API_KEY

# Se vazio, definir
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Erro: "failed to create MCP manager"
```bash
# Verificar paths em config.yaml
# Testar manualmente cada servidor MCP:

# data-orchestrator
cd ../data-orchestrator-mcp
python3 -m src.server
# Deve iniciar sem erro (Ctrl+C para sair)

# site-research-mcp
cd ../cmd/site-research-mcp
SITE_RESEARCH_CATALOG=../../data/catalog.json \
SITE_RESEARCH_FTS_DB=../../data/catalog.sqlite \
SITE_RESEARCH_DATA_DIR=../../data \
./site-research-mcp
# Deve iniciar sem erro (Ctrl+C para sair)
```

### Erro: "catalog.json not found"
```bash
# Gerar catálogo primeiro
cd ..
./cmd/site-research/site-research discover -config config.yaml
./cmd/site-research/site-research crawl -config config.yaml
./cmd/site-research/site-research build-catalog -config config.yaml
```

### Logs detalhados
```bash
# Adicionar -v para verbose
# (feature não implementada no MVP)

# Ver logs em tempo real
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/api 2>&1 | jq
# Formata JSON logs
```

## Próximos Passos

1. ✓ Servidor iniciado
2. ✓ Health check funcionando
3. ✓ Chat respondendo
4. 🎯 Testar diferentes tipos de consultas
5. 🎯 Monitorar custos da API Anthropic
6. 🎯 Implementar features adicionais (sessões, métricas, etc)

## Contato

Para dúvidas ou problemas, consultar:
- [README.md](README.md)
- [PLANO_BACKEND_CRISTAL.md](../PLANO_BACKEND_CRISTAL.md)
