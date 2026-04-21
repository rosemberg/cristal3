# Instalação e Uso do Data Orchestrator MCP no Claude Code

## ✅ Status: INSTALADO E CONFIGURADO

O MCP server foi configurado e está pronto para uso com Claude Code.

## Configuração Atual

### Arquivo: `/Users/rosemberg/projetos-gemini/cristal3/.mcp.json`

```json
{
  "mcpServers": {
    "data-orchestrator": {
      "command": "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp/venv/bin/python",
      "args": ["-m", "src.server"],
      "cwd": "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp"
    }
  }
}
```

## Como Usar

### 1. Reinicie o Claude Code

Para que o MCP server seja reconhecido, você precisa:
- Fechar completamente o Claude Code
- Abrir novamente no diretório `/Users/rosemberg/projetos-gemini/cristal3`

### 2. Aprovar o MCP Server

Na primeira vez que o Claude Code iniciar, ele pode pedir para aprovar o MCP server `data-orchestrator`. **Clique em Aprovar**.

### 3. Usar os Tools

Após aprovação, você terá acesso a 4 tools:

#### **research** - Busca completa com extração automática
```
Busca páginas no catálogo, extrai documentos (PDFs/Excel) e agrega dados.
```

**Exemplo de uso:**
```
Quanto foi gasto em diárias em 2026?
```

#### **get_document** - Extrai documento específico
```
Baixa e extrai dados de um documento específico.
```

**Exemplo de uso:**
```
Extrair dados do PDF: https://exemplo.com/documento.pdf
```

#### **get_cached** - Consulta cache
```
Verifica se uma query já está no cache.
```

**Exemplo de uso:**
```
Verificar cache para: diárias 2026
```

#### **metrics** - Métricas do sistema
```
Retorna estatísticas de uso do servidor MCP.
```

**Exemplo de uso:**
```
Mostrar métricas do data-orchestrator
```

## Verificar Tools Disponíveis

No Claude Code, você pode verificar se o servidor está ativo procurando por ferramentas que começam com `mcp__data_orchestrator__`:

- `mcp__data_orchestrator__research`
- `mcp__data_orchestrator__get_document`
- `mcp__data_orchestrator__get_cached`
- `mcp__data_orchestrator__metrics`

## Estrutura de Diretórios

```
/Users/rosemberg/projetos-gemini/cristal3/
├── .mcp.json                           # Configuração do MCP server
└── data-orchestrator-mcp/              # Servidor MCP
    ├── venv/                            # Ambiente virtual Python
    ├── src/                             # Código fonte
    │   ├── server.py                   # Servidor MCP principal
    │   ├── cache.py                    # Gerenciador de cache
    │   ├── models.py                   # Models Pydantic
    │   ├── metrics.py                  # Sistema de métricas
    │   ├── extractors/                 # Extractors de dados
    │   │   ├── pdf.py                  # Extrator de PDFs
    │   │   └── spreadsheet.py          # Extrator de planilhas
    │   └── clients/                    # Clientes HTTP/MCP
    │       ├── http.py                 # Cliente HTTP
    │       └── site_research.py        # Cliente MCP site-research
    ├── cache/                          # Cache de dados
    │   ├── queries/                    # Queries cacheadas (24h)
    │   ├── documents/                  # Documentos cacheados (7d)
    │   └── extracted/                  # Dados extraídos (Parquet)
    ├── tests/                          # Testes automatizados
    ├── scripts/                        # Scripts utilitários
    │   └── clean_cache.py             # Limpeza de cache
    ├── config.yaml                     # Configuração do servidor
    ├── requirements.txt                # Dependências Python
    └── README.md                       # Documentação completa
```

## Cache

O servidor mantém cache de:
- **Queries**: 24 horas (86400s)
- **Documentos**: 7 dias (604800s)
- **Dados extraídos**: Formato Parquet (compressão eficiente)

### Limpar Cache

```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
python scripts/clean_cache.py --mode all
```

## Logs

O servidor usa logging estruturado (JSON). Para ver logs detalhados, você pode verificar a saída do servidor no Claude Code.

## Troubleshooting

### MCP Server não aparece

1. Verifique se o arquivo `.mcp.json` existe em `/Users/rosemberg/projetos-gemini/cristal3/`
2. Reinicie completamente o Claude Code
3. Verifique se o ambiente virtual está ativado e as dependências instaladas

### Erro ao iniciar servidor

1. Teste manualmente:
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
venv/bin/python -m src.server
```

2. Verifique as dependências:
```bash
venv/bin/pip install -r requirements.txt
```

### Cache crescendo muito

Execute a limpeza periódica:
```bash
python scripts/clean_cache.py --mode expired  # Remove apenas expirados
python scripts/clean_cache.py --mode all      # Remove tudo
```

## Funcionalidades Principais

✅ **Busca inteligente** com detecção automática de necessidade de extração  
✅ **Extração de PDFs** com valores monetários (formato BR: 1.234,56)  
✅ **Extração de planilhas** (CSV, Excel) com estatísticas  
✅ **Cache inteligente** com TTL configurável  
✅ **Agregação automática** de múltiplos documentos  
✅ **Métricas em tempo real** (cache hits, extrações, erros)  
✅ **Logging estruturado** para debugging  

## Próximos Passos

1. **Integrar com MCP site-research real** (atualmente usando mock)
2. **Testar com dados reais** do TRE-PI
3. **Adicionar novos extractors** (Word, imagens com OCR, etc.)
4. **Configurar webhooks** para invalidação de cache
5. **Deploy em produção** (Docker, Kubernetes, etc.)

## Suporte

Para dúvidas ou problemas:
1. Consulte o `README.md` completo
2. Execute os testes: `pytest tests/ -v`
3. Verifique os logs do servidor
4. Revise a documentação técnica em `RELATORIO_FASE*.md`

---

**Implementado em:** 21 de abril de 2026  
**Localização:** `/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp/`  
**Status:** ✅ PRONTO PARA USO
