#!/bin/bash
# Script de teste automatizado do cristal-chat

set -e

echo "=== Teste do Cristal Chat ==="
echo ""

# Teste 1: Version
echo "Teste 1: Version"
./bin/cristal --version
echo "✓ Passou"
echo ""

# Teste 2: Help
echo "Teste 2: Help"
./bin/cristal --help 2>&1 | grep -q "config"
echo "✓ Passou"
echo ""

# Teste 3: Simulação interativa
echo "Teste 3: Comandos interativos"
echo "Enviando comandos: /help, /tools, /quit"
echo ""

# Criar arquivo de entrada com comandos
cat > /tmp/cristal_test_input.txt <<EOF
/help
/tools
/quit
EOF

# Executar com timeout e entrada simulada
./bin/cristal < /tmp/cristal_test_input.txt 2>/tmp/cristal_test_stderr.txt > /tmp/cristal_test_output.txt || true

# Verificar output
echo "--- Output do Chat ---"
cat /tmp/cristal_test_output.txt
echo ""
echo "--- Stderr (Logs) ---"
cat /tmp/cristal_test_stderr.txt
echo ""

# Validações
if grep -q "Cristal Chat" /tmp/cristal_test_output.txt; then
    echo "✓ Welcome screen apareceu"
else
    echo "✗ Welcome screen não apareceu"
    exit 1
fi

if grep -q "Comandos Disponíveis" /tmp/cristal_test_output.txt; then
    echo "✓ /help funcionou"
else
    echo "✗ /help não funcionou"
    exit 1
fi

if grep -q "Tools Disponíveis" /tmp/cristal_test_output.txt; then
    echo "✓ /tools funcionou"
else
    echo "✗ /tools não funcionou"
    exit 1
fi

if grep -q "Até logo" /tmp/cristal_test_output.txt; then
    echo "✓ /quit funcionou"
else
    echo "✗ /quit não funcionou"
    exit 1
fi

if grep -q "MCP inicializado com sucesso" /tmp/cristal_test_stderr.txt; then
    echo "✓ MCP conectou com sucesso"
else
    echo "✗ MCP não conectou"
    exit 1
fi

# Limpar
rm -f /tmp/cristal_test_input.txt /tmp/cristal_test_output.txt /tmp/cristal_test_stderr.txt

echo ""
echo "=== Todos os testes passaram! ==="
