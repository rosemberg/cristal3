# CRISTAL — Fase 2-4
**Consulta e Relatórios Inteligentes de Transparência Automatizado Local**

MCP Server para extração e análise de dados de portais de transparência pública.

> **Nota**: Este é o projeto **CRISTAL** (Fases 2-4), construído sobre o catálogo gerado pelo **site-research** (Fase 1).

---

## 📋 Status do Projeto

🟡 **Em Especificação** - Pronto para iniciar implementação

**Última atualização**: 2026-04-20

---

## 🎯 O que é o CRISTAL?

CRISTAL é um **MCP Server** que:
- ✅ Consome dados do portal de transparência via MCP (site-research)
- ✅ Extrai e processa documentos (PDFs, CSVs) automaticamente
- ✅ Processa consultas de forma assíncrona
- ✅ Expõe ferramentas estruturadas para aplicações customizadas
- ✅ Retorna dados em JSON otimizado para consumo por LLMs

---

## 🏗️ Arquitetura

```
┌─────────────────────────────────────────┐
│  Aplicação Customizada (com LLM)        │
│  • Interpreta linguagem natural         │
│  • Chama ferramentas MCP do CRISTAL     │
└────────────────┬────────────────────────┘
                 │ MCP Protocol
┌────────────────▼────────────────────────┐
│      CRISTAL MCP Server (Fase 2-4)      │
│  ┌──────────────────────────────────┐   │
│  │  5 Ferramentas MCP               │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │  Job Queue (Celery + Redis)      │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │  Data Extractors (PDF, CSV)      │   │
│  └──────────────────────────────────┘   │
│  ┌──────────────────────────────────┐   │
│  │  Data Processor (pandas)         │   │
│  └──────────────────────────────────┘   │
└────────────────┬────────────────────────┘
                 │ MCP Client
┌────────────────▼────────────────────────┐
│   site-research MCP Server (Fase 1)     │
│   • Catálogo: 656 páginas              │
│   • 32 páginas com documentos           │
│   • 114 documentos listados             │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│   Portal TRE-PI                         │
│   transparencia-e-prestacao-de-contas   │
└─────────────────────────────────────────┘
```

---

## 🛠️ Stack Tecnológica

### Decisão: Python 3.11+

**Escolhido em vez de Go** por:
- ✅ MCP SDK oficial e maduro (Anthropic)
- ✅ pandas - imbatível para processamento de dados tabulares
- ✅ pdfplumber - extração robusta de PDFs
- ✅ Celery - sistema de filas maduro e confiável
- ✅ Desenvolvimento 2-3x mais rápido

### Componentes

```
Backend (Python)
├── MCP SDK (Anthropic oficial)
├── Celery + Redis (processamento assíncrono)
├── pandas + numpy (processamento de dados)
├── pdfplumber + poppler-utils (extração de PDFs)
├── httpx (HTTP assíncrono)
├── pydantic (validação de schemas)
└── Docker (deployment)
```

---

## 🔧 Ferramentas MCP Expostas

### 1. cristal_search
Busca e extrai dados do portal por tema/período.

```typescript
cristal_search(
  topic: string,              // "diárias", "contratos", etc
  year?: number,              // 2022, 2023
  month?: number,             // 1-12
  section?: string,           // "Recursos Humanos"
  limit?: number = 10,
  extract_documents?: boolean = true
) → {job_id: string, status: "queued"}
```

### 2. cristal_stats
Retorna estatísticas gerais do catálogo (síncrono).

```typescript
cristal_stats() → {
  total_pages: 656,
  pages_with_documents: 32,
  sections: [...],
  ...
}
```

### 3. cristal_extract_document
Extrai dados de um documento específico (PDF/CSV).

```typescript
cristal_extract_document(
  url: string,
  format?: "json" | "csv" = "json"
) → {job_id: string, status: "queued"}
```

### 4. cristal_analyze
Executa análises agregadas sobre dados.

```typescript
cristal_analyze(
  category: string,           // "diárias", "contratos"
  start_date?: string,
  end_date?: string,
  group_by?: "beneficiary" | "destination" | "month",
  top_n?: number = 10
) → {job_id: string, status: "queued"}
```

### 5. cristal_job_status
Verifica status de processamento assíncrono (síncrono).

```typescript
cristal_job_status(
  job_id: string
) → {
  status: "queued" | "processing" | "completed" | "failed",
  progress: 0-100,
  result?: {...},
  error?: string
}
```

---

## 📊 Exemplo de Uso

```python
# Aplicação customizada chama CRISTAL via MCP

# 1. Inicia busca assíncrona
result = await mcp_client.call_tool(
    "cristal_search",
    {
        "topic": "diárias",
        "year": 2022,
        "month": 8
    }
)
# Retorna: {job_id: "abc123", status: "queued"}

# 2. Aguarda ou faz polling
await asyncio.sleep(5)

# 3. Consulta resultado
status = await mcp_client.call_tool(
    "cristal_job_status",
    {"job_id": "abc123"}
)

# 4. Processa resultado
if status["status"] == "completed":
    data = status["result"]
    # {
    #   "data": {
    #     "records": [...],
    #     "summary": {
    #       "total_records": 94,
    #       "total_value": 153000.00,
    #       "unique_beneficiaries": 87
    #     },
    #     "insights": [
    #       "68 servidores (72%) viajaram para Teresina",
    #       "Valor médio de diária: R$ 1.627,66"
    #     ]
    #   },
    #   "metadata": {...}
    # }
```

