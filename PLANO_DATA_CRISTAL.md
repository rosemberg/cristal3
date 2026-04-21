# Plano de Desenvolvimento: Data Orchestrator MCP

## Visão Geral

Implementação em **4 fases incrementais**, cada uma entregando funcionalidade testável.

**Estratégia:** Construir camada por camada, validando a cada fase antes de avançar.

---

## Fase 0: Setup e Estrutura Base

**Objetivo:** Criar estrutura do projeto e ambiente de desenvolvimento.

**Duração estimada:** 1-2 horas

### Tarefas

#### 0.1 Estrutura de Diretórios
```bash
mkdir -p data-orchestrator-mcp/src/{extractors,clients}
mkdir -p data-orchestrator-mcp/cache/{queries,documents,extracted}
mkdir -p data-orchestrator-mcp/tests
mkdir -p data-orchestrator-mcp/scripts
```

**Arquivos criados:**
```
data-orchestrator-mcp/
├── src/
│   ├── __init__.py
│   ├── server.py
│   ├── cache.py
│   ├── models.py
│   ├── extractors/
│   │   ├── __init__.py
│   │   ├── base.py
│   │   ├── pdf.py
│   │   └── spreadsheet.py
│   └── clients/
│       ├── __init__.py
│       ├── site_research.py
│       └── http.py
├── tests/
│   ├── __init__.py
│   ├── test_cache.py
│   ├── test_extractors.py
│   └── test_integration.py
├── scripts/
│   └── clean_cache.py
├── cache/          # .gitignore
├── config.yaml
├── requirements.txt
├── .env.example
├── .gitignore
└── README.md
```

#### 0.2 Dependências

**requirements.txt:**
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

# Dev/Test
pytest>=8.0.0
pytest-asyncio>=0.23.0
```

#### 0.3 Configuração Base

**config.yaml:**
```yaml
mcp:
  site_research_url: "stdio"  # ou URL se remoto

cache:
  directory: "./cache"
  ttl_queries: 86400
  ttl_documents: 604800

extraction:
  pdf:
    engine: "pypdf"
  spreadsheet:
    engine: "polars"

http:
  timeout: 30
  max_retries: 3

logging:
  level: "INFO"
  format: "json"
```

**.env.example:**
```bash
# MCP Site Research
MCP_SITE_RESEARCH_URL=stdio

# Cache
CACHE_DIR=./cache
CACHE_TTL_QUERIES=86400
CACHE_TTL_DOCUMENTS=604800

# Logging
LOG_LEVEL=INFO
```

#### 0.4 .gitignore

```
cache/
*.pyc
__pycache__/
.env
.pytest_cache/
*.log
.DS_Store
```

### Critérios de Aceitação

- ✅ Estrutura de diretórios criada
- ✅ requirements.txt instalável (`pip install -r requirements.txt`)
- ✅ Arquivos de configuração criados
- ✅ .gitignore configurado

### Teste

```bash
cd data-orchestrator-mcp
pip install -r requirements.txt
python -c "import mcp, httpx, pypdf, polars; print('OK')"
```

---

## Fase 1: MVP - Servidor MCP Básico

**Objetivo:** Servidor MCP funcional com 1 tool básico, sem extração ainda.

**Duração estimada:** 3-4 horas

### Tarefas

#### 1.1 Models (Pydantic)

**src/models.py:**
```python
from pydantic import BaseModel, Field
from typing import Optional, Dict, Any, List
from datetime import datetime

class CacheEntry(BaseModel):
    query: str
    timestamp: datetime
    ttl: int
    summary: Dict[str, Any]
    data_file: Optional[str] = None

class DocumentMetadata(BaseModel):
    url: str
    type: str
    extracted_at: datetime
    data: Dict[str, Any]

class ResearchResponse(BaseModel):
    query: str
    summary: Dict[str, Any]
    sources: List[Dict[str, str]]
    data: Optional[Dict[str, Any]] = None
