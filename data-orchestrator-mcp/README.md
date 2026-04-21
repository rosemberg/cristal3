# Data Orchestrator MCP

Servidor MCP (Model Context Protocol) que integra busca em catálogo com extração automática de dados de documentos, fornecendo respostas completas com dados estruturados.

## Visão Geral

O Data Orchestrator MCP combina:
- Busca inteligente em catálogo via site-research
- Extração automática de dados de PDFs, CSVs e Excel
- Cache inteligente com TTL configurável
- Métricas e observabilidade
- Logging estruturado

## Instalação

### Pré-requisitos

- Python 3.10+
- pip

### Passos

1. Clone o repositório:
```bash
git clone <repo-url>
cd data-orchestrator-mcp
```

2. Crie um ambiente virtual:
```bash
python -m venv .venv
source .venv/bin/activate  # Linux/Mac
# ou
.venv\Scripts\activate  # Windows
```

3. Instale as dependências:
```bash
pip install -r requirements.txt
```

## Configuração

### 1. Arquivo de ambiente (opcional)

Copie `.env.example` para `.env` se necessário:
```bash
cp .env.example .env
```

### 2. Arquivo config.yaml

O arquivo `config.yaml` contém configurações principais:

```yaml
cache:
  directory: "./cache"
  ttl_queries: 3600      # 1 hora
  ttl_documents: 86400   # 24 horas

mcp:
  site_research_url: "stdio"

http:
  timeout: 30
  max_retries: 3
```

## Uso

### Iniciar o servidor

```bash
python -m src.server
```

O servidor estará disponível via stdio para conexão MCP.

### Conectar via Claude Code

Adicione em `~/.claude/settings.json` ou `.claude/settings.json`:

```json
{
  "mcpServers": {
    "data-orchestrator": {
      "command": "python",
      "args": ["-m", "src.server"],
      "cwd": "/caminho/absoluto/para/data-orchestrator-mcp",
      "env": {}
    }
  }
}
```

### Tools disponíveis

#### 1. research
Busca completa com extração automática de dados.

```json
{
  "query": "quanto foi gasto com diárias em 2026",
  "force_fetch": false
}
```

**Funcionalidades:**
- Verifica cache primeiro (exceto se force_fetch=true)
- Busca no catálogo via site-research
- Detecta necessidade de extração (keywords: quanto, valor, total, gasto, custo, despesa)
- Baixa e extrai automaticamente até 3 documentos
- Agrega resultados com totais
- Cacheia resultado para futuras consultas

#### 2. get_cached
Retorna dados do cache se disponíveis.

```json
{
  "query": "gastos com diárias"
}
```

#### 3. get_document
Baixa e extrai documento específico.

```json
{
  "url": "https://example.com/relatorio.pdf"
}
```

**Suporta:**
- PDFs com extração de valores monetários (formato brasileiro: 1.234,56)
- CSVs e planilhas Excel
- Cache de documentos com TTL configurável

#### 4. metrics
Retorna estatísticas e métricas do sistema.

```json
{}
```

**Informações:**
- Uptime
- Cache hits/misses e taxa de acerto
- Extrações (sucesso/falhas)
- Páginas processadas
- Valores monetários extraídos
- Dados transferidos
- Total de erros

## Estrutura do Projeto

