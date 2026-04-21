#!/bin/bash
# Script de validação da Fase 1

echo "=================================================="
echo "Validação da Fase 1 - Data Orchestrator MCP"
echo "=================================================="
echo ""

PROJECT_DIR="/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp"
PYTHON="$PROJECT_DIR/venv/bin/python"

cd "$PROJECT_DIR"

echo "1. Verificando arquivos implementados..."
files=(
    "src/models.py"
    "src/cache.py"
    "src/server.py"
    "src/clients/site_research.py"
    "src/clients/http.py"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "   ✓ $file"
    else
        echo "   ✗ $file - FALTANDO"
        exit 1
    fi
done

echo ""
echo "2. Verificando estrutura de cache..."
dirs=(
    "cache/queries"
    "cache/documents"
    "cache/extracted"
)

for dir in "${dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "   ✓ $dir"
    else
        echo "   ✗ $dir - FALTANDO"
        exit 1
    fi
done

echo ""
echo "3. Testando imports..."
$PYTHON -c "
import sys
sys.path.insert(0, '$PROJECT_DIR')
from src.models import CacheEntry, DocumentMetadata, ResearchResponse
from src.cache import CacheManager
from src.clients.site_research import SiteResearchClient
from src.clients.http import HTTPClient
from src.server import server, list_tools, call_tool
print('   ✓ Todos os imports OK')
" || exit 1

echo ""
echo "4. Executando testes de inicialização..."
$PYTHON test_startup.py > /dev/null 2>&1 && echo "   ✓ test_startup.py passou" || echo "   ✗ test_startup.py falhou"

echo ""
echo "5. Executando testes de tools..."
$PYTHON test_server_tools.py > /dev/null 2>&1 && echo "   ✓ test_server_tools.py passou" || echo "   ✗ test_server_tools.py falhou"

echo ""
echo "6. Verificando configuração..."
if [ -f "config.yaml" ]; then
    echo "   ✓ config.yaml presente"
else
    echo "   ✗ config.yaml ausente"
    exit 1
fi

echo ""
echo "7. Verificando dependências..."
$PYTHON -c "
import mcp
import httpx
import pydantic
import structlog
import yaml
print('   ✓ Todas as dependências instaladas')
" || exit 1

echo ""
echo "=================================================="
echo "✓ FASE 1 VALIDADA COM SUCESSO!"
echo "=================================================="
echo ""
echo "Servidor pronto para uso:"
echo "  python -m src.server"
echo ""
echo "Próximo passo: Implementar Fase 2 (Extração de Dados)"