```

#### 1.2 Cache Manager

**src/cache.py:**
```python
import json
import hashlib
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional
from .models import CacheEntry

class CacheManager:
    def __init__(self, cache_dir: str, ttl_queries: int, ttl_documents: int):
        self.cache_dir = Path(cache_dir)
        self.queries_dir = self.cache_dir / "queries"
        self.documents_dir = self.cache_dir / "documents"
        self.extracted_dir = self.cache_dir / "extracted"
        
        # Criar diretórios
        self.queries_dir.mkdir(parents=True, exist_ok=True)
        self.documents_dir.mkdir(parents=True, exist_ok=True)
        self.extracted_dir.mkdir(parents=True, exist_ok=True)
        
        self.ttl_queries = ttl_queries
        self.ttl_documents = ttl_documents
    
    def _hash_key(self, key: str) -> str:
        return hashlib.md5(key.encode()).hexdigest()
    
    def get_query(self, query: str) -> Optional[CacheEntry]:
        cache_file = self.queries_dir / f"{self._hash_key(query)}.json"
        if not cache_file.exists():
            return None
        
        data = json.loads(cache_file.read_text())
        entry = CacheEntry(**data)
        
        # Verificar TTL
        if datetime.now() - entry.timestamp > timedelta(seconds=self.ttl_queries):
            cache_file.unlink()  # Remove expirado
            return None
        
        return entry
    
    def set_query(self, query: str, summary: dict, data_file: Optional[str] = None):
        entry = CacheEntry(
            query=query,
            timestamp=datetime.now(),
            ttl=self.ttl_queries,
            summary=summary,
            data_file=data_file
        )
        
        cache_file = self.queries_dir / f"{self._hash_key(query)}.json"
        cache_file.write_text(entry.model_dump_json(indent=2))
    
    def get_document(self, url: str) -> Optional[dict]:
        cache_file = self.documents_dir / f"{self._hash_key(url)}.json"
        if not cache_file.exists():
            return None
        
        data = json.loads(cache_file.read_text())
        
        # Verificar TTL
        extracted_at = datetime.fromisoformat(data['extracted_at'])
        if datetime.now() - extracted_at > timedelta(seconds=self.ttl_documents):
            cache_file.unlink()
            return None
        
        return data
    
    def set_document(self, url: str, data: dict):
        metadata = {
            "url": url,
            "extracted_at": datetime.now().isoformat(),
            "data": data
        }
        
        cache_file = self.documents_dir / f"{self._hash_key(url)}.json"
        cache_file.write_text(json.dumps(metadata, indent=2))
```

#### 1.3 Cliente MCP Site Research

**src/clients/site_research.py:**
```python
import httpx
from typing import List, Dict, Any

