#!/bin/bash

# Script para executar testes da Fase 3
# Testa integração completa com extração automática

echo "==================================="
echo "Testes da Fase 3 - Integração Completa"
echo "==================================="
echo ""

cd "$(dirname "$0")/.."

# Usar pytest do ambiente virtual
PYTEST=".venv/bin/pytest"

echo "1. Testando detecção de necessidade de dados..."
$PYTEST tests/test_phase3_integration.py::TestNeedsDetailedData -v
echo ""

echo "2. Testando agregação de resultados..."
$PYTEST tests/test_phase3_integration.py::TestAggregateResults -v
echo ""

echo "3. Testando formatação de resposta..."
$PYTEST tests/test_phase3_integration.py::TestFormatResponse -v
echo ""

echo "4. Testando cache Parquet..."
$PYTEST tests/test_phase3_integration.py::TestCacheParquet -v
echo ""

echo "5. Testando cliente site-research mock..."
$PYTEST tests/test_phase3_integration.py::TestMockSiteResearchClient -v
echo ""

echo "6. Testando integração end-to-end..."
$PYTEST tests/test_phase3_integration.py::TestEndToEndIntegration -v
echo ""

echo "==================================="
echo "7. Executando TODOS os testes da Fase 3..."
echo "==================================="
$PYTEST tests/test_phase3_integration.py -v --tb=short

exit_code=$?

if [ $exit_code -eq 0 ]; then
    echo ""
    echo "✅ Todos os testes da Fase 3 passaram!"
    echo ""
    echo "Próximos passos:"
    echo "  - Testar manualmente via MCP"
    echo "  - Validar com dados reais"
    echo "  - Avançar para Fase 4 (Refinamento)"
else
    echo ""
    echo "❌ Alguns testes falharam. Verifique os erros acima."
fi

exit $exit_code