---

## 📈 Casos de Uso Validados

Durante a sessão de especificação, validamos com dados reais:

✅ **Busca de diárias** (agosto/2022)  
   - 94 registros encontrados
   - R$ 153.000,00 total
   - Extração de PDF com pdftotext funcionou perfeitamente

✅ **Estatísticas do catálogo**  
   - 656 páginas catalogadas
   - 32 páginas com documentos anexos
   - 114 documentos listados

✅ **Agregações e análises**  
   - Top 5 beneficiários
   - Distribuição por destino
   - Insights gerados automaticamente

✅ **Processamento de PDF complexo**  
   - 6 páginas, 42KB
   - Tabelas extraídas corretamente
   - Dados estruturados em DataFrame

---

## 📁 Estrutura do Projeto

```
cristal/
├── src/
│   ├── server.py              # Entry point - MCP Server
│   ├── tools/                 # MCP Tools (5 ferramentas)
│   │   ├── search.py
│   │   ├── stats.py
│   │   ├── extract.py
│   │   ├── analyze.py
│   │   └── jobs.py
│   ├── mcp_clients/           # Clients para MCP externos
│   │   └── site_research.py   # Client para site-research
│   ├── extractors/            # Data extractors
│   │   ├── base.py
│   │   ├── pdf.py
│   │   └── csv.py
│   ├── processors/            # Data processing
│   │   ├── data_processor.py
│   │   └── insights.py
│   ├── workers/               # Celery workers
│   │   └── tasks.py
│   ├── cache/
│   │   └── redis_cache.py
│   ├── models/                # Pydantic models
│   │   ├── query.py
│   │   ├── result.py
│   │   └── job.py
│   └── config.py
├── tests/
│   ├── test_tools.py
│   ├── test_extractors.py
│   └── test_processors.py
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── requirements.txt
├── pyproject.toml
├── SPEC_MIDLEWARE_CRISTAL.md   # Especificação técnica completa
├── PROGRESSO.md                # Progresso e decisões
└── CRISTAL_README.md           # Este arquivo
```

---

## 🚀 Roadmap de Desenvolvimento

### Fase 1: Core MCP Server (3 semanas)
- [ ] Setup do projeto Python
- [ ] Implementação MCP Server base
- [ ] MCP Client para site-research
- [ ] Extractors básicos (PDF com pdftotext, CSV com pandas)
- [ ] Cache em memória
- [ ] Tools: `cristal_stats` e `cristal_search` (versão síncrona)

### Fase 2: Processamento Assíncrono (2 semanas)
- [ ] Integração Celery + Redis
- [ ] Job queue e workers
- [ ] Tool: `cristal_job_status`
- [ ] Atualizar `cristal_search` para async
- [ ] Tool: `cristal_extract_document`
- [ ] Cache com Redis
- [ ] Data Processor: filtering, aggregation, summarization

### Fase 3: Análises Avançadas (2 semanas)
- [ ] Tool: `cristal_analyze`
- [ ] Geração automática de insights
- [ ] Agregações complexas (group_by, top_n)
- [ ] Testes unitários (cobertura > 80%)
- [ ] Testes de integração

### Fase 4: Produção (1 semana)
- [ ] Dockerfile otimizado
- [ ] Docker Compose completo
- [ ] Documentação de deployment
- [ ] Logs estruturados (JSON)
- [ ] Otimizações de performance
- [ ] Documentação completa de uso

**Total**: 8 semanas

---

## 📚 Documentação

- **[SPEC_MIDLEWARE_CRISTAL.md](SPEC_MIDLEWARE_CRISTAL.md)** - Especificação técnica completa (v2.0)
- **[PROGRESSO.md](PROGRESSO.md)** - Progresso, decisões e próximos passos
- **[CRISTAL_README.md](CRISTAL_README.md)** - Este arquivo

---

## 🎓 Aprendizados da Sessão

1. ✅ MCP site-research funciona perfeitamente para o catálogo TRE-PI
2. ✅ pdftotext extrai PDFs de diárias com 100% de precisão
3. ✅ Dados estruturados em JSON são ideais para consumo por LLMs
4. ✅ Processamento assíncrono é essencial (PDFs demoram)
5. ✅ Insights automáticos agregam muito valor
6. ✅ Python é a escolha certa para este projeto

---

## 🔧 Próximos Passos

1. Criar estrutura inicial do projeto Python
2. Implementar MCP Server base
3. Implementar extractors (PDF, CSV)
4. Setup Celery + Redis
5. Implementar ferramentas MCP uma a uma

---

## 📊 Métricas de Sucesso

| Métrica | Target |
|---------|--------|
| Tempo de resposta síncrono | < 2s |
| Tempo de processamento async | < 30s por PDF |
| Taxa de sucesso de extração | > 95% |
| Cache hit rate | > 70% |
| Uptime | > 99% |
| Cobertura de testes | > 80% |

---

## 📝 Contexto Institucional

Projeto de pesquisa e desenvolvimento conduzido por **Rosemberg Maia Gomes** (Coordenador de Transformação Digital, COTDI/STI/TRE-PI).

**Fases do Projeto**:
- ✅ **Fase 1** (site-research) - Crawler e catálogo - **COMPLETO**
- 🟡 **Fase 2-4** (CRISTAL) - MCP Server, extração e análise - **EM ESPECIFICAÇÃO**

---

**Versão**: 0.1.0 (spec)  
**Última atualização**: 2026-04-20  
**Status**: Especificação completa, pronto para implementação
