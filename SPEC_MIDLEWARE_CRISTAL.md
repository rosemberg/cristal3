# Especificação do Sistema CRISTAL
## Consulta e Relatórios Inteligentes de Transparência Automatizado Local

---

## 1. Visão Geral

O **CRISTAL** é um **MCP Server** que atua como middleware entre portais de transparência pública e aplicações LLM, fornecendo ferramentas estruturadas para consulta, extração e processamento de dados.

### 1.1 Arquitetura

```
Aplicação Customizada (LLM)
           ↓
    CRISTAL MCP Server
    (ferramentas estruturadas)
           ↓
    ┌──────────────────────────┐
    │  Query Planner           │ ← decide origem dos dados
    └──────────────────────────┘
           ↓
    ┌────────┬────────┬────────┐
    │ Parquet│ Redis  │ Blob   │ ← persistência multicamada
    │ Lake   │ Cache  │ Store  │
    └────────┴────────┴────────┘
           ↓ (em cache miss)
    MCP Client + HTTP Fetcher
           ↓
    site-research → URLs → Portais de Transparência
```

### 1.2 Objetivos

- Expor ferramentas MCP para consulta de dados de transparência
- Extrair e processar documentos (PDF, CSV) automaticamente
- Processar consultas de forma assíncrona
- Retornar dados estruturados (JSON) para consumo por LLMs
- Cachear dados para performance
- **Persistir dados extraídos em data lake Parquet** para eliminar re-processamento e reduzir carga nos portais externos

### 1.3 Casos de Uso

1. **Busca por tema**: `cristal_search(topic="diárias", year=2022, month=8)`
2. **Estatísticas gerais**: `cristal_stats()`
3. **Extração de documento**: `cristal_extract_document(url="...")`
4. **Análise agregada**: `cristal_analyze(category="diárias", group_by="beneficiary")`
5. **Status de job**: `cristal_job_status(job_id="...")`

---

## 2. Arquitetura do Sistema

```
┌─────────────────────────────────────────────────────────┐
│         Aplicação Customizada (com LLM)                 │
│         • Interpreta linguagem natural do usuário       │
│         • Chama ferramentas MCP do CRISTAL              │
└────────────────────┬────────────────────────────────────┘
                     │ MCP Protocol
┌────────────────────▼────────────────────────────────────┐
│              CRISTAL MCP Server                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │  MCP Tools Interface                             │  │
│  │  • cristal_search                                │  │
│  │  • cristal_extract                               │  │
│  │  • cristal_analyze                               │  │
│  │  • cristal_stats                                 │  │
│  │  • cristal_job_status                            │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Query Planner + Job Orchestrator                │  │
│  │  • Consulta manifest Parquet primeiro            │  │
│  │  • Decide: hit total / parcial / miss            │  │
│  │  • Agenda fetch+extract apenas do faltante       │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌─────────────────────┬────────────────────────────┐  │
│  │  MCP Client         │  HTTP Fetcher              │  │
│  │  (site-research)    │  (download de PDFs/CSVs)   │  │
│  └─────────────────────┴────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Data Extractors                                 │  │
│  │  • PDF → pandas DataFrame                        │  │
│  │  • CSV → pandas DataFrame                        │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Data Processor + Parquet Reader/Writer          │  │
│  │  • Filtering, Aggregation, Sorting               │  │
│  │  • Escrita particionada (category/year/month)    │  │
│  │  • Leitura via DuckDB / PyArrow                  │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Camada de persistência                          │  │
│  │  • Redis     → cache quente (catálogo, jobs)     │  │
│  │  • Blob      → PDFs/CSVs brutos (hash da URL)    │  │
│  │  • Parquet   → dados extraídos consultáveis      │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │ (somente em cache miss)
┌────────────────────▼────────────────────────────────────┐
│         Fontes Externas                                 │
│         • site-research MCP (catálogo TRE-PI)           │
│         • Portais de transparência (HTTP)               │
└─────────────────────────────────────────────────────────┘
```

---

## 3. Ferramentas MCP Expostas

O CRISTAL expõe as seguintes ferramentas via protocolo MCP:

### 3.1 cristal_search

Busca páginas e documentos no portal de transparência.

```typescript
cristal_search(
  topic: string,              // "diárias", "contratos", "licitações"
  section?: string,           // "Recursos Humanos", "Licitações"
  year?: number,              // 2022, 2023, 2024
  month?: number,             // 1-12
  limit?: number = 10,        // max resultados
  extract_documents?: boolean = true  // extrair PDFs/CSVs automaticamente
) → {
  job_id: string,            // ID para acompanhar processamento
  status: "queued" | "processing" | "completed",
  results?: SearchResult[]   // se processamento rápido (< 5s)
}
```

**Retorno (quando completed)**:
```json
{
  "query": {
    "topic": "diárias",
    "filters": {"year": 2022, "month": 8}
  },
  "pages_found": 4,
  "documents_extracted": 2,
  "data": {
    "records": [
      {
        "favorecido": "João Silva",
        "cargo": "Analista Judiciário",
        "valor": 1681.92,
        "destino": "Teresina",
        "data_inicial": "2022-08-28",
        "data_final": "2022-08-31"
      }
    ],
    "summary": {
      "total_records": 94,
      "total_value": 153000.00,
      "unique_beneficiaries": 87,
      "date_range": ["2022-08-01", "2022-08-31"]
    }
  },
  "metadata": {
    "sources": [
      {
        "url": "https://..../diarias-agosto-2022.pdf",
        "type": "pdf",
        "extracted_at": "2026-04-20T21:30:00Z"
      }
    ],
    "extraction_methods": ["pdftotext"],
    "processing_time_ms": 3421
  }
}
```

