# Especificação: Data Orchestrator MCP

## Objetivo

Camada MCP que fornece dados completos e estruturados para agentes de IA, combinando busca em catálogo local com fetch e extração automática de documentos web.

## Problema Resolvido

- MCP site-research retorna apenas metadados (título, resumo, URLs)
- Dados tabulares estão em PDFs/CSVs não indexados
- Agentes de IA precisam de dados completos, não apenas links

## Solução

MCP Orchestrator que:
1. Busca no catálogo local (via MCP site-research)
2. Detecta quando faltam dados detalhados
3. Faz fetch e extração automática dos documentos
4. Armazena em cache (JSON + Parquet)
5. Retorna dados estruturados completos

---

## Arquitetura

```
┌──────────────────────────────┐
│ Agentes de IA (Produção)     │
└──────────┬───────────────────┘
           │ MCP Protocol
           ▼
┌──────────────────────────────┐
│ data-orchestrator-mcp        │
│  ├─ research(query)          │
│  ├─ get_document(url)        │
│  └─ get_cached(query)        │
└──────────┬───────────────────┘
           │
    ┌──────┴──────┐
    ▼             ▼
MCP site-research  HTTP/WebFetch
(catálogo local)   (documentos)
```

---

## Funcionalidades (Tools MCP)

### 1. `research(query: str, force_fetch: bool = False)`

Busca completa com dados extraídos.

**Fluxo:**
1. Busca no MCP site-research
2. Verifica cache local
3. Se necessário, faz fetch dos URLs retornados
4. Extrai dados de PDFs/CSVs/Excel
5. Armazena no cache
6. Retorna dados estruturados

**Retorno:**
```json
{
  "query": "gastos diárias 2026",
  "summary": {
    "total": 108975.14,
    "count": 63,
    "period": "2026-01 a 2026-02"
  },
  "sources": [
    {"url": "...", "type": "pdf", "extracted_at": "..."}
  ],
  "data": {...}
}
```

### 2. `get_document(url: str)`

Baixa e extrai dados de documento específico.

**Suporta:**
- PDF (extração de texto e tabelas)
- CSV/Excel (conversão para estrutura tabular)

**Retorno:**
```json
{
  "url": "...",
  "type": "pdf",
  "extracted_at": "2026-04-21T10:30:00Z",
  "data": {...}
}
```

### 3. `get_cached(query: str)`

Retorna dados do cache se disponíveis.

**Retorno:**
- Dados cacheados (se existirem)
- `null` se não houver cache

---

## Armazenamento

### Cache JSON (metadados + sumários)

```
cache/
├── queries/
│   └── {hash(query)}.json
└── documents/
    └── {hash(url)}.json
```

**Estrutura query cache:**
```json
{
  "query": "gastos diárias 2026",
  "timestamp": "2026-04-21T10:30:00Z",
  "ttl": 86400,
  "summary": {
    "total": 108975.14,
    "count": 63
  },
  "data_file": "extracted/diarias_2026.parquet"
}
```

### Parquet (dados tabulares)

```
cache/extracted/
├── diarias_2026_01.parquet
├── diarias_2026_02.parquet
└── ...
```

### TTL (Time to Live)

- Cache queries: 24 horas
- Cache documents: 7 dias
- Configurável via variável de ambiente

---

## Extração de Dados

### PDF
- Ferramenta: `pypdf`
- Extrai: texto completo + tabelas detectadas
- Formato saída: estrutura tabular quando possível

### CSV/Excel
- Ferramenta: `polars`
- Lê e converte para estrutura padronizada
- Formato saída: lista de dicionários

### Detecção de Tabelas

Heurística simples:
- Linhas com valores numéricos monetários
- Cabeçalhos identificáveis (TOTAL, VALOR, DATA, etc.)
- Padrões de separadores

---

## Dependências

```txt
# MCP
mcp>=1.0.0

# HTTP
httpx[http2]>=0.27.0

# Extração
pypdf>=4.0.0
polars>=0.20.0
openpyxl>=3.1.0

# Armazenamento
pyarrow>=15.0.0

# Utilitários
pydantic>=2.6.0
python-dotenv>=1.0.0
structlog>=24.1.0
```

---

## Estrutura de Arquivos

