# Plano de Implementação - Melhorias Críticas de Rastreabilidade

## Objetivo

Garantir que TODA consulta tenha rastreabilidade completa com:
- Busca obrigatória no portal via site-research
- Metadados de fonte em todas as respostas
- URLs de origem sempre visíveis

## Melhorias Críticas (3)

### 🔥 Crítica 1: Forçar Busca Obrigatória no Site-Research

**Problema atual:**
- `research()` aceita dados sem buscar no portal
- Mock retorna dados falsos
- Não há validação se site-research foi usado

**Solução:**
1. Remover mock de `src/clients/site_research.py`
2. Fazer busca no site-research OBRIGATÓRIA
3. Falhar explicitamente se site-research não funcionar
4. Adicionar validação de integração no startup

**Arquivos a modificar:**
- `src/clients/site_research.py` - remover mock, implementar cliente real MCP
- `src/server.py` - adicionar validação obrigatória

**Implementação:**

```python
# src/clients/site_research.py

class SiteResearchClient:
    def __init__(self, url: str = "stdio"):
        self.url = url
        self.session = None
        self.connected = False
    
    async def connect(self):
        """Conecta ao MCP site-research"""
        # TODO: Implementar conexão MCP real
        # Por enquanto, lançar erro se não implementado
        raise NotImplementedError(
            "site-research MCP não está implementado. "
            "Configure o site-research MCP primeiro."
        )
    
    async def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Busca REAL no catálogo via MCP"""
        
        if not self.connected:
            await self.connect()
        
        # Implementação real do cliente MCP
        # Por enquanto, retornar erro claro
        raise NotImplementedError(
            "Busca no portal requer site-research MCP ativo. "
            "Consultas locais não são permitidas para garantir rastreabilidade."
        )
```

```python
# src/server.py - adicionar validação

async def validate_site_research():
    """Valida que site-research está disponível"""
    try:
        # Testar conexão
        test = await site_research.search("teste", limit=1)
        log.info("site_research_validated", status="ok")
        return True
    except NotImplementedError as e:
        log.error("site_research_not_implemented", error=str(e))
        return False
    except Exception as e:
        log.error("site_research_connection_failed", error=str(e))
        return False

# Modificar research()
async def research(query: str, force_fetch: bool = False):
    """Busca completa com validação obrigatória"""
    
    # 1. Verificar cache
    if not force_fetch:
        cached = cache.get_query(query)
        if cached:
            log.info("cache_hit", query=query)
            return _format_response(cached.summary)
    
    # 2. 🔥 BUSCA OBRIGATÓRIA no site-research
    log.info("searching_portal", query=query)
    
    try:
        results = await site_research.search(query, limit=10)
    except NotImplementedError as e:
        return {
            "content": [{
                "type": "text",
                "text": f"""
❌ **ERRO CRÍTICO:** Sistema de busca não disponível

{str(e)}

**Para usar este MCP, você precisa:**
1. Configurar o site-research MCP no .mcp.json
2. Aprovar o site-research no Claude Code
3. Garantir que o índice do portal está atualizado

**Dados locais NÃO são permitidos** para garantir rastreabilidade.
"""
            }]
        }
    except Exception as e:
        log.error("search_failed", error=str(e))
        return {
            "content": [{
                "type": "text",
                "text": f"❌ Erro ao buscar no portal: {str(e)}"
            }]
        }
    
    if not results or len(results) == 0:
        return {
            "content": [{
                "type": "text",
                "text": "⚠️ Nenhum resultado encontrado no portal para essa consulta."
            }]
        }
    
    # 3. Continuar com extração...
    # (resto do código existente)
```

---

### 🔥 Crítica 2: Metadados de Fonte Obrigatórios

**Problema atual:**
- `ExtractedData` não tem campos de rastreabilidade
- Dados extraídos não registram origem
- Sem timestamp de coleta

**Solução:**
1. Atualizar `models.py` com campos obrigatórios
2. Modificar extractors para capturar metadados
3. Garantir que TODO dado tem URL de origem

**Arquivos a modificar:**
- `src/models.py` - adicionar campos de rastreabilidade
- `src/extractors/base.py` - adicionar metadados na interface
- `src/extractors/pdf.py` - capturar metadados
- `src/extractors/spreadsheet.py` - capturar metadados

