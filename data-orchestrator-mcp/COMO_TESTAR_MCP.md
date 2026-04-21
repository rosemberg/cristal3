# Como Testar o Data Orchestrator MCP

Este guia explica como testar o servidor MCP Data Orchestrator após a implementação da Fase 3.

## Pré-requisitos

```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp

# Se ainda não criou o ambiente virtual
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

## Método 1: Testes Automatizados

### Executar todos os testes da Fase 3

```bash
./tests/run_phase3_tests.sh
```

**Saída esperada**:
```
===================================
Testes da Fase 3 - Integração Completa
===================================

...

============================== 19 passed in 1.18s ==============================

✅ Todos os testes da Fase 3 passaram!
```

### Executar testes específicos

```bash
# Testar detecção de extração
.venv/bin/pytest tests/test_phase3_integration.py::TestNeedsDetailedData -v

# Testar agregação
.venv/bin/pytest tests/test_phase3_integration.py::TestAggregateResults -v

# Testar cache Parquet
.venv/bin/pytest tests/test_phase3_integration.py::TestCacheParquet -v

# Testes end-to-end
.venv/bin/pytest tests/test_phase3_integration.py::TestEndToEndIntegration -v
```

## Método 2: Testes Manuais (Linha de Comando)

### Script de teste manual

```bash
.venv/bin/python tests/test_manual_phase3.py
```

**O que este script faz**:
1. Testa queries que devem acionar extração (com keywords)
2. Testa queries que NÃO devem acionar extração
3. Valida comportamento de cache

**Saída esperada**:
```
================================================================================
TESTES MANUAIS - FASE 3: INTEGRAÇÃO COMPLETA
================================================================================

[info] searching query='quanto foi gasto em diárias em 2026'
[info] extraction_required query='quanto foi gasto...' results_count=3

...

✅ TODOS OS TESTES MANUAIS CONCLUÍDOS
```

## Método 3: Testar via MCP (Servidor Rodando)

### Passo 1: Iniciar o servidor

```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
.venv/bin/python -m src.server
```

O servidor ficará rodando e aguardando conexões MCP via stdio.

### Passo 2: Configurar Claude Code

Adicione ao arquivo de configuração MCP do Claude Code:

**~/.claude/mcp.json** (ou equivalente):
```json
{
  "mcpServers": {
    "data-orchestrator": {
      "command": "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp/.venv/bin/python",
      "args": ["-m", "src.server"],
      "cwd": "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp"
    }
  }
}
```

### Passo 3: Testar no Claude Code

Abra o Claude Code e use os comandos MCP:

#### Teste 1: Query com extração automática

```python
mcp__data_orchestrator__research(query="quanto foi gasto em diárias em 2026")
```

**Resultado esperado**:
```markdown
# Resultados: quanto foi gasto em diárias em 2026

**Paginas encontradas:** 3
**Documentos extraidos:** 0

**Fontes:**
- https://www.tre-pi.jus.br/transparencia/diarias-fevereiro-2026
- https://www.tre-pi.jus.br/transparencia/diarias-janeiro-2026
- https://www.tre-pi.jus.br/transparencia/despesas-2026
```

**Observação**: Como estamos usando mock, os documentos retornam 404 e `Documentos extraidos` será 0. Com dados reais, mostraria o total.

#### Teste 2: Query sem extração

```python
mcp__data_orchestrator__research(query="diárias e passagens")
```

**Resultado esperado**:
```markdown
# Resultados: diárias e passagens

**Paginas encontradas:** 3
**Documentos extraidos:** 0

**Fontes:**
- https://www.tre-pi.jus.br/transparencia/diarias-fevereiro-2026
- https://www.tre-pi.jus.br/transparencia/diarias-janeiro-2026
- https://www.tre-pi.jus.br/transparencia/despesas-2026
```

**Observação**: Não há tentativa de extração porque query não tem keywords.

#### Teste 3: Verificar cache

```python
# Primeira chamada
mcp__data_orchestrator__research(query="quanto foi gasto em diárias")

# Segunda chamada (deve usar cache)
mcp__data_orchestrator__research(query="quanto foi gasto em diárias")