### 3.2 cristal_stats

Retorna estatísticas gerais do catálogo.

```typescript
cristal_stats() → CatalogStats
```

**Retorno**:
```json
{
  "total_pages": 656,
  "pages_with_documents": 32,
  "total_documents": 114,
  "sections": [
    {"name": "Colegiados", "page_count": 107},
    {"name": "Gestão", "page_count": 96}
  ],
  "page_types": {
    "landing": 327,
    "article": 224,
    "listing": 31,
    "empty": 74
  },
  "last_updated": "2026-04-20T21:05:58Z"
}
```

### 3.3 cristal_extract_document

Extrai dados de um documento específico.

```typescript
cristal_extract_document(
  url: string,                // URL do PDF/CSV
  format?: "json" | "csv" = "json"
) → {
  job_id: string,
  status: string,
  data?: ExtractedData
}
```

**Retorno**:
```json
{
  "document": {
    "url": "https://.../diarias-agosto-2022.pdf",
    "type": "pdf",
    "pages": 6,
    "file_size_mb": 0.043
  },
  "data": {
    "records": [...],
    "columns": ["favorecido", "cargo", "valor", "destino", "data_inicial"],
    "row_count": 94
  },
  "extraction": {
    "method": "pdftotext",
    "success": true,
    "extracted_at": "2026-04-20T21:30:00Z"
  }
}
```

### 3.4 cristal_analyze

Executa análises agregadas sobre dados.

```typescript
cristal_analyze(
  category: string,           // "diárias", "contratos"
  start_date?: string,        // "2022-01-01"
  end_date?: string,          // "2022-12-31"
  group_by?: "beneficiary" | "destination" | "month" | "category",
  top_n?: number = 10
) → {
  job_id: string,
  status: string,
  analysis?: AnalysisResult
}
```

**Retorno**:
```json
{
  "analysis": {
    "period": {"start": "2022-08-01", "end": "2022-08-31"},
    "totals": {
      "records": 94,
      "total_value": 153000.00,
      "avg_value": 1627.66
    },
    "grouped": [
      {
        "group": "José de Ribamar Portela de Carvalho",
        "count": 3,
        "total_value": 11777.92,
        "percentage": 7.7
      }
    ],
    "insights": [
      "68 servidores (72%) viajaram para Teresina",
      "Valor médio de diária: R$ 1.627,66",
      "Maior beneficiário: José de Ribamar (R$ 11.777,92)"
    ]
  }
}
```

### 3.5 cristal_job_status

Verifica status de processamento assíncrono.

```typescript
cristal_job_status(
  job_id: string
) → JobStatus
```

**Retorno**:
```json
{
  "job_id": "abc123",
  "status": "completed",
  "progress": 100,
  "created_at": "2026-04-20T21:30:00Z",
  "completed_at": "2026-04-20T21:30:15Z",
  "result": { /* resultado da operação */ },
  "error": null
}

---

## 4. Componentes Internos

### 4.1 Data Extractors

**Responsabilidade**: Extrair dados estruturados de documentos.

#### 4.1.1 PDF Extractor
```python
class PDFExtractor:
    async def extract(self, pdf_url: str) -> pd.DataFrame:
        """
        Baixa e extrai dados de PDF.
        
        Pipeline:
        1. Download do PDF
        2. Extração de texto (pdftotext)
        3. Parsing para estrutura tabular
        4. Retorna DataFrame
        
        Fallback: Se PDF for imagem, usa OCR (Tesseract)
        """
```

#### 4.1.2 CSV Extractor
```python
class CSVExtractor:
    async def extract(self, csv_url: str) -> pd.DataFrame:
        """
        Baixa e processa CSV.
        
        Features:
        - Auto-detecção de encoding (chardet)
        - Auto-detecção de delimitador
        - Type inference para colunas
        - Limpeza de dados (strip, lowercase)
        """
```

**Tecnologias**:
- PDF: `poppler-utils` (pdftotext), `pdfplumber`, `tesseract-ocr`
- CSV: `pandas`, `chardet`
- HTTP: `httpx` (async)

### 4.2 Data Processor

**Responsabilidade**: Processar e agregar dados extraídos.

```python
class DataProcessor:
    def filter_by_date(self, df: pd.DataFrame, year: int, month: int = None) -> pd.DataFrame:
        """Filtra por período"""
        
    def aggregate(self, df: pd.DataFrame, group_by: str) -> pd.DataFrame:
        """
        Agrupa e totaliza.
        group_by: "beneficiary", "destination", "month", "category"
        """
        
    def summarize(self, df: pd.DataFrame) -> dict:
        """
        Gera estatísticas:
        - total_records, total_value, avg_value
        - unique beneficiaries/destinations
        - date_range
        """
        
    def generate_insights(self, df: pd.DataFrame) -> List[str]:
        """
        Gera insights em linguagem natural:
        - "68 servidores (72%) viajaram para Teresina"
        - "Valor médio de diária: R$ 1.627,66"
        """
```

### 4.3 MCP Client Layer

**Responsabilidade**: Consumir servidores MCP externos.

```python
class SiteResearchClient:
    async def search(self, query: str, section: str = None, limit: int = 10):
        """Busca no catálogo site-research"""
        
    async def inspect_page(self, url: str):
        """Obtém detalhes de página específica"""
        
    async def catalog_stats(self):
        """Obtém estatísticas do catálogo"""
