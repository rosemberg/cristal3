#!/bin/bash
# Teste com debug habilitado

echo "=== Teste com Debug ==="
echo ""

# Criar arquivo de entrada
cat > /tmp/cristal_debug_test.txt <<EOF
teste
/quit
EOF

# Executar com debug
./bin/cristal --debug < /tmp/cristal_debug_test.txt 2>&1 | tee /tmp/cristal_debug_full.txt

echo ""
echo "--- Log completo salvo em /tmp/cristal_debug_full.txt ---"
echo ""
echo "--- Últimas 50 linhas ---"
tail -50 /tmp/cristal_debug_full.txt

# Limpar
rm -f /tmp/cristal_debug_test.txt