**Implementação:**

```python
# src/models.py

from pydantic import BaseModel, Field, HttpUrl
from typing import Optional, Dict, Any, List
from datetime import datetime

class SourceMetadata(BaseModel):
    """Metadados obrigatórios de fonte de dados"""
    
    url: HttpUrl                                # 🔥 OBRIGATÓRIO
    source_type: str                            # "pdf", "excel", "csv", "html"
    extracted_at: datetime = Field(default_factory=datetime.now)
    document_title: Optional[str] = None
    document_date: Optional[str] = None
    portal_section: Optional[str] = None
    file_size: Optional[int] = None
    checksum: Optional[str] = None              # MD5 do documento

class ExtractedData(BaseModel):
    """Dados extraídos com rastreabilidade completa"""
    
    metadata: SourceMetadata                    # 🔥 OBRIGATÓRIO
    data: Dict[str, Any]                        # Dados extraídos
    extraction_method: str                      # "pypdf", "polars", etc.
    success: bool = True
    error: Optional[str] = None

class ResearchResponse(BaseModel):
    query: str
    search_timestamp: datetime = Field(default_factory=datetime.now)
    total_sources: int
    sources: List[ExtractedData]                # 🔥 Com metadados completos
    aggregated_data: Optional[Dict[str, Any]] = None
    cache_hit: bool = False
```

```python
# src/extractors/base.py

from abc import ABC, abstractmethod
from typing import Dict, Any
from ..models import SourceMetadata, ExtractedData

class BaseExtractor(ABC):
    
    @abstractmethod
    async def extract(
        self, 
        content: bytes, 
        metadata: SourceMetadata  # 🔥 NOVO: metadados obrigatórios
    ) -> ExtractedData:
        """Extrai dados com metadados de rastreabilidade"""
        pass
    
    @abstractmethod
    def can_handle(self, content_type: str, url: str) -> bool:
        """Verifica se pode processar este tipo"""
        pass
```

```python
# src/extractors/pdf.py

async def extract(self, content: bytes, metadata: SourceMetadata) -> ExtractedData:
    """Extrai texto e valores monetários de PDF"""
    
    try:
        pdf = PdfReader(BytesIO(content))
        
        full_text = ""
        for page in pdf.pages:
            full_text += page.extract_text() + "\n"
        
        valores = self._extract_monetary_values(full_text)
        
        data = {
            "type": "pdf",
            "pages": len(pdf.pages),
            "text_length": len(full_text),
            "text": full_text[:1000],
            "valores_encontrados": len(valores),
            "valores": valores,
            "total": sum(valores) if valores else 0
        }
        
        return ExtractedData(
            metadata=metadata,
            data=data,
            extraction_method="pypdf",
            success=True
        )
        
    except Exception as e:
        return ExtractedData(
            metadata=metadata,
            data={},
            extraction_method="pypdf",
            success=False,
            error=str(e)
        )
```

---

### 🔥 Crítica 3: Formatação com Fontes Visíveis

**Problema atual:**
- `_format_response()` não mostra URLs
- Sem timestamp de consulta
- Sem seção do portal

**Solução:**
1. Reformatar `_format_response()` para incluir todas as fontes
2. Adicionar seção "Fontes de Dados" com metadados completos
3. Incluir timestamp de consulta

**Arquivos a modificar:**
- `src/server.py` - função `_format_response()`

**Implementação:**