```
data-orchestrator-mcp/
├── src/
│   ├── server.py           # Servidor MCP principal
│   ├── cache.py            # Gerenciamento de cache
│   ├── extractors/
│   │   ├── pdf.py          # Extração de PDFs
│   │   ├── spreadsheet.py  # CSV/Excel
│   │   └── base.py         # Interface base
│   ├── clients/
│   │   ├── site_research.py  # Cliente MCP site-research
│   │   └── http.py           # HTTP fetch
│   └── models.py           # Pydantic models
├── cache/                  # Cache local (gitignore)
│   ├── queries/
│   ├── documents/
│   └── extracted/
├── config.yaml             # Configurações
├── requirements.txt
└── README.md
```

---

## Configuração

### config.yaml

```yaml
mcp:
  site_research_url: "http://localhost:3000"  # MCP site-research

cache:
  directory: "./cache"
  ttl_queries: 86400      # 24h
  ttl_documents: 604800   # 7 dias

extraction:
  pdf:
    engine: "pypdf"
  spreadsheet:
    engine: "polars"

http:
  timeout: 30
  max_retries: 3
```

---

## Fluxo de Dados - Exemplo

### Pergunta: "Quanto foi gasto em diárias em 2026?"

```
1. Agent chama: research("gastos diárias 2026")
   ↓
2. Orchestrator busca: mcp_site_research.search("diárias 2026")
   Retorna: URLs de páginas sobre diárias
   ↓
3. Verifica cache: get_cached("gastos diárias 2026")
   Não encontrado
   ↓
4. Fetch URLs: 
   - https://.../diarias-janeiro-2026-pdf
   - https://.../diarias-fevereiro-2026-pdf
   ↓
5. Extrai dados:
   - PDF → texto → valores monetários
   - Identifica tabelas
   ↓
6. Agrega:
   Janeiro: R$ 37.376,93
   Fevereiro: R$ 71.598,21
   Total: R$ 108.975,14
   ↓
7. Armazena cache:
   - JSON: metadados + sumário
   - Parquet: dados tabulares completos
   ↓
8. Retorna para Agent:
   {
     "summary": {"total": 108975.14, ...},
     "data": {...}
   }
```

---

## Limitações e Não-Escopo

**O que NÃO faz:**
- ❌ Interpretação semântica complexa de documentos
- ❌ OCR de PDFs escaneados (apenas PDFs com texto)
- ❌ Processamento de imagens/gráficos
- ❌ Cache distribuído (apenas local)
- ❌ Rate limiting sofisticado
- ❌ Sistema de filas/workers
- ❌ Versionamento de cache

**Mantém simples:**
- Cache local em disco
- Processamento síncrono
- Extração baseada em heurísticas simples
- TTL fixo configurável

---

## Testes

### Teste via Claude Code

```python
# Conectar ao MCP
mcp_data_orchestrator.research("gastos diárias 2026")

# Deve retornar:
# - Total: R$ 108.975,14
# - Período: Janeiro e Fevereiro 2026
# - Fontes: URLs dos PDFs
```

### Teste Unitário Mínimo

- Extração de PDF com valores monetários
- Extração de CSV simples
- Cache hit/miss
- Cliente MCP site-research

---

## Roadmap de Implementação

### Fase 1: MVP
1. Servidor MCP básico
2. Integração com MCP site-research
3. HTTP fetch simples
4. Extração básica de PDF (texto)
5. Cache JSON simples

### Fase 2: Extração
1. Detecção de tabelas em PDFs
2. Extração de CSV/Excel com Polars
3. Armazenamento Parquet

### Fase 3: Refinamento
1. Melhorar heurísticas de extração
2. TTL configurável
3. Logging estruturado
4. Testes automatizados

---

## Métricas de Sucesso

- ✅ Responde pergunta "quanto foi gasto em X" diretamente
- ✅ Cache reduz fetches em 80%+
- ✅ Extração automática de PDFs/CSVs funcionando
- ✅ Tempo de resposta < 5s (com cache)
- ✅ Tempo de resposta < 30s (sem cache, com fetch)

---

## Manutenção

### Limpeza de Cache

Script simples:
```bash
# Limpar cache expirado
python scripts/clean_cache.py --expired

# Limpar tudo
python scripts/clean_cache.py --all
```

### Monitoramento

Logs estruturados (structlog):
- Cache hits/misses
- Fetches realizados
- Extrações bem-sucedidas/falhadas
- Tempo de processamento

---

## Conclusão

Esta camada resolve o problema de forma simples:
- Agentes perguntam → recebem dados completos
- Sem complexidade desnecessária
- Fácil manter e evoluir
- Reutilizável para outros portais