# Consultar cache diretamente
mcp__data_orchestrator__get_cached(query="quanto foi gasto em diárias")
```

#### Teste 4: Force fetch (ignorar cache)

```python
mcp__data_orchestrator__research(
    query="quanto foi gasto em diárias",
    force_fetch=True
)
```

#### Teste 5: Extrair documento específico

```python
mcp__data_orchestrator__get_document(
    url="https://www.tre-pi.jus.br/docs/diarias-jan-2026.pdf"
)
```

**Observação**: URL mock retornará 404, mas com URL real funcionaria.

## Método 4: Testar com Python REPL

### Iniciar Python REPL

```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
.venv/bin/python
```

### Testar funções diretamente

```python
import asyncio
from src.server import research, _needs_detailed_data, _aggregate_results

# Testar detecção
print(_needs_detailed_data("quanto foi gasto", []))  # True
print(_needs_detailed_data("diárias", []))  # False

# Testar research
async def test():
    result = await research("quanto foi gasto em diárias")
    print(result["content"][0]["text"])

asyncio.run(test())
```

## Verificar Logs

Durante a execução, o servidor gera logs estruturados JSON:

```bash
# Logs aparecem no stdout do servidor
[2026-04-21T12:32:39.529231Z] [info] searching query='quanto foi gasto'
[2026-04-21T12:32:39.529547Z] [info] extraction_required results_count=3
[2026-04-21T12:32:39.529560Z] [info] auto_extracting url=https://...
```

### Filtrar logs específicos

```bash
# Em outro terminal, enquanto servidor roda
tail -f /path/to/logs | grep "extraction_required"
```

## Verificar Cache

### Listar arquivos em cache

```bash
# Queries em cache
ls -lh cache/queries/
# Exemplo: 5f4dcc3b5aa765d61d8327deb882cf99.json

# Documentos em cache
ls -lh cache/documents/

# Dados extraídos (Parquet)
ls -lh cache/extracted/
```

### Inspecionar cache de query

```bash
cat cache/queries/*.json | python -m json.tool
```

**Exemplo de saída**:
```json
{
  "query": "quanto foi gasto em diárias",
  "timestamp": "2026-04-21T12:32:39.694000",
  "ttl": 86400,
  "summary": {
    "query": "quanto foi gasto em diárias",
    "found_pages": 3,
    "extracted_documents": 0,
    "sources": [...]
  },
  "data_file": null
}
```

## Limpar Cache (para testar de novo)

```bash
rm -rf cache/queries/*
rm -rf cache/documents/*
rm -rf cache/extracted/*
```

Ou use o script (a ser implementado na Fase 4):
```bash
python scripts/clean_cache.py --mode all
```

## Troubleshooting

### Erro: "No module named 'src'"

**Solução**: Execute de dentro do diretório do projeto:
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
.venv/bin/python -m src.server
```

### Erro: "No module named pytest"

**Solução**: Instale as dependências:
```bash
.venv/bin/pip install -r requirements.txt
```

### Erro: "404 Not Found" nos logs

**Esperado**: Estamos usando mock com URLs que não existem. Com integração real, os documentos serão baixados corretamente.

### Cache não está sendo usado

**Verificar**:
1. Chame a mesma query duas vezes
2. Observe os logs: segunda chamada deve mostrar `[info] cache_hit`
3. Se não aparecer, verifique TTL no `config.yaml`

### Extração não está sendo acionada

**Verificar**:
1. Query tem keywords? (quanto, valor, total, gasto, custo, despesa)
2. Veja logs: deve aparecer `[info] extraction_required`
3. Se não aparecer, função `_needs_detailed_data()` não detectou keyword

## Testes com Dados Reais (Futuro)

Quando integrar com dados reais:

### 1. Atualizar mock do site-research

Substituir URLs mock por URLs reais do TRE-PI:
```python
# Em src/clients/site_research.py
"documents": [
    "https://www.tre-pi.jus.br/real-document.pdf"  # URL real
]
```

### 2. Testar extração de PDF real

```python
result = await research("quanto foi gasto em diárias em fevereiro 2026")
print(result["content"][0]["text"])

# Agora deve mostrar:
# **Total:** R$ 71,598.21
# **Registros:** 41
# **Documentos extraidos:** 1
```

### 3. Validar valores extraídos

Baixe o PDF manualmente e compare totais.

## Próximos Passos

Após validar testes:

1. ✅ Fase 3 completa e testada
2. 🔄 Integrar com MCP site-research real (Fase 4)
3. 🔄 Adicionar PDFs reais para teste
4. 🔄 Implementar métricas e observabilidade

---

**Dúvidas?** Consulte:
- `FASE3_COMPLETA.md` - Documentação técnica
- `RELATORIO_FASE3.md` - Relatório executivo
- `PLANO_DATA_CRISTAL.md` - Plano geral