```

**Observação**: O `site-research` retorna **metadados e URLs**, não os bytes dos documentos. O download efetivo é responsabilidade do HTTP Fetcher (seção 4.4).

### 4.4 HTTP Fetcher

**Responsabilidade**: Baixar documentos (PDFs, CSVs) diretamente dos portais, a partir de URLs fornecidas pelo `site-research`.

```python
class HTTPFetcher:
    def __init__(self, allowed_domains: List[str], max_size_mb: int = 50):
        self.allowed_domains = allowed_domains
        self.max_size_mb = max_size_mb
        
    async def fetch(self, url: str) -> bytes:
        """
        Baixa um documento com as seguintes garantias:
        - Valida whitelist de domínios (proteção contra SSRF)
        - Aplica limite de tamanho
        - Timeout configurável
        - Retry com backoff exponencial em falhas transientes
        - Deduplicação por hash da URL (blob store)
        """
        
    async def fetch_many(self, urls: List[str], concurrency: int = 4) -> List[FetchResult]:
        """
        Download paralelo com tolerância a falhas parciais.
        Retorna sucessos e falhas separadamente.
        """
```

**Tecnologias**: `httpx` (async), `aiofiles`, `tenacity` (retry).

**Política de cache**: Arquivos baixados são armazenados em blob store (sistema de arquivos ou S3-compatível) indexados por `sha256(url)`. Como documentos públicos são imutáveis, não há TTL — apenas política de LRU se o volume exceder limite configurado.

### 4.5 Job Queue

**Responsabilidade**: Processar tarefas assíncronas.

```python
# Usando Celery ou RQ
@celery.task
async def process_search_task(query_params: dict) -> dict:
    """
    Pipeline assíncrono com short-circuit via Parquet:
    
    1. Query Planner consulta manifest Parquet
    2. Se cobertura total: lê Parquet e retorna (sem fetch externo)
    3. Se cobertura parcial ou nula:
       a. Busca URLs faltantes no MCP site-research
       b. HTTP Fetcher baixa documentos (paralelo, com retry)
       c. Extractors convertem em DataFrames
       d. Parquet Writer grava novas partições
       e. Processor executa agregações sobre o conjunto completo
    4. Retorna resultado
    """
```

**Tecnologias**: `celery` + `redis` (broker/backend)

### 4.6 Cache Manager

**Responsabilidade**: Gerenciar cache de consultas e documentos para melhorar performance.

```python
class CacheManager:
    def __init__(self, ttl: int = 900):  # 15 minutos default
        self.ttl = ttl
        
    def get(self, key: str) -> Optional[Any]:
        """Recupera item do cache"""
        
    def set(self, key: str, value: Any, ttl: Optional[int] = None):
        """Armazena item no cache"""
        
    def invalidate(self, pattern: str):
        """Invalida cache por padrão"""
```

**Estratégias de Cache**:
- Consultas MCP: 15 minutos
- Documentos baixados: 1 hora
- Estatísticas gerais: 6 horas
- Dados processados: 30 minutos

**Tecnologias**: `redis`, `diskcache`, ou cache em memória simples

### 4.7 Parquet Data Lake

**Responsabilidade**: Persistir dados extraídos em formato colunar consultável, eliminando re-processamento em consultas futuras.

**Motivação**: Redis resolve o caso de *exatamente a mesma consulta repetida*. Parquet resolve o caso mais comum: *consultas diferentes sobre o mesmo período já extraído*. Um PDF de "diárias agosto 2022" uma vez processado vira um conjunto de registros estruturados que atende a qualquer pergunta sobre aquele período — por beneficiário, por destino, por faixa de valor — sem tocar no portal nem no PDF original.

#### 4.7.1 Estrutura de Particionamento

```
/data/cristal/
├── diarias/
│   ├── year=2022/month=08/data.parquet
│   ├── year=2022/month=09/data.parquet
│   └── year=2023/month=01/data.parquet
├── contratos/
│   ├── year=2023/data.parquet
│   └── year=2024/data.parquet
├── licitacoes/
│   └── year=2024/quarter=Q1/data.parquet
└── _manifest/
    └── sources.parquet     ← índice de procedência
```

O particionamento segue a granularidade natural de cada categoria: diárias por mês, contratos por ano, licitações por trimestre. Isso permite que `cristal_analyze(category="diárias", year=2022)` leia apenas as 12 partições relevantes.

#### 4.7.2 Manifest de Procedência

Arquivo Parquet especial (`_manifest/sources.parquet`) que registra a origem e validade de cada partição:

| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `category` | string | "diárias", "contratos", etc. |
| `partition_path` | string | Caminho relativo da partição |
| `source_url` | string | URL original do documento |
| `source_hash` | string | SHA-256 do arquivo baixado |
| `extracted_at` | timestamp | Quando foi extraído |
| `extractor_version` | string | Versão do extractor (v1, v2...) |
| `row_count` | int | Registros na partição |
| `is_frozen` | bool | Dado de período fechado (nunca re-extrai) |
| `valid_until` | timestamp | Null se frozen; TTL se dado corrente |

O manifest é o **componente crítico** para o Query Planner decidir se uma consulta está coberta.

#### 4.7.3 Reader/Writer

```python
class ParquetStore:
    async def query(
        self,
        category: str,
        filters: dict
    ) -> pd.DataFrame:
        """
        Lê partições relevantes usando DuckDB ou PyArrow.
        Pushdown de filtros para otimização.
        """
        
    async def write(
        self,
        df: pd.DataFrame,
        category: str,
        partition_keys: dict,
        source_metadata: dict
    ) -> None:
        """
        Escreve partição e atualiza manifest atomicamente.
        """
        
    async def check_coverage(
        self,
        category: str,
        filters: dict
    ) -> CoverageReport:
        """
        Consulta manifest e retorna:
        - hit_total: todos os dados pedidos estão no lake
        - hit_partial: alguns presentes, outros ausentes
        - miss: nada presente
        - missing_partitions: lista de chaves a buscar
        """