class SiteResearchClient:
    def __init__(self, url: str = "stdio"):
        self.url = url
        # TODO: Implementar cliente MCP real
        # Por enquanto, mock para desenvolvimento
    
    async def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Busca no catálogo via MCP site-research"""
        # Mock para desenvolvimento
        return [
            {
                "title": "Diárias e Passagens",
                "url": "https://www.tre-pi.jus.br/.../diarias-e-passagens",
                "section": "Recursos Humanos"
            }
        ]
    
    async def inspect_page(self, url: str) -> Dict[str, Any]:
        """Inspeciona página específica"""
        # Mock para desenvolvimento
        return {
            "url": url,
            "title": "Diárias",
            "documents": []
        }
```

#### 1.4 Cliente HTTP

**src/clients/http.py:**
```python
import httpx
from typing import Optional

class HTTPClient:
    def __init__(self, timeout: int = 30, max_retries: int = 3):
        self.timeout = timeout
        self.max_retries = max_retries
        self.client = httpx.AsyncClient(timeout=timeout)
    
    async def fetch(self, url: str) -> Optional[bytes]:
        """Faz download de URL"""
        for attempt in range(self.max_retries):
            try:
                response = await self.client.get(url)
                response.raise_for_status()
                return response.content
            except httpx.HTTPError as e:
                if attempt == self.max_retries - 1:
                    raise
                continue
        return None
    
    async def close(self):
        await self.client.aclose()
```

#### 1.5 Servidor MCP (MVP)

**src/server.py:**
```python
import asyncio
from mcp.server import Server
from mcp.server.stdio import stdio_server
import structlog
from pathlib import Path
import yaml

from .cache import CacheManager
from .clients.site_research import SiteResearchClient
from .clients.http import HTTPClient

# Setup logging
log = structlog.get_logger()

# Carregar config
config_path = Path(__file__).parent.parent / "config.yaml"
config = yaml.safe_load(config_path.read_text())

# Inicializar componentes
cache = CacheManager(
    cache_dir=config['cache']['directory'],
    ttl_queries=config['cache']['ttl_queries'],
    ttl_documents=config['cache']['ttl_documents']
)

site_research = SiteResearchClient(config['mcp']['site_research_url'])
http_client = HTTPClient(
    timeout=config['http']['timeout'],
    max_retries=config['http']['max_retries']
)

# Criar servidor MCP
server = Server("data-orchestrator")

@server.list_tools()
async def list_tools():
    return [
        {
            "name": "research",
            "description": "Busca completa com dados extraídos",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Consulta de busca"},
                    "force_fetch": {"type": "boolean", "default": False}
                },
                "required": ["query"]
            }
        },
        {
            "name": "get_cached",
            "description": "Retorna dados do cache se disponíveis",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string"}
                },
                "required": ["query"]
            }
        }
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    log.info("tool_called", tool=name, args=arguments)
    
    if name == "research":
        return await research(
            query=arguments["query"],
            force_fetch=arguments.get("force_fetch", False)
        )
    
    elif name == "get_cached":
        return await get_cached(query=arguments["query"])
    
    else:
        raise ValueError(f"Unknown tool: {name}")

async def research(query: str, force_fetch: bool = False):
    """Busca completa com dados extraídos"""
    
    # 1. Verificar cache
    if not force_fetch:
        cached = cache.get_query(query)
        if cached:
            log.info("cache_hit", query=query)
            return {
                "content": [
                    {
                        "type": "text",
                        "text": f"# Resultado (cache)\n\n{cached.summary}"
                    }
                ]
            }
    
    # 2. Buscar no site-research
    log.info("searching", query=query)
    results = await site_research.search(query)
    
    # 3. Preparar resposta básica (sem extração ainda)
    summary = {
        "query": query,
        "found": len(results),
        "sources": results
    }
    
    # 4. Cachear
    cache.set_query(query, summary)
    
    # 5. Retornar
    return {
        "content": [
            {
                "type": "text",
                "text": f"# Resultados para: {query}\n\nEncontrados: {len(results)} páginas"
            }
        ]
    }

async def get_cached(query: str):
    """Retorna dados do cache"""
    cached = cache.get_query(query)
    
    if not cached:
        return {
            "content": [
                {"type": "text", "text": "Cache não encontrado"}
            ]
        }
    
    return {
        "content": [
            {
                "type": "text",
                "text": f"# Cache: {query}\n\n{cached.summary}"
            }
        ]
    }

async def main():
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())

if __name__ == "__main__":
    asyncio.run(main())
```

### Critérios de Aceitação

- ✅ Servidor MCP inicia sem erros
- ✅ Tool `research` retorna resultados (mesmo que mock)
- ✅ Tool `get_cached` verifica cache
- ✅ Cache salva e recupera dados
- ✅ Logs estruturados funcionando

### Teste

```bash
# Iniciar servidor
cd data-orchestrator-mcp
python -m src.server

# Em outro terminal (Claude Code)
# Testar chamada ao MCP
mcp__data_orchestrator__research(query="diárias 2026")
```

---

## Fase 2: Extração de Dados

**Objetivo:** Implementar extração real de PDFs e CSV/Excel.

**Duração estimada:** 4-6 horas

### Tarefas

#### 2.1 Extractor Base

**src/extractors/base.py:**
```python
from abc import ABC, abstractmethod
from typing import Dict, Any

class BaseExtractor(ABC):
    @abstractmethod
    async def extract(self, content: bytes) -> Dict[str, Any]:
        """Extrai dados do conteúdo"""
        pass
    
    @abstractmethod
    def can_handle(self, content_type: str, url: str) -> bool:
        """Verifica se pode processar este tipo"""
        pass
```

#### 2.2 PDF Extractor

**src/extractors/pdf.py:**
```python
import re
from typing import Dict, Any, List
from pypdf import PdfReader
from io import BytesIO
from .base import BaseExtractor

class PDFExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return 'pdf' in content_type.lower() or url.endswith('.pdf')
    
    async def extract(self, content: bytes) -> Dict[str, Any]:
        """Extrai texto e valores monetários de PDF"""
        pdf = PdfReader(BytesIO(content))
        
        full_text = ""
        for page in pdf.pages:
            full_text += page.extract_text() + "\n"
        
        # Extrair valores monetários (formato brasileiro: 1.234,56)
        valores = self._extract_monetary_values(full_text)
        
        return {
            "type": "pdf",
            "pages": len(pdf.pages),
            "text_length": len(full_text),
            "text": full_text[:1000],  # Primeiros 1000 chars
            "valores_encontrados": len(valores),
            "valores": valores,
            "total": sum(valores) if valores else 0
        }
    
    def _extract_monetary_values(self, text: str) -> List[float]:
        """Extrai valores monetários do texto"""
        # Pattern: 1.234,56 ou 234,56
        pattern = r'\b\d{1,3}(?:\.\d{3})*,\d{2}\b'
        matches = re.findall(pattern, text)
        
        valores = []
        for match in matches:
            # Converter formato brasileiro para float
            valor_float = float(match.replace('.', '').replace(',', '.'))
            valores.append(valor_float)
        
        return valores
```

#### 2.3 Spreadsheet Extractor

**src/extractors/spreadsheet.py:**
```python
import polars as pl
from typing import Dict, Any
from io import BytesIO
from .base import BaseExtractor

class SpreadsheetExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return any(ext in url.lower() for ext in ['.csv', '.xlsx', '.xls'])
    
    async def extract(self, content: bytes) -> Dict[str, Any]:
        """Extrai dados de CSV ou Excel"""
        
        # Detectar tipo
        if b'PK' in content[:4]:  # Excel (ZIP format)
            df = pl.read_excel(BytesIO(content))
        else:  # CSV
            df = pl.read_csv(BytesIO(content))
        
        # Calcular estatísticas básicas
        summary = {
            "type": "spreadsheet",
            "rows": df.height,
            "columns": df.width,
            "column_names": df.columns
        }
        
        # Se houver coluna com valores, tentar somar
        valor_cols = [col for col in df.columns if 'valor' in col.lower()]
        if valor_cols:
            total = df[valor_cols[0]].sum()
            summary["total"] = float(total)
        
        return summary
```

#### 2.4 Integrar Extractors no Servidor

**Atualizar src/server.py:**

```python
from .extractors.pdf import PDFExtractor
from .extractors.spreadsheet import SpreadsheetExtractor

# Inicializar extractors
extractors = [
    PDFExtractor(),
    SpreadsheetExtractor()
]

def get_extractor(content_type: str, url: str):
    for extractor in extractors:
        if extractor.can_handle(content_type, url):
            return extractor
    return None

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    # ... código existente ...
    
    if name == "get_document":
        return await get_document(url=arguments["url"])

async def get_document(url: str):
    """Baixa e extrai documento específico"""
    
    # 1. Verificar cache
    cached = cache.get_document(url)
    if cached:
        log.info("document_cache_hit", url=url)
        return {"content": [{"type": "text", "text": str(cached)}]}
    
    # 2. Fazer download
    log.info("fetching_document", url=url)
    content = await http_client.fetch(url)
    
    if not content:
        return {"content": [{"type": "text", "text": "Erro ao baixar documento"}]}
    
    # 3. Detectar tipo e extrair
    content_type = "application/pdf"  # Simplificado, melhorar depois
    extractor = get_extractor(content_type, url)
    
    if not extractor:
        return {"content": [{"type": "text", "text": "Tipo de documento não suportado"}]}
    
    extracted = await extractor.extract(content)
    
    # 4. Cachear
    cache.set_document(url, extracted)
    
    # 5. Retornar
    return {
        "content": [
            {
                "type": "text",
                "text": f"# Documento extraído\n\n{extracted}"
            }
        ]
    }
```

#### 2.5 Atualizar `list_tools()` com get_document

```python
@server.list_tools()
async def list_tools():
    return [
        # ... tools existentes ...
        {
            "name": "get_document",
            "description": "Baixa e extrai dados de documento específico",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "url": {"type": "string", "description": "URL do documento"}
                },
                "required": ["url"]
            }
        }
    ]
```

### Critérios de Aceitação

- ✅ PDFExtractor extrai texto e valores monetários
- ✅ SpreadsheetExtractor lê CSV e Excel
- ✅ Tool `get_document` baixa e extrai PDFs
- ✅ Dados extraídos são cacheados
- ✅ Total de valores é calculado corretamente

### Teste

```python
# Testar extração de PDF real
mcp__data_orchestrator__get_document(
    url="https://www.tre-pi.jus.br/.../diarias-fevereiro-2026-pdf"
)

# Deve retornar:
# - Total: R$ 71.598,21
# - Valores encontrados: 41
```

---

## Fase 3: Integração Completa

**Objetivo:** Integrar extração automática no fluxo `research()`.

**Duração estimada:** 3-4 horas

### Tarefas

#### 3.1 Atualizar `research()` com Extração Automática

**src/server.py:**

```python
async def research(query: str, force_fetch: bool = False):
    """Busca completa com dados extraídos"""
    
    # 1. Verificar cache
    if not force_fetch:
        cached = cache.get_query(query)
        if cached:
            log.info("cache_hit", query=query)
            return _format_response(cached.summary)
    
    # 2. Buscar no site-research
    log.info("searching", query=query)
    results = await site_research.search(query, limit=10)
    
    # 3. Detectar se precisa de dados detalhados
    needs_extraction = _needs_detailed_data(query, results)
    
    extracted_data = []
    if needs_extraction:
        # 4. Extrair documentos automaticamente
        for result in results[:3]:  # Limitar a 3 primeiros
            if 'documents' in result:
                for doc_url in result['documents'][:2]:  # Max 2 docs por página
                    log.info("auto_extracting", url=doc_url)
                    
                    # Verificar cache de documento
                    doc_cached = cache.get_document(doc_url)
                    if doc_cached:
                        extracted_data.append(doc_cached)
                        continue
                    
                    # Fetch e extração
                    content = await http_client.fetch(doc_url)
                    if content:
                        extractor = get_extractor("application/pdf", doc_url)
                        if extractor:
                            extracted = await extractor.extract(content)
                            cache.set_document(doc_url, extracted)
                            extracted_data.append(extracted)
    
    # 5. Agregar resultados
    summary = _aggregate_results(query, results, extracted_data)
    
    # 6. Cachear
    cache.set_query(query, summary)
    
    # 7. Retornar
    return _format_response(summary)

def _needs_detailed_data(query: str, results: List[Dict]) -> bool:
    """Detecta se query precisa de dados extraídos"""
    keywords = ['quanto', 'valor', 'total', 'gasto', 'custo', 'despesa']
    return any(kw in query.lower() for kw in keywords)

def _aggregate_results(query: str, results: List[Dict], extracted: List[Dict]) -> Dict:
    """Agrega dados extraídos"""
    
    summary = {
        "query": query,
        "found_pages": len(results),
        "extracted_documents": len(extracted),
        "sources": [r.get('url') for r in results]
    }
    
    # Se houver dados extraídos com valores
    if extracted:
        totals = [e.get('total', 0) for e in extracted if 'total' in e]
        if totals:
            summary["total"] = sum(totals)
            summary["count"] = sum(e.get('valores_encontrados', 0) for e in extracted)
            summary["details"] = extracted
    
    return summary

def _format_response(summary: Dict) -> Dict:
    """Formata resposta para MCP"""
    text = f"# Resultados: {summary['query']}\n\n"
    
    if 'total' in summary:
        text += f"**Total:** R$ {summary['total']:,.2f}\n"
        text += f"**Registros:** {summary.get('count', 0)}\n\n"
    
    text += f"**Páginas encontradas:** {summary.get('found_pages', 0)}\n"
    text += f"**Documentos extraídos:** {summary.get('extracted_documents', 0)}\n\n"
    
    if 'sources' in summary:
        text += "**Fontes:**\n"
        for src in summary['sources'][:5]:
            text += f"- {src}\n"
    
    return {"content": [{"type": "text", "text": text}]}
```

#### 3.2 Melhorar Cliente Site Research

**src/clients/site_research.py:**

Implementar integração real com MCP site-research usando cliente MCP:

```python
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

class SiteResearchClient:
    def __init__(self, url: str = "stdio"):
        self.url = url
        self.session = None
    
    async def connect(self):
        """Conecta ao MCP site-research"""
        server_params = StdioServerParameters(
            command="node",
            args=["path/to/site-research-mcp/build/index.js"],
            env=None
        )
        
        self.stdio, self.write = await stdio_client(server_params)
        self.session = ClientSession(self.stdio, self.write)
        await self.session.initialize()
    
    async def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Busca no catálogo via MCP"""
        if not self.session:
            await self.connect()
        
        result = await self.session.call_tool(
            "search",
            arguments={"query": query, "limit": limit}
        )
        
        # Parse resultado
        # TODO: adaptar ao formato real do site-research
        return self._parse_search_results(result)
