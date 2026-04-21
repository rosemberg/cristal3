#!/bin/bash
# Teste de query real

set -e

echo "=== Teste de Query Real ==="
echo ""

# Criar arquivo de entrada com uma query
cat > /tmp/cristal_query_test.txt <<EOF
licitações
/quit
EOF

# Executar com timeout
./bin/cristal < /tmp/cristal_query_test.txt 2>/tmp/cristal_query_stderr.txt > /tmp/cristal_query_output.txt || true

# Mostrar output
echo "--- Output ---"
cat /tmp/cristal_query_output.txt
echo ""
echo "--- Logs (últimas 20 linhas) ---"
tail -20 /tmp/cristal_query_stderr.txt
echo ""

# Verificar se a query foi executada
if grep -q "Pesquisando..." /tmp/cristal_query_output.txt; then
    echo "✓ Query foi enviada"
else
    echo "✗ Query não foi enviada"
fi

# Verificar se houve algum resultado (pode ser cache miss ou resultado real)
if grep -qE "(summary|pages|documents|error|cache|não encontrado)" /tmp/cristal_query_output.txt; then
    echo "✓ Houve resposta da query"
else
    echo "⚠ Sem resposta clara da query (verificar logs)"
fi

# Limpar
rm -f /tmp/cristal_query_test.txt /tmp/cristal_query_output.txt /tmp/cristal_query_stderr.txt

echo ""
echo "=== Teste de query concluído ==="