```

**Tecnologias**: `pyarrow` (leitura/escrita), `duckdb` (queries SQL analíticas sobre Parquet), `pandas` (interoperabilidade com extractors).

#### 4.7.4 Política de Atualização

- **Dados *frozen*** (períodos fechados): extraídos uma vez, nunca re-extraídos a não ser por `force_refresh` explícito ou upgrade de `extractor_version`.
- **Dados *live*** (em andamento): `valid_until` define quando o Query Planner deve considerar a partição expirada e buscar versão nova. Ex.: contratos com aditivos pendentes.
- **Schema evolution**: mudanças no extractor incrementam `extractor_version` no manifest. Partições antigas continuam válidas até re-extração seletiva.

### 4.8 Query Planner

**Responsabilidade**: Decidir, antes de qualquer acesso externo, se uma consulta pode ser servida pelo Parquet lake.

```python
class QueryPlanner:
    async def plan(self, tool_name: str, params: dict) -> QueryPlan:
        """
        Produz um plano de execução:
        - Se hit total: read_only plan → lê Parquet, responde síncrono
        - Se hit parcial: hybrid plan → lê parcial + agenda fetch do faltante
        - Se miss: full plan → fluxo completo (MCP + Fetcher + Extract + Write)
        """
```

**Árvore de decisão para `cristal_search(topic="diárias", year=2022, month=8)`**:

```
1. Planner consulta manifest: existe partição diarias/year=2022/month=08?
   ├─ SIM e não expirada → QueryPlan(mode="cache_hit", source="parquet")
   │                       → lê Parquet, responde em ~100ms, sem job
   │
   ├─ SIM mas expirada     → QueryPlan(mode="refresh", source="parquet+fetch")
   │                       → serve do Parquet, agenda refresh em background
   │
   └─ NÃO                  → QueryPlan(mode="cache_miss", source="fetch")
                           → fluxo completo assíncrono, grava Parquet ao fim
```

**Parâmetro `force_refresh`**: Todas as tools que consultam dados aceitam `force_refresh: bool = false`. Quando `true`, ignora o Parquet e executa o fluxo completo (útil para corrigir dados após mudanças no portal ou testar novas versões de extractors).

---

## 5. Fluxo de Operação

### 5.1 Fluxo de Consulta com Query Planner

```
Aplicação LLM chama:
  cristal_search(topic="diárias", year=2022, month=8)
         ↓
Query Planner consulta manifest Parquet
         ↓
    ┌────┴────┬─────────────┐
    ▼         ▼             ▼
  hit      parcial         miss
   │         │              │
   │    ┌────┴────┐         │
   │    │         │         │
   │  lê       agenda       │
   │  Parquet  fetch do     │
   │  parcial  faltante     │
   │    │         │         │
   │    ▼         ▼         ▼
   │  mescla   Worker assíncrono executa:
   │    │       1. MCP site-research → URLs
   │    │       2. HTTP Fetcher → PDFs/CSVs
   │    │       3. Extractors → DataFrames
   │    │       4. Parquet Writer → grava partições
   │    │       5. Processor → agrega
   │    │       6. Gera insights
   │    │
   ▼    ▼               ▼
 RESPOSTA (síncrona)   job_id (assíncrona)
         ↓                 ↓
     ~100ms         cristal_job_status → resultado
```

**Ganho esperado**: Após o sistema estar "aquecido" com extrações históricas, a grande maioria das consultas analíticas (`cristal_analyze`, `cristal_search` sobre períodos fechados) deve responder em modo síncrono sem tocar em fontes externas.

### 5.2 Fluxo Síncrono (queries rápidas)

Para consultas simples (< 5s), CRISTAL pode retornar imediatamente:

```
cristal_stats()
    ↓
CRISTAL consulta cache
    ↓
Se cache hit: retorna imediatamente
Se cache miss:
    ↓
    Consulta site-research.catalog_stats()
    ↓
    Cacheia resultado (TTL 6h)
    ↓
    Retorna dados
```

### 5.3 Exemplo Completo: Busca de Diárias

**Chamada MCP**:
```json
{
  "tool": "cristal_search",
  "parameters": {
    "topic": "diárias",
    "year": 2022,
    "month": 8,
    "extract_documents": true
  }
}
```

**Pipeline de Processamento**:

1. **Busca no catálogo**
```python
results = await site_research.search(query="diárias 2022", limit=20)
# Retorna 4 páginas
```

2. **Inspeção de páginas**
```python
for page in results:
    details = await site_research.inspect_page(page.url)
    if details.documents:
        document_urls.extend(details.documents)