```

#### 3.3 Armazenamento Parquet

**src/cache.py - adicionar:**

```python
import polars as pl

class CacheManager:
    # ... métodos existentes ...
    
    def save_parquet(self, data: List[Dict], filename: str):
        """Salva dados tabulares em Parquet"""
        df = pl.DataFrame(data)
        filepath = self.extracted_dir / f"{filename}.parquet"
        df.write_parquet(filepath)
        return str(filepath)
    
    def load_parquet(self, filename: str) -> pl.DataFrame:
        """Carrega dados de Parquet"""
        filepath = self.extracted_dir / f"{filename}.parquet"
        return pl.read_parquet(filepath)
```

### Critérios de Aceitação

- ✅ `research()` detecta quando precisa de extração
- ✅ Extração automática de documentos funciona
- ✅ Agregação de valores está correta
- ✅ Cliente real MCP site-research integrado
- ✅ Dados tabulares salvos em Parquet

### Teste End-to-End

```python
# Via Claude Code
mcp__data_orchestrator__research(query="quanto foi gasto em diárias em 2026")

# Deve retornar:
# Total: R$ 108.975,14
# Registros: 63
# Documentos extraídos: 2
# Fontes: [URLs...]
```

---

## Fase 4: Refinamento e Produção

**Objetivo:** Melhorias, testes, documentação e preparação para produção.

**Duração estimada:** 4-6 horas

### Tarefas

#### 4.1 Testes Automatizados

**tests/test_cache.py:**
```python
import pytest
from src.cache import CacheManager
import tempfile

