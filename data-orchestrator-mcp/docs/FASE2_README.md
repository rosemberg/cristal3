# Fase 2: Extração de Dados - Guia de Uso

## Visão Geral

A Fase 2 adiciona capacidade de extração de dados de documentos (PDFs e planilhas) ao Data Orchestrator MCP.

## Funcionalidades

### 1. Extractors

Sistema extensível para processar diferentes tipos de documentos.

#### PDFExtractor
```python
from src.extractors.pdf import PDFExtractor

extractor = PDFExtractor()

# Verificar se pode processar
can_handle = extractor.can_handle("application/pdf", "documento.pdf")

# Extrair dados
with open("documento.pdf", "rb") as f:
    result = await extractor.extract(f.read())

# Resultado:
{
    "type": "pdf",
    "pages": 5,
    "text_length": 1500,
    "text": "Primeiros 1000 caracteres...",
    "valores_encontrados": 10,
    "valores": [1200.0, 850.5, ...],
    "total": 5551.25
}
```

#### SpreadsheetExtractor
```python
from src.extractors.spreadsheet import SpreadsheetExtractor

extractor = SpreadsheetExtractor()

# Processar CSV
with open("dados.csv", "rb") as f:
    result = await extractor.extract(f.read())

# Resultado:
{
    "type": "spreadsheet",
    "rows": 100,
    "columns": 5,
    "column_names": ["nome", "valor", "data", ...],
    "total": 12500.50  # Se houver coluna "valor"
}
```

### 2. Tool MCP: get_document

Baixa e extrai documento via URL.

```python
# Via MCP Client
result = await call_tool("get_document", {
    "url": "https://example.com/documento.pdf"
})
```

#### Fluxo de Execução
1. Verifica cache (retorna se encontrado)
2. Faz download via HTTP
3. Detecta tipo do arquivo
4. Seleciona extractor apropriado
5. Extrai dados
6. Cachea resultado
7. Retorna dados extraídos

### 3. Cache de Documentos

Documentos extraídos são automaticamente cacheados.

- **Diretório:** `cache/documents/`
- **TTL padrão:** 7 dias (604800 segundos)
- **Formato:** JSON com metadados

#### Estrutura do Cache
```json
{
  "url": "https://example.com/doc.pdf",
  "extracted_at": "2026-04-21T10:30:00",
  "data": {
    "type": "pdf",
    "pages": 5,
    "valores_encontrados": 10,
    "total": 5551.25
  }
}
```

## Extração de Valores Monetários

### Formato Suportado
Valores no formato brasileiro: **1.234,56**

### Regex Pattern
```python
pattern = r'\b\d{1,3}(?:\.\d{3})*,\d{2}\b'
```

### Exemplos Suportados
- `R$ 100,50` → 100.50
- `R$ 1.000,00` → 1000.00
- `R$ 123.456,78` → 123456.78
- `R$ 999.999.999,99` → 999999999.99

### Conversão
```python
# "1.234,56" → 1234.56
valor_float = float(match.replace('.', '').replace(',', '.'))
```

## Testes

### Executar Todos os Testes
```bash
cd data-orchestrator-mcp
./tests/run_all_phase2_tests.sh
```

### Executar Teste Específico
```bash
source venv/bin/activate

# Testes unitários
python tests/test_extractors.py

# Testes de integração
python tests/test_phase2_integration.py

# Testes do servidor
python tests/test_server_get_document.py

# Testes end-to-end
python tests/test_e2e_document_extraction.py
```

## Exemplos de Uso

### Exemplo 1: Extrair PDF Local
```python
from src.extractors.pdf import PDFExtractor
import asyncio

async def extrair_pdf_local():
    extractor = PDFExtractor()
    
    with open("diarias_fev_2026.pdf", "rb") as f:
        resultado = await extractor.extract(f.read())
    
    print(f"Total de valores: R$ {resultado['total']:,.2f}")
    print(f"Valores encontrados: {resultado['valores_encontrados']}")
    
asyncio.run(extrair_pdf_local())
```