# Encontra 2 PDFs
```

3. **Extração de documentos**
```python
df = await pdf_extractor.extract(
    "https://.../diarias-agosto-2022.pdf"
)
# DataFrame com 94 linhas
```

4. **Filtragem e agregação**
```python
df_filtered = df[df['data_inicial'].dt.month == 8]
summary = {
    "total_records": len(df_filtered),
    "total_value": df_filtered['valor'].sum(),
    "unique_beneficiaries": df_filtered['favorecido'].nunique()
}
```

5. **Geração de insights**
```python
insights = [
    f"{top_destination_count} servidores ({pct}%) viajaram para {top_destination}",
    f"Valor médio de diária: R$ {avg_value:.2f}",
    f"Maior beneficiário: {top_beneficiary} (R$ {top_value:.2f})"
]
```

6. **Retorno JSON**
```json
{
  "query": {"topic": "diárias", "filters": {"year": 2022, "month": 8}},
  "data": {
    "records": [...],
    "summary": {...},
    "insights": [...]
  },
  "metadata": {
    "sources": [...],
    "processing_time_ms": 3421
  }
}
```

---

## 6. Protocolo MCP

### 6.1 Implementação

CRISTAL implementa o **MCP Server Protocol** para expor suas ferramentas.

**Transporte**: stdio ou HTTP SSE  
**Formato**: JSON-RPC 2.0

### 6.2 Inicialização

```json
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "clientInfo": {
      "name": "CustomApp",
      "version": "1.0.0"
    }
  }
}
```

**Resposta**:
```json
{
  "protocolVersion": "2024-11-05",
  "capabilities": {
    "tools": {
      "listChanged": true
    }
  },
  "serverInfo": {
    "name": "cristal-mcp-server",
    "version": "1.0.0"
  }
}
```

### 6.3 Listagem de Ferramentas

```json
{
  "jsonrpc": "2.0",
  "method": "tools/list"
}
```

**Resposta**: Lista com `cristal_search`, `cristal_stats`, `cristal_extract_document`, `cristal_analyze`, `cristal_job_status`

### 6.4 Chamada de Ferramenta

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "cristal_search",
    "arguments": {
      "topic": "diárias",
      "year": 2022,
      "month": 8
    }
  }
}
```

---

## 7. Requisitos Técnicos

### 7.1 Requisitos Funcionais

| ID | Requisito | Prioridade |
|----|-----------|------------|
| RF01 | Expor ferramentas via MCP Server | Alta |
| RF02 | Consumir MCP site-research como cliente | Alta |
| RF03 | Extração de dados de PDFs | Alta |
| RF04 | Extração de dados de CSVs | Alta |
| RF05 | Processamento assíncrono (jobs) | Alta |
| RF06 | Agregação e análise de dados | Alta |
| RF07 | Cache de consultas e documentos | Alta |
| RF08 | Geração de insights automáticos | Média |
| RF09 | Retorno de dados em JSON estruturado | Alta |
| RF10 | Download HTTP de documentos com whitelist de domínios | Alta |
| RF11 | Persistência de dados extraídos em Parquet particionado | Alta |
| RF12 | Query Planner com short-circuit via manifest Parquet | Alta |
| RF13 | Parâmetro `force_refresh` em tools de consulta | Média |
| RF14 | Versionamento de extractors registrado no manifest | Média |

### 7.2 Requisitos Não-Funcionais

| ID | Requisito | Métrica |
|----|-----------|---------|
| RNF01 | Tempo de resposta síncrono | < 2s para stats, < 5s para buscas simples |
| RNF02 | Tempo de processamento assíncrono | < 30s para extração de 1 PDF |
| RNF03 | Cache hit rate Redis | > 70% |
| RNF04 | Disponibilidade | 99% uptime |
| RNF05 | Compatibilidade | Linux, macOS (desenvolvimento) |
| RNF06 | Confiabilidade | Taxa de sucesso de extração > 95% |
| RNF07 | Parquet lake hit rate (após período de warm-up) | > 85% para consultas analíticas |
| RNF08 | Tempo de resposta em Parquet hit | < 500ms |

### 7.3 Stack Tecnológico

**Backend**:
- **Linguagem**: Python 3.11+
- **MCP Server**: `mcp` (implementação oficial Python)
- **MCP Client**: `mcp` client para site-research
- **Data Processing**: `pandas`, `numpy`
- **PDF Processing**: `poppler-utils` (pdftotext), `pdfplumber`
- **CSV Processing**: `pandas`, `chardet`
- **Data Lake**: `pyarrow` (Parquet read/write), `duckdb` (queries analíticas)
- **Async HTTP**: `httpx`, `tenacity` (retry com backoff)
- **Task Queue**: `celery` + `redis` (broker/backend)
- **Cache**: `redis`
- **Models**: `pydantic`

**Infraestrutura**:
- **Container**: Docker + Docker Compose
- **Message Broker**: Redis
- **Storage**: 
  - Sistema de arquivos local para blob store (PDFs/CSVs brutos)
  - Sistema de arquivos local para Parquet lake (ou S3-compatível em deploy)
- **Logs**: `structlog` (JSON logs)

---

## 8. Estrutura do Projeto

```
cristal/
├── src/
│   ├── server.py              # Entry point - MCP Server
│   ├── tools/                 # MCP Tools
│   │   ├── __init__.py
│   │   ├── search.py          # cristal_search
│   │   ├── stats.py           # cristal_stats
│   │   ├── extract.py         # cristal_extract_document
│   │   ├── analyze.py         # cristal_analyze
│   │   └── jobs.py            # cristal_job_status
│   ├── planner/               # Query Planner
│   │   ├── __init__.py
│   │   └── query_planner.py
│   ├── mcp_clients/           # Clients para MCP externos
│   │   ├── __init__.py
│   │   └── site_research.py   # Client para site-research
│   ├── fetcher/               # HTTP Fetcher
│   │   ├── __init__.py
│   │   └── http_fetcher.py
│   ├── extractors/            # Data extractors
│   │   ├── __init__.py
│   │   ├── base.py
│   │   ├── pdf.py
│   │   └── csv.py
│   ├── processors/            # Data processing
│   │   ├── __init__.py
│   │   ├── data_processor.py
│   │   └── insights.py
│   ├── storage/               # Camada de persistência
│   │   ├── __init__.py
│   │   ├── parquet_store.py   # Reader/Writer Parquet + manifest
│   │   └── blob_store.py      # PDFs/CSVs brutos
│   ├── workers/               # Celery workers
│   │   ├── __init__.py
│   │   └── tasks.py
│   ├── cache/
│   │   ├── __init__.py
│   │   └── redis_cache.py
│   ├── models/                # Pydantic models
│   │   ├── __init__.py
│   │   ├── query.py
│   │   ├── result.py
│   │   ├── plan.py            # QueryPlan, CoverageReport
│   │   └── job.py
│   └── config.py              # Configurações
├── data/                      # Parquet lake (volume persistente)
│   ├── diarias/
│   ├── contratos/
│   ├── licitacoes/
│   └── _manifest/
├── tests/
│   ├── test_tools.py
│   ├── test_extractors.py
│   ├── test_processors.py
│   ├── test_planner.py
│   └── test_parquet_store.py
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── requirements.txt
├── pyproject.toml
├── README.md
└── SPEC_MIDLEWARE_CRISTAL.md
```