@pytest.fixture
def cache():
    with tempfile.TemporaryDirectory() as tmpdir:
        yield CacheManager(tmpdir, ttl_queries=3600, ttl_documents=7200)

def test_cache_query_set_get(cache):
    cache.set_query("test", {"result": "ok"})
    result = cache.get_query("test")
    assert result is not None
    assert result.summary["result"] == "ok"

def test_cache_query_ttl_expired(cache):
    cache.ttl_queries = -1  # Já expirado
    cache.set_query("test", {"result": "ok"})
    result = cache.get_query("test")
    assert result is None
```

**tests/test_extractors.py:**
```python
import pytest
from src.extractors.pdf import PDFExtractor

@pytest.mark.asyncio
async def test_pdf_extract_monetary_values():
    extractor = PDFExtractor()
    
    # PDF mock com valores
    # TODO: usar PDF de teste real
    
    result = await extractor.extract(pdf_content)
    assert "valores" in result
    assert result["total"] > 0
```

**tests/test_integration.py:**
```python
import pytest
from src.server import research

@pytest.mark.asyncio
async def test_research_with_cache():
    # Primeira chamada
    result1 = await research("test query")
    
    # Segunda chamada (deve usar cache)
    result2 = await research("test query")
    
    assert result1 == result2
```

#### 4.2 Script de Limpeza de Cache

**scripts/clean_cache.py:**
```python
#!/usr/bin/env python3
import argparse
from pathlib import Path
import shutil

