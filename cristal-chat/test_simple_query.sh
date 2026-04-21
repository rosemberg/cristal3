#!/bin/bash
# Teste de query simples

echo "=== Teste de Query Simples ==="
echo ""

# Criar arquivo de entrada com uma query simples
cat > /tmp/cristal_simple_test.txt <<EOF
teste
/quit
EOF

# Executar com timeout maior
./bin/cristal < /tmp/cristal_simple_test.txt 2>/tmp/cristal_simple_stderr.txt > /tmp/cristal_simple_output.txt || true

# Mostrar output
echo "--- Output ---"
cat /tmp/cristal_simple_output.txt
echo ""

# Verificar
if grep -q "Pesquisando..." /tmp/cristal_simple_output.txt; then
    echo "✓ Query foi enviada"

    # Verificar se houve resposta (não timeout)
    if grep -q "timeout" /tmp/cristal_simple_output.txt; then
        echo "⚠ Query deu timeout (esperado se site-research não estiver acessível)"
    elif grep -qE "(summary|pages|documents|Summary|Pages|Não encontrado)" /tmp/cristal_simple_output.txt; then
        echo "✓ Query teve resposta"
    else
        echo "⚠ Query sem resposta clara"
    fi
else
    echo "✗ Query não foi enviada"
fi

# Limpar
rm -f /tmp/cristal_simple_test.txt /tmp/cristal_simple_output.txt /tmp/cristal_simple_stderr.txt

echo ""
echo "=== Teste concluído ==="