---

## 9. Casos de Uso

### 9.1 UC01: Busca de Diárias por Período

**Ator**: Aplicação customizada (com LLM)

**Pré-condições**: CRISTAL MCP Server ativo, site-research conectado

**Fluxo Principal (cache miss — primeira vez)**:
1. App chama MCP tool: `cristal_search(topic="diárias", year=2022, month=8)`
2. Query Planner consulta manifest Parquet → partição ausente
3. CRISTAL cria job assíncrono e retorna: `{job_id: "abc123", status: "queued"}`
4. Worker processa:
   - Busca no site-research: `search("diárias 2022")` → recebe URLs
   - Encontra 4 páginas, identifica 2 PDFs
   - HTTP Fetcher baixa: `tre-pi-diarias-pagas-agosto-2022.pdf`
   - Extrai com pdftotext → DataFrame com 94 linhas
   - **Grava partição `diarias/year=2022/month=08/data.parquet`**
   - **Atualiza manifest com procedência e extractor_version**
   - Filtra month=8, agrega, gera insights
   - Salva resultado em cache Redis
5. App consulta: `cristal_job_status(job_id="abc123")`
6. CRISTAL retorna: `{status: "completed", result: {...}}`

**Fluxo Alternativo (cache hit — consultas subsequentes)**:
1. App chama: `cristal_search(topic="diárias", year=2022, month=8)`
2. Query Planner consulta manifest → partição presente e frozen
3. CRISTAL lê Parquet diretamente via DuckDB
4. Retorna **síncronamente** em ~100ms com mesmo payload estruturado

**Fluxo Alternativo 3a**: PDF com OCR
- Worker usa Tesseract para extração
- Tempo de processamento aumenta
- Partição Parquet é gravada normalmente (extractor_version reflete uso de OCR)

**Pós-condições**: 
- Dados persistidos em Parquet (permanente)
- Resultado em cache Redis por 1h
- Qualquer consulta futura sobre agosto/2022 atende do Parquet

### 9.2 UC02: Estatísticas do Catálogo

**Ator**: Aplicação customizada

**Fluxo Principal**:
1. App chama: `cristal_stats()`
2. CRISTAL verifica cache
3. Se cache miss:
   - Consulta site-research: `catalog_stats()`
   - Cacheia resultado (TTL 6h)
4. Retorna JSON com estatísticas

**Tempo**: < 1s (cache hit) ou < 3s (cache miss)

### 9.3 UC03: Extração de Documento Específico

**Ator**: Aplicação customizada

**Fluxo Principal**:
1. App chama: `cristal_extract_document(url="https://.../diarias-agosto-2022.pdf")`
2. CRISTAL consulta blob store por `sha256(url)`:
   - **Cache hit**: usa arquivo já baixado, pula para extração/leitura
   - **Cache miss**: HTTP Fetcher baixa o PDF
3. Worker:
   - Extrai dados (ou lê Parquet se já extraído previamente)
   - Retorna DataFrame em JSON
4. App obtém resultado via `cristal_job_status`

**Otimização**: Se a URL já foi processada antes, resultado é servido diretamente do Parquet lake sem nem tocar no blob store.

---

## 10. Roadmap de Desenvolvimento

### Fase 1: Core MCP Server (3 semanas)
- [x] Especificação completa
- [ ] Implementação MCP Server base
  - [ ] Protocolo MCP (stdio/SSE)
  - [ ] Tool registration
  - [ ] Error handling
- [ ] MCP Client para site-research
- [ ] HTTP Fetcher (com whitelist e retry)
- [ ] Extratores básicos:
  - [ ] PDF (pdftotext)
  - [ ] CSV (pandas)
- [ ] Cache em memória
- [ ] Tools mínimos:
  - [ ] `cristal_stats` (síncrono)
  - [ ] `cristal_search` (síncrono simplificado)

### Fase 2: Processamento Assíncrono (2 semanas)
- [ ] Integração Celery + Redis
- [ ] Job queue e workers
- [ ] `cristal_job_status` tool
- [ ] Atualizar `cristal_search` para async
- [ ] `cristal_extract_document` tool
- [ ] Cache com Redis
- [ ] Data Processor:
  - [ ] Filtering
  - [ ] Aggregation
  - [ ] Summarization

### Fase 3: Data Lake Parquet (2 semanas)
- [ ] `ParquetStore` (reader/writer particionado)
- [ ] Manifest de procedência (`_manifest/sources.parquet`)
- [ ] `QueryPlanner` com árvore de decisão hit/parcial/miss
- [ ] Integração DuckDB para queries analíticas
- [ ] Parâmetro `force_refresh` em tools
- [ ] Versionamento de extractors no manifest
- [ ] Política de dados frozen vs live
- [ ] Refresh em background para dados expirados