def clean_cache(cache_dir: str, mode: str = "expired"):
    cache_path = Path(cache_dir)
    
    if mode == "all":
        # Limpar tudo
        for subdir in ['queries', 'documents', 'extracted']:
            path = cache_path / subdir
            if path.exists():
                shutil.rmtree(path)
                path.mkdir()
        print(f"Cache limpo: {cache_dir}")
    
    elif mode == "expired":
        # TODO: implementar limpeza seletiva baseada em TTL
        print("Limpeza de expirados não implementada ainda")

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--cache-dir", default="./cache")
    parser.add_argument("--mode", choices=["all", "expired"], default="expired")
    
    args = parser.parse_args()
    clean_cache(args.cache_dir, args.mode)
```

#### 4.3 Logging Estruturado

**Melhorar logging em src/server.py:**

```python
import structlog

structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer()
    ]
)

log = structlog.get_logger()

# Usar em todo código:
log.info("cache_hit", query=query, duration_ms=123)
log.error("extraction_failed", url=url, error=str(e))
```

#### 4.4 README.md

**Documentação completa:**

```markdown
# Data Orchestrator MCP

MCP server que fornece dados completos através de busca em catálogo + extração automática.

## Instalação

```bash
pip install -r requirements.txt
```

## Configuração

Copiar `.env.example` para `.env` e ajustar.

