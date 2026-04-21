#!/bin/bash

# Script para executar todos os testes da Fase 2

echo "============================================================"
echo "EXECUTANDO TODOS OS TESTES DA FASE 2"
echo "============================================================"
echo ""

cd "$(dirname "$0")/.."
source venv/bin/activate

FAILED=0

echo "▶ Teste 1/4: Extractors Unitários"
python tests/test_extractors.py
if [ $? -ne 0 ]; then
    FAILED=$((FAILED + 1))
    echo "❌ FALHOU"
else
    echo ""
fi

echo "▶ Teste 2/4: Integração Fase 2"
python tests/test_phase2_integration.py
if [ $? -ne 0 ]; then
    FAILED=$((FAILED + 1))
    echo "❌ FALHOU"
else
    echo ""
fi

echo "▶ Teste 3/4: Servidor MCP"
python tests/test_server_get_document.py
if [ $? -ne 0 ]; then
    FAILED=$((FAILED + 1))
    echo "❌ FALHOU"
else
    echo ""
fi

echo "▶ Teste 4/4: End-to-End"
python tests/test_e2e_document_extraction.py
if [ $? -ne 0 ]; then
    FAILED=$((FAILED + 1))
    echo "❌ FALHOU"
else
    echo ""
fi

echo ""
echo "============================================================"
if [ $FAILED -eq 0 ]; then
    echo "✅ TODOS OS TESTES PASSARAM! (4/4)"
    echo "============================================================"
    echo ""
    echo "FASE 2 COMPLETA E VALIDADA!"
    echo ""
    echo "Funcionalidades implementadas:"
    echo "  ✓ BaseExtractor (ABC)"
    echo "  ✓ PDFExtractor com extração de valores monetários"
    echo "  ✓ SpreadsheetExtractor para CSV/Excel"
    echo "  ✓ Tool get_document no servidor MCP"
    echo "  ✓ Cache de documentos extraídos"
    echo "  ✓ Seleção automática de extractor"
    echo ""
    exit 0
else
    echo "❌ $FAILED TESTE(S) FALHARAM"
    echo "============================================================"
    exit 1
fi