### Fase 4: Análises Avançadas (2 semanas)
- [ ] `cristal_analyze` tool (usa Parquet diretamente)
- [ ] Geração automática de insights
- [ ] Agregações complexas via DuckDB
- [ ] Suporte a múltiplas fontes
- [ ] Testes unitários (cobertura > 80%)
- [ ] Testes de integração

### Fase 5: Produção (1 semana)
- [ ] Dockerfile otimizado
- [ ] Docker Compose com volumes persistentes (Parquet)
- [ ] Documentação de deployment
- [ ] Logs estruturados (JSON)
- [ ] Otimizações de performance
- [ ] Documentação de uso (README)

---

## 11. Métricas de Sucesso

| Métrica | Target | Medição |
|---------|--------|---------|
| Tempo de resposta síncrono | < 2s | Logs estruturados |
| Tempo de processamento async | < 30s por PDF | Worker logs |
| Taxa de sucesso de extração | > 95% | Logs de extractors |
| Cache hit rate (Redis) | > 70% | Redis metrics |
| Parquet hit rate (após warm-up) | > 85% | Planner metrics |
| Tempo de resposta em Parquet hit | < 500ms | Query Planner logs |
| Redução de fetches externos | > 80% | Contador de HTTP requests |
| Uptime | > 99% | Health checks |
| Taxa de erro de jobs | < 5% | Celery metrics |
| Cobertura de testes | > 80% | pytest-cov |

---

## 12. Considerações de Segurança

Como sistema **interno**, as preocupações de segurança são simplificadas:

### 12.1 Validação de Inputs
- Validação de URLs (whitelist de domínios permitidos)
- Limite de tamanho de documentos (max 50MB)
- Validação de parâmetros (Pydantic models)

### 12.2 Proteção contra DoS
- Limite de jobs simultâneos por cliente
- Timeout de processamento (max 5 min por job)
- Limpeza automática de jobs antigos (> 24h)

### 12.3 Dados Temporários
- Cache com TTL automático
- PDFs baixados removidos após extração
- Jobs completados expiram em 24h

---

## 13. Extensibilidade

### 13.1 Novos MCP Clients

Para adicionar suporte a novos portais/servidores MCP:

```python
# src/mcp_clients/new_portal.py
class NewPortalClient:
    async def search(self, query: str) -> List[dict]:
        """Implementa busca no novo portal"""
        
    async def get_document(self, url: str) -> bytes:
        """Download de documento"""
```

Registrar no config:
```python
MCP_CLIENTS = {
    "site-research": SiteResearchClient,
    "new-portal": NewPortalClient
}
```

### 13.2 Novos Extractors

Para suportar novos formatos:

```python
# src/extractors/excel.py
class ExcelExtractor(BaseExtractor):
    async def extract(self, file_path: str) -> pd.DataFrame:
        """Extrai dados de Excel (XLSX)"""
```

### 13.3 Novas Ferramentas MCP

Para adicionar novas funcionalidades:

```python
# src/tools/compare.py
@mcp_tool
async def cristal_compare(
    category: str,
    period1: str,
    period2: str
) -> dict:
    """Compara dados entre dois períodos"""
```

---

## 14. Referências

- **MCP Specification**: https://modelcontextprotocol.io/
- **MCP Python SDK**: https://github.com/modelcontextprotocol/python-sdk
- **Pandas Documentation**: https://pandas.pydata.org/
- **Celery Documentation**: https://docs.celeryq.dev/
- **Pydantic Documentation**: https://docs.pydantic.dev/
- **Portal de Transparência TRE-PI**: https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas

---

## 15. Glossário

- **MCP**: Model Context Protocol - protocolo para comunicação entre LLMs e fontes de dados
- **MCP Server**: Servidor que expõe ferramentas/recursos via protocolo MCP
- **MCP Client**: Cliente que consome ferramentas de um MCP Server
- **Tool**: Função exposta por um MCP Server que pode ser chamada remotamente
- **Job**: Tarefa assíncrona processada por workers Celery
- **Cache hit rate**: Proporção de consultas atendidas pelo cache
- **OCR**: Optical Character Recognition - reconhecimento óptico de caracteres
- **TTL**: Time To Live - tempo de vida de item em cache
- **DataFrame**: Estrutura de dados tabular do pandas
- **Worker**: Processo que executa tarefas assíncronas (Celery)
- **Parquet**: Formato de arquivo colunar comprimido otimizado para queries analíticas
- **Data Lake**: Repositório de dados estruturados persistidos em formato aberto (Parquet)
- **Partição**: Subdivisão física do Parquet lake por chaves (category/year/month)
- **Manifest**: Arquivo Parquet especial que registra procedência e validade das partições
- **Query Planner**: Componente que decide se uma consulta pode ser servida pelo lake
- **Frozen**: Dado de período fechado, imutável, que nunca precisa ser re-extraído
- **DuckDB**: Engine SQL analítica embarcada que lê Parquet diretamente
- **SSRF**: Server-Side Request Forgery — ataque mitigado pela whitelist do HTTP Fetcher

---

## 16. Anexos

### 16.1 Configuração do Sistema