## Uso

### Iniciar servidor

```bash
python -m src.server
```

### Conectar via Claude Code

Adicionar em MCP settings:

```json
{
  "mcpServers": {
    "data-orchestrator": {
      "command": "python",
      "args": ["-m", "src.server"],
      "cwd": "/path/to/data-orchestrator-mcp"
    }
  }
}
```

### Tools disponíveis

- `research(query)` - Busca completa com extração
- `get_document(url)` - Extrai documento específico
- `get_cached(query)` - Consulta cache

## Desenvolvimento

### Testes

```bash
pytest tests/
```

### Limpeza de cache

```bash
python scripts/clean_cache.py --mode all
```
```

#### 4.5 Tratamento de Erros

**Adicionar tratamento robusto:**

```python
async def research(query: str, force_fetch: bool = False):
    try:
        # ... lógica existente ...
    except Exception as e:
        log.error("research_failed", query=query, error=str(e))
        return {
            "content": [
                {
                    "type": "text",
                    "text": f"Erro ao processar busca: {str(e)}"
                }
            ]
        }
```

#### 4.6 Métricas e Observabilidade

**src/metrics.py:**

```python
from collections import defaultdict
from datetime import datetime

class Metrics:
    def __init__(self):
        self.stats = defaultdict(int)
        self.start_time = datetime.now()
    
    def increment(self, metric: str):
        self.stats[metric] += 1
    
    def get_summary(self):
        uptime = (datetime.now() - self.start_time).total_seconds()
        return {
            "uptime_seconds": uptime,
            "cache_hits": self.stats["cache_hit"],
            "cache_misses": self.stats["cache_miss"],
            "extractions": self.stats["extraction"],
            "errors": self.stats["error"]
        }

