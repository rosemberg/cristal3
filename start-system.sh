#!/bin/bash
# Script de inicialização completa do sistema Cristal
# Inicia backend e frontend com verificação de pré-requisitos

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Cristal Chat - Inicialização${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Verificar pré-requisitos
echo -e "${YELLOW}[1/5] Verificando pré-requisitos...${NC}"

# Verificar catálogo
if [ ! -f "data/catalog.json" ]; then
    echo -e "${RED}❌ Catálogo não encontrado: data/catalog.json${NC}"
    echo "Execute: cd . && ./bin/site-research build-catalog"
    exit 1
fi
echo -e "${GREEN}✓ Catálogo encontrado${NC}"

# Verificar binário do backend
if [ ! -f "cristal-backend/bin/api" ]; then
    echo -e "${YELLOW}⚠ Backend não compilado, compilando...${NC}"
    cd cristal-backend && go build -o bin/api ./cmd/api
    cd ..
    echo -e "${GREEN}✓ Backend compilado${NC}"
else
    echo -e "${GREEN}✓ Backend já compilado${NC}"
fi

# Verificar site-research-mcp
if [ ! -f "bin/site-research-mcp" ]; then
    echo -e "${RED}❌ site-research-mcp não encontrado: bin/site-research-mcp${NC}"
    echo "Execute: go build -o bin/site-research-mcp ./cmd/site-research-mcp"
    exit 1
fi
echo -e "${GREEN}✓ site-research-mcp encontrado${NC}"

# Verificar data-orchestrator-mcp
if [ ! -d "data-orchestrator-mcp/venv" ] && [ ! -d "data-orchestrator-mcp/.venv" ]; then
    echo -e "${RED}❌ Ambiente virtual Python não encontrado${NC}"
    echo "Execute: cd data-orchestrator-mcp && python3 -m venv venv && source venv/bin/activate && pip install -r requirements.txt"
    exit 1
fi
echo -e "${GREEN}✓ Ambiente Python encontrado${NC}"

# Verificar API Key
if [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo -e "${YELLOW}⚠ Nenhuma API key configurada${NC}"
    echo "Configure uma das opções:"
    echo "  - ANTHROPIC_API_KEY=sk-ant-... (para Claude)"
    echo "  - GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json (para Vertex AI)"
    echo ""
    read -p "Continuar mesmo assim? (s/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Ss]$ ]]; then
        exit 1
    fi
fi

# Verificar frontend
echo -e "${YELLOW}[2/5] Verificando frontend...${NC}"
if [ ! -d "cristal-chat-ui/node_modules" ]; then
    echo -e "${YELLOW}⚠ Dependências do frontend não instaladas, instalando...${NC}"
    cd cristal-chat-ui && npm install
    cd ..
fi
echo -e "${GREEN}✓ Frontend pronto${NC}"

# Criar diretório de logs
mkdir -p logs

# Iniciar backend
echo ""
echo -e "${YELLOW}[3/5] Iniciando backend (porta 8080)...${NC}"
cd cristal-backend
nohup ./bin/api > ../logs/backend.log 2>&1 &
BACKEND_PID=$!
cd ..
echo -e "${GREEN}✓ Backend iniciado (PID: $BACKEND_PID)${NC}"

# Aguardar backend estar pronto
echo -e "${YELLOW}[4/5] Aguardando backend inicializar...${NC}"
sleep 3
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${RED}❌ Backend não respondeu ao health check${NC}"
    echo "Veja os logs em: logs/backend.log"
    kill $BACKEND_PID 2>/dev/null || true
    exit 1
fi
echo -e "${GREEN}✓ Backend respondendo${NC}"

# Iniciar frontend
echo -e "${YELLOW}[5/5] Iniciando frontend (porta 3000)...${NC}"
cd cristal-chat-ui
nohup npm run dev > ../logs/frontend.log 2>&1 &
FRONTEND_PID=$!
cd ..
echo -e "${GREEN}✓ Frontend iniciado (PID: $FRONTEND_PID)${NC}"

# Salvar PIDs para facilitar shutdown
echo "$BACKEND_PID" > logs/backend.pid
echo "$FRONTEND_PID" > logs/frontend.pid

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   ✓ Sistema iniciado com sucesso!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "URLs:"
echo -e "  Frontend: ${BLUE}http://localhost:3000${NC}"
echo -e "  Backend:  ${BLUE}http://localhost:8080${NC}"
echo ""
echo -e "Logs:"
echo -e "  Backend:  ${BLUE}logs/backend.log${NC}"
echo -e "  Frontend: ${BLUE}logs/frontend.log${NC}"
echo ""
echo -e "Para parar o sistema:"
echo -e "  ${BLUE}./stop-system.sh${NC}"
echo ""
echo -e "Acompanhar logs em tempo real:"
echo -e "  ${BLUE}tail -f logs/backend.log${NC}"
echo -e "  ${BLUE}tail -f logs/frontend.log${NC}"
echo ""