```yaml
# config.yaml
cristal:
  mcp_server:
    transport: "stdio"  # ou "sse"
    
  mcp_clients:
    site_research:
      transport: "stdio"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-postgres"]
      timeout: 30
      
  http_fetcher:
    allowed_domains:
      - "tre-pi.jus.br"
      - "www.tre-pi.jus.br"
    max_size_mb: 50
    timeout_seconds: 30
    retry_attempts: 3
    retry_backoff: "exponential"
    concurrency: 4
    
  cache:
    backend: "redis"
    host: "localhost"
    port: 6379
    db: 0
    ttl:
      stats: 21600        # 6h
      search: 3600        # 1h
      documents: 7200     # 2h
      
  parquet_lake:
    root_path: "/data/cristal"
    manifest_path: "/data/cristal/_manifest/sources.parquet"
    compression: "zstd"       # ou "snappy"
    engine: "duckdb"          # para queries analíticas
    default_frozen_threshold_days: 90  # dados > 90d viram frozen
    partitioning:
      diarias: ["year", "month"]
      contratos: ["year"]
      licitacoes: ["year", "quarter"]
      
  celery:
    broker: "redis://localhost:6379/1"
    backend: "redis://localhost:6379/2"
    task_timeout: 300   # 5 min
    
  extractors:
    pdf:
      version: "v1"     # registrado no manifest Parquet
      ocr_enabled: true
      ocr_lang: "por"
      max_size_mb: 50
      poppler_path: "/usr/bin"
    csv:
      version: "v1"
      encoding: "utf-8"
      max_size_mb: 10
      
  jobs:
    max_concurrent: 10
    retention_hours: 24
```

### 16.2 Exemplo de Docker Compose

```yaml
version: '3.8'

services:
  cristal:
    build: .
    volumes:
      - ./src:/app/src
      - /tmp/cristal:/tmp/cristal
      - cristal_data:/data/cristal    # Parquet lake persistente
      - cristal_blob:/data/blob       # PDFs/CSVs brutos
    environment:
      - REDIS_URL=redis://redis:6379/0
      - CELERY_BROKER=redis://redis:6379/1
      - PARQUET_ROOT=/data/cristal
    depends_on:
      - redis
    command: python -m src.server
    
  worker:
    build: .
    volumes:
      - ./src:/app/src
      - /tmp/cristal:/tmp/cristal
      - cristal_data:/data/cristal    # workers também escrevem Parquet
      - cristal_blob:/data/blob
    environment:
      - REDIS_URL=redis://redis:6379/0
      - CELERY_BROKER=redis://redis:6379/1
      - PARQUET_ROOT=/data/cristal
    depends_on:
      - redis
    command: celery -A src.workers worker --loglevel=info
    
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  redis_data:
  cristal_data:    # Parquet lake (volume crítico, backup recomendado)
  cristal_blob:    # Blob store (reconstituível se perdido)
```

### 16.3 Exemplo de Resposta MCP Tool

```json
{
  "content": [
    {
      "type": "text",
      "text": "{\"query\": {\"topic\": \"diárias\", \"filters\": {\"year\": 2022, \"month\": 8}}, \"data\": {\"records\": [{\"favorecido\": \"José de Ribamar Portela de Carvalho\", \"cargo\": \"Técnico Judiciário\", \"valor\": 11777.92, \"viagens\": 3}], \"summary\": {\"total_records\": 94, \"total_value\": 153000.0, \"unique_beneficiaries\": 87, \"date_range\": [\"2022-08-01\", \"2022-08-31\"]}, \"insights\": [\"68 servidores (72%) viajaram para Teresina\", \"Valor médio de diária: R$ 1.627,66\", \"Maior beneficiário: José de Ribamar (R$ 11.777,92)\"]}, \"metadata\": {\"sources\": [{\"url\": \"https://.../diarias-agosto-2022.pdf\", \"type\": \"pdf\", \"extracted_at\": \"2026-04-20T21:30:00Z\"}], \"processing_time_ms\": 3421}}"
    }
  ],
  "isError": false
}
```

---

## 17. Resumo Executivo

O **CRISTAL** é um MCP Server especializado que:

1. **Consome** dados de portais de transparência via MCP clients (site-research)
2. **Baixa** documentos referenciados via HTTP Fetcher (com whitelist e retry)
3. **Processa** documentos (PDFs, CSVs) de forma assíncrona
4. **Persiste** dados extraídos em **Parquet data lake** particionado
5. **Responde** consultas analíticas direto do lake quando possível (short-circuit)
6. **Expõe** ferramentas MCP para aplicações customizadas
7. **Retorna** dados estruturados em JSON para consumo por LLMs

### Arquitetura Simplificada
- **Interface**: MCP Server (stdio/SSE)
- **Orquestração**: Query Planner decide origem (Parquet vs fetch externo)
- **Processamento**: Assíncrono (Celery) apenas em cache miss
- **Persistência**:
  - Redis — cache quente (catálogo, jobs)
  - Blob store — arquivos brutos imutáveis
  - Parquet lake — dados extraídos consultáveis
- **Extração**: pdftotext, pandas
- **Análise**: pandas, DuckDB sobre Parquet, insights automáticos

### Ferramentas Principais
1. `cristal_search` — Busca e extração de dados
2. `cristal_stats` — Estatísticas do catálogo
3. `cristal_extract_document` — Extração de documento específico
4. `cristal_analyze` — Análises agregadas (beneficia-se diretamente do Parquet)
5. `cristal_job_status` — Status de processamento

### Diferenciais
- Processamento assíncrono para não bloquear aplicação
- Cache inteligente em múltiplas camadas (Redis + Parquet)
- **Data lake Parquet elimina re-extração** — documentos processados uma vez atendem a qualquer pergunta futura sobre aquele período
- **Redução drástica de carga nos portais externos** após período de warm-up
- Extração automática de PDFs complexos
- Insights gerados automaticamente
- Pronto para integração com LLMs

---

**Documento criado em**: 2026-04-20  
**Última atualização**: 2026-04-21  
**Versão**: 2.1 (Parquet Data Lake)  
**Autor**: Especificação CRISTAL  
**Status**: Pronto para implementação