metrics = Metrics()
```

**Adicionar tool para métricas:**

```python
@server.call_tool()
async def call_tool(name: str, arguments: dict):
    # ... outras tools ...
    
    if name == "metrics":
        return {
            "content": [
                {
                    "type": "text",
                    "text": str(metrics.get_summary())
                }
            ]
        }
```

### Critérios de Aceitação

- ✅ Testes automatizados passando
- ✅ Script de limpeza funcionando
- ✅ Logging estruturado configurado
- ✅ README.md completo
- ✅ Tratamento de erros robusto
- ✅ Métricas básicas implementadas

### Teste Final

```bash
# Executar suite de testes
pytest tests/ -v

# Testar end-to-end via Claude Code
mcp__data_orchestrator__research(query="gastos diárias 2026")
mcp__data_orchestrator__metrics()

# Verificar logs
tail -f logs/orchestrator.log

# Verificar cache
ls -lh cache/queries/
ls -lh cache/extracted/
```

---

## Fase 5: Deploy e Monitoramento (Opcional)

**Objetivo:** Preparar para ambiente de produção.

### Tarefas

#### 5.1 Docker

**Dockerfile:**
```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "-m", "src.server"]
```

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  data-orchestrator:
    build: .
    volumes:
      - ./cache:/app/cache
      - ./config.yaml:/app/config.yaml
    environment:
      - LOG_LEVEL=INFO
```

#### 5.2 Health Check

```python
@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "health":
        return {
            "content": [
                {
                    "type": "text",
                    "text": "OK"
                }
            ]
        }
```

#### 5.3 Documentação de Deploy

**docs/DEPLOY.md** com instruções de produção.

---

## Checklist Final

### Funcionalidades Core
- [ ] Servidor MCP funcional
- [ ] Tool `research` com extração automática
- [ ] Tool `get_document` funcionando
- [ ] Tool `get_cached` consultando cache
- [ ] Cache com TTL configurável
- [ ] Extração de PDFs
- [ ] Extração de CSV/Excel
- [ ] Armazenamento Parquet
- [ ] Cliente MCP site-research integrado

### Qualidade
- [ ] Testes automatizados (>70% coverage)
- [ ] Logging estruturado
- [ ] Tratamento de erros
- [ ] Documentação completa
- [ ] Script de limpeza de cache

### Produção
- [ ] Configuração via .env
- [ ] Docker (opcional)
- [ ] Health check
- [ ] Métricas básicas

---

## Próximos Passos

Após conclusão das 4 fases principais:

1. **Integrar com múltiplos portais** (TRE-RJ, CNJ, etc.)
2. **Melhorar heurísticas de extração** (ML para detecção de tabelas)
3. **Cache distribuído** (Redis) se necessário
4. **Rate limiting** para APIs externas
5. **Webhooks** para invalidação de cache

---

## Estimativa Total

- **Fase 0:** 1-2h
- **Fase 1:** 3-4h
- **Fase 2:** 4-6h
- **Fase 3:** 3-4h
- **Fase 4:** 4-6h

**Total: 15-22 horas** de desenvolvimento

**Recomendação:** Fazer uma fase por dia, validando antes de avançar.