```
data-orchestrator-mcp/
├── src/
│   ├── extractors/          # Extratores de dados
│   │   ├── base.py          # Classe base para extractors
│   │   ├── pdf.py           # Extrator de PDFs
│   │   └── spreadsheet.py   # Extrator de CSV/Excel
│   ├── clients/             # Clientes externos
│   │   ├── site_research.py # Cliente MCP site-research
│   │   └── http.py          # Cliente HTTP com retry
│   ├── cache.py             # Gerenciamento de cache
│   ├── metrics.py           # Métricas e observabilidade
│   ├── models.py            # Modelos de dados (Pydantic)
│   └── server.py            # Servidor MCP principal
├── tests/                   # Testes automatizados
│   ├── test_cache.py        # Testes do CacheManager
│   ├── test_extractors.py  # Testes dos extractors
│   └── test_integration.py # Testes de integração
├── scripts/
│   └── clean_cache.py       # Script de limpeza de cache
├── cache/                   # Diretório de cache
│   ├── queries/             # Cache de queries
│   ├── documents/           # Cache de documentos
│   └── extracted/           # Dados extraídos em Parquet
├── docs/                    # Documentação
├── config.yaml              # Configuração principal
├── requirements.txt         # Dependências Python
└── README.md               # Este arquivo
```

## Funcionalidades

### Extração de Dados

- **PDFs**: Extrai texto completo e valores monetários no formato brasileiro
- **CSVs/Excel**: Processa planilhas com detecção automática de colunas
- **Valores monetários**: Regex otimizado para formato BR (1.234,56)
- **Metadata**: Extrai páginas, tamanho, timestamps

### Cache Inteligente

- **Queries**: TTL configurável (padrão 1h)
- **Documentos**: TTL configurável (padrão 24h)
- **Parquet**: Armazenamento eficiente para dados tabulares
- **Hash-based**: Identificação única via MD5

### Métricas e Observabilidade

- Cache hit/miss rate
- Extrações sucessos/falhas
- Páginas processadas
- Valores monetários extraídos
- Dados transferidos (bytes/MB)
- Uptime tracking
- Thread-safe counters

### Logging Estruturado

- Formato JSON via structlog
- Timestamps ISO
- Stack traces em erros
- Contexto rico (query, url, errors)

## Desenvolvimento

### Executar todos os testes

```bash
pytest tests/ -v
```

### Executar testes específicos

```bash
# Testes de cache
pytest tests/test_cache.py -v

# Testes de extractors
pytest tests/test_extractors.py -v

# Testes de integração
pytest tests/test_integration.py -v
```

### Limpeza de cache

#### Limpar todo o cache
```bash
python scripts/clean_cache.py --mode all
```

#### Limpar apenas expirados
```bash
python scripts/clean_cache.py --mode expired
```

#### Simular limpeza (dry-run)
```bash
python scripts/clean_cache.py --mode all --dry-run
```

#### Ver estatísticas do cache
```bash
python scripts/clean_cache.py --stats
```

### Adicionar novo extractor

1. Crie arquivo em `src/extractors/`:
```python
from .base import BaseExtractor

class MyExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return 'mytype' in content_type.lower()

    async def extract(self, content: bytes) -> dict:
        # Implementar extração
        return {"type": "mytype", "data": ...}
```

2. Registre em `src/server.py`:
```python
from .extractors.my_extractor import MyExtractor

extractors = [
    PDFExtractor(),
    SpreadsheetExtractor(),
    MyExtractor()  # Adicionar aqui
]
```

## Troubleshooting

### Servidor não inicia

- Verifique se o ambiente virtual está ativado
- Verifique se todas as dependências foram instaladas
- Verifique logs de erro no stdout

### Cache não funciona

- Verifique permissões do diretório `cache/`
- Verifique TTL em `config.yaml`
- Use `--stats` para ver estado do cache

### Extração falha

- Verifique se o PDF não está criptografado
- Verifique logs estruturados para detalhes do erro
- Use ferramenta `metrics` para ver taxa de falhas

## Roadmap

- [x] Fase 0: Setup e estrutura base
- [x] Fase 1: Integração com site-research
- [x] Fase 2: Extração de PDFs e cache
- [x] Fase 3: Agregação e formatação
- [x] Fase 4: Testes, métricas e produção

## Contribuindo

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## Licença

MIT License - veja LICENSE para detalhes.

## Contato

- Projeto: Data Orchestrator MCP
- Versão: 1.0.0 (Produção)
- Status: Todas as fases completas