```python
# src/server.py

def _format_response(research_response: ResearchResponse) -> Dict:
    """Formata resposta MCP com RASTREABILIDADE COMPLETA"""
    
    text = f"# Resultados: {research_response.query}\n\n"
    
    # Timestamp da consulta
    text += f"🕐 **Consulta realizada em:** {research_response.search_timestamp.strftime('%d/%m/%Y %H:%M:%S')}\n\n"
    
    # Cache hit indicator
    if research_response.cache_hit:
        text += "💾 **Fonte:** Cache local\n\n"
    
    # Dados agregados (se houver)
    if research_response.aggregated_data:
        agg = research_response.aggregated_data
        
        if 'total' in agg:
            text += f"## 💰 Resumo\n\n"
            text += f"- **Total:** R$ {agg['total']:,.2f}\n"
            text += f"- **Registros:** {agg.get('count', 0)}\n"
            text += f"- **Documentos analisados:** {research_response.total_sources}\n\n"
    
    # 🔥 FONTES COM METADADOS COMPLETOS (OBRIGATÓRIO)
    text += f"## 📄 Fontes de Dados ({research_response.total_sources})\n\n"
    
    if research_response.sources and len(research_response.sources) > 0:
        for idx, source in enumerate(research_response.sources, 1):
            meta = source.metadata
            
            text += f"### {idx}. {meta.document_title or 'Documento'}\n\n"
            text += f"- **🔗 URL:** {meta.url}\n"
            text += f"- **📁 Tipo:** {meta.source_type}\n"
            text += f"- **📂 Seção:** {meta.portal_section or 'N/A'}\n"
            text += f"- **🕐 Extraído em:** {meta.extracted_at.strftime('%d/%m/%Y %H:%M:%S')}\n"
            
            if meta.document_date:
                text += f"- **📅 Data do documento:** {meta.document_date}\n"
            
            # Dados extraídos deste documento
            if source.success and source.data:
                if 'total' in source.data:
                    text += f"- **💰 Valor:** R$ {source.data['total']:,.2f}\n"
                if 'rows' in source.data:
                    text += f"- **📊 Registros:** {source.data['rows']}\n"
            else:
                text += f"- **⚠️ Erro:** {source.error}\n"
            
            text += "\n"
    else:
        text += "⚠️ **ALERTA:** Nenhuma fonte identificada!\n\n"
    
    # Rodapé com informações de rastreabilidade
    text += "---\n\n"
    text += "**📌 Rastreabilidade**\n\n"
    text += "Todos os dados apresentados possuem fonte identificável.\n"
    text += "Clique nos links de URL para verificar os documentos originais.\n"
    
    return {"content": [{"type": "text", "text": text}]}
```

---

## Ordem de Implementação

### Passo 1: Models (base)
1. Atualizar `src/models.py` com novos models
2. Testar imports

### Passo 2: Extractors (interface)
1. Modificar `src/extractors/base.py`
2. Atualizar `src/extractors/pdf.py`
3. Atualizar `src/extractors/spreadsheet.py`
4. Testar extractors com metadados

### Passo 3: Server (integração)
1. Atualizar `src/clients/site_research.py`
2. Modificar `src/server.py`:
   - Adicionar validação startup
   - Modificar `research()` com busca obrigatória
   - Atualizar `_format_response()`
3. Testar integração completa

### Passo 4: Testes
1. Testar com erro de site-research (deve falhar explicitamente)
2. Testar formatação com metadados
3. Validar URLs nas respostas

## Critérios de Aceitação

- [ ] TODA consulta tem URL de origem
- [ ] TODA resposta mostra timestamp de extração
- [ ] Sistema FALHA explicitamente se site-research não estiver disponível
- [ ] Cache também mostra fontes originais
- [ ] Extractors capturam metadados completos
- [ ] Formatação inclui seção "Fontes de Dados"

## Testes de Validação

### Teste 1: Erro sem site-research
```python
# Desabilitar site-research
result = await research("teste")
assert "ERRO CRÍTICO" in result
assert "site-research MCP" in result
```

### Teste 2: Metadados completos
```python
result = await research("diárias 2026")
assert "URL:" in result
assert "Extraído em:" in result
assert "Seção:" in result
```

### Teste 3: Formatação com fontes
```python
result = await research("contratos")
assert "📄 Fontes de Dados" in result
assert "http" in result  # Deve ter pelo menos uma URL
```

## Impacto

**Antes:**
- ❌ Dados sem fonte
- ❌ Sem rastreabilidade
- ❌ Aceita dados locais
- ❌ Compliance impossível

**Depois:**
- ✅ Toda resposta tem URL
- ✅ Rastreabilidade completa
- ✅ Busca obrigatória no portal
- ✅ Compliance garantido

## Tempo Estimado

- Passo 1 (Models): 30min
- Passo 2 (Extractors): 1h
- Passo 3 (Server): 1h30min
- Passo 4 (Testes): 30min

**Total: ~3h30min**