### Exemplo 2: Processar Múltiplos Documentos
```python
from src.cache import CacheManager
from src.extractors.pdf import PDFExtractor
import asyncio

async def processar_multiplos():
    cache = CacheManager("./cache", 3600, 7200)
    extractor = PDFExtractor()
    
    urls = [
        "https://example.com/jan_2026.pdf",
        "https://example.com/fev_2026.pdf",
        "https://example.com/mar_2026.pdf"
    ]
    
    totais = []
    for url in urls:
        # Verificar cache
        cached = cache.get_document(url)
        if cached:
            totais.append(cached['data']['total'])
            continue
        
        # Processar (assumindo que já baixou)
        # resultado = await extractor.extract(content)
        # totais.append(resultado['total'])
    
    print(f"Total geral: R$ {sum(totais):,.2f}")

asyncio.run(processar_multiplos())
```

### Exemplo 3: Via Servidor MCP
```python
import asyncio
from src.server import call_tool

async def usar_via_mcp():
    resultado = await call_tool("get_document", {
        "url": "https://www.tre-pi.jus.br/diarias-fevereiro-2026.pdf"
    })
    
    print(resultado['content'][0]['text'])

asyncio.run(usar_via_mcp())
```

## Adicionar Novo Extractor

### Passo 1: Criar Classe
```python
# src/extractors/novo_tipo.py
from .base import BaseExtractor

class NovoTipoExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return '.novo' in url.lower()
    
    async def extract(self, content: bytes) -> dict:
        # Sua lógica aqui
        return {
            "type": "novo_tipo",
            "data": "..."
        }
```

### Passo 2: Registrar no Servidor
```python
# src/server.py
from .extractors.novo_tipo import NovoTipoExtractor

extractors = [
    PDFExtractor(),
    SpreadsheetExtractor(),
    NovoTipoExtractor()  # Adicionar aqui
]
```

### Passo 3: Criar Testes
```python
# tests/test_novo_tipo.py
def test_novo_tipo_extractor():
    extractor = NovoTipoExtractor()
    assert extractor.can_handle("", "arquivo.novo")
```

## Configuração

### config.yaml
```yaml
cache:
  directory: "./cache"
  ttl_queries: 86400      # 1 dia
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

## Logs

Logs estruturados em JSON:

```json
{
  "event": "document_cache_hit",
  "url": "https://example.com/doc.pdf",
  "timestamp": "2026-04-21T10:30:00"
}
```

```json
{
  "event": "fetching_document",
  "url": "https://example.com/doc.pdf",
  "timestamp": "2026-04-21T10:30:01"
}
```

## Troubleshooting

### Erro: Extractor não encontrado
```
Tipo de documento não suportado
```

**Solução:** Verificar se há extractor para o tipo de arquivo.

### Erro: Valores não extraídos
```python
valores_encontrados: 0
```

**Solução:** Verificar se valores estão no formato brasileiro (1.234,56).

### Erro: Cache não funcionando
```
Cache não encontrado
```

**Solução:** Verificar permissões no diretório cache/.

## Performance

### Benchmarks
- **PDF pequeno (1MB):** ~100ms extração
- **PDF grande (10MB):** ~500ms extração
- **CSV (1000 linhas):** ~50ms leitura
- **Excel (1000 linhas):** ~150ms leitura

### Otimizações
- Cache evita reprocessamento
- Polars usa Arrow (eficiente)
- Extração paralela (futuro)

## Limitações Atuais

1. **Content-Type simplificado:** Sempre assume "application/pdf"
2. **Sem OCR:** Apenas texto extraível
3. **Sem tabelas estruturadas:** Apenas valores monetários
4. **Sem paralelização:** Um documento por vez

## Roadmap (Futuro)

- [ ] Detecção automática de content-type via HTTP headers
- [ ] Extração de tabelas estruturadas com tabula/camelot
- [ ] OCR para PDFs escaneados
- [ ] Processamento paralelo de múltiplos documentos
- [ ] Suporte a mais formatos (JSON, XML, etc)
- [ ] Validação de dados extraídos
- [ ] Métricas de qualidade da extração

## Referências

- [pypdf Documentation](https://pypdf.readthedocs.io/)
- [Polars Documentation](https://pola-rs.github.io/polars/)
- [PLANO_DATA_CRISTAL.md](../PLANO_DATA_CRISTAL.md)
- [Relatório Fase 2](../RELATORIO_FASE2.md)

## Contribuindo

Para adicionar funcionalidades:

1. Criar branch: `git checkout -b feature/novo-extractor`
2. Implementar código
3. Adicionar testes (manter cobertura >90%)
4. Executar `./tests/run_all_phase2_tests.sh`
5. Criar PR

## Licença

Mesmo que o projeto principal.

---

**Última atualização:** 21 de Abril de 2026  
**Versão:** 1.0.0
