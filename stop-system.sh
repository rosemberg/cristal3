#!/bin/bash
# Script para parar o sistema Cristal

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Parando sistema Cristal...${NC}"

# Parar backend
if [ -f "logs/backend.pid" ]; then
    BACKEND_PID=$(cat logs/backend.pid)
    if kill -0 $BACKEND_PID 2>/dev/null; then
        kill $BACKEND_PID
        echo -e "${GREEN}✓ Backend parado (PID: $BACKEND_PID)${NC}"
    fi
    rm -f logs/backend.pid
fi

# Parar frontend
if [ -f "logs/frontend.pid" ]; then
    FRONTEND_PID=$(cat logs/frontend.pid)
    if kill -0 $FRONTEND_PID 2>/dev/null; then
        kill $FRONTEND_PID
        echo -e "${GREEN}✓ Frontend parado (PID: $FRONTEND_PID)${NC}"
    fi
    rm -f logs/frontend.pid
fi

# Limpar processos órfãos na porta 8080 e 3000
lsof -ti:8080 | xargs kill -9 2>/dev/null || true
lsof -ti:3000 | xargs kill -9 2>/dev/null || true

echo -e "${GREEN}✓ Sistema parado${NC}"
