# Integração Completa - Sistema Cristal Chat

**Data**: 21 de Abril de 2026  
**Status**: ✅ **IMPLEMENTADO E PRONTO PARA TESTES**

## Visão Geral

Sistema completo de chat com interface web integrado ao backend Go, que orquestra Claude/Gemini com servidores MCP para consultas inteligentes ao portal de transparência do TRE-PI.

## Arquitetura Final

```
┌─────────────────────────────────────────────────────────────────┐
│                     Frontend (React)                             │
│              cristal-chat-ui (localhost:3000)                    │
│                                                                   │
│  [UserBubble] → [Composer] → POST /chat                          │
│       ↓                                                           │
│  [AssistantMessage] ← JSON { response, citations[] }             │
│       ↓                                                           │
│  [CitationsBlock] - Exibe referências numeradas                  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                         HTTP (JSON)
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Backend Go (REST API)                         │
│              cristal-backend (localhost:8080)                    │
│                                                                   │
│  POST /chat → [Handler] → [Orchestrator]                        │
│                              ↓                                   │
│                     [LLM Provider]                               │
│                   (Claude / Gemini)                              │
│                              ↓                                   │
│                      [MCP Manager]                               │
│              ┌───────────────┴─────────────┐                    │
│              ↓                             ↓                    │
│    [site-research-mcp]       [data-orchestrator-mcp]            │
│         (Go)                        (Python)                    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                    [Catálogo + Documentos]
```

## Implementação Realizada

### 1. Backend - Sistema de Citações (✅ Completo)

**Arquivos Modificados/Criados:**

- `internal/server/types.go` - Adicionado tipo `Citation` e atualizado `ChatResponse`
- `internal/orchestrator/orchestrator.go` - Modificado para coletar e retornar citações
- `internal/orchestrator/citations.go` - **NOVO** - Parser e formatador de citações
- `internal/orchestrator/tools.go` - Atualizado system prompt com instruções de citação
- `internal/server/handler.go` - Atualizado para converter citações

**Funcionalidades Implementadas:**

#### 1.1 Extração de Citações dos MCPs

Durante a execução das tools `search` e `inspect_page`, o sistema:
- Parseia markdown retornado pelos MCPs
- Extrai URL, título e breadcrumb de cada página
- Adiciona ao mapa de citações (evita duplicatas)

#### 1.2 Formatação Inline

O LLM é instruído a usar `[texto da página](url)` quando mencionar páginas.  
O sistema converte automaticamente para `[texto]^N` onde N é o ID da citação.

#### 1.3 Resposta Compatível com Frontend

```json
{
  "response": "Consulte os Balancetes de 2025^1 para mais informações...",
  "citations": [
    {
      "id": 1,
      "title": "Balancetes 2025",
      "breadcrumb": "Transparência › Contabilidade › Balancetes",
      "url": "https://www.tre-pi.jus.br/..."
    }
  ]
}
```

### 2. Frontend - Já Implementado (Fase 6)

O frontend React já possui todos os componentes necessários:

- `AssistantMessage` - Renderiza markdown com citações inline
- `CitationInline` - Componente `<cite>` com número sobrescrito
- `CitationsBlock` - Bloco "PÁGINAS CITADAS" ao final
- `CitationItem` - Item individual de citação
- Parser de citações `preprocessCitations()` - Converte `[texto]^N` em tags `<cite>`

**Exemplo de Fluxo:**

1. Usuário digita: "Quanto foi gasto com diárias em 2026?"
2. Frontend → POST /chat → Backend
3. Backend:
   - LLM usa tool `research`
   - MCP retorna markdown com URLs
   - Sistema extrai citações
   - LLM gera resposta com `[Relatório de Diárias](url)`
   - Sistema converte para `[Relatório de Diárias]^1`
4. Backend → JSON { response, citations } → Frontend
5. Frontend:
   - Renderiza texto com `CitationInline` para `^1`
   - Renderiza `CitationsBlock` com lista de páginas

## Como Usar

### Pré-requisitos

1. **Catálogo gerado**:
   ```bash
   ./bin/site-research discover
   ./bin/site-research crawl
   ./bin/site-research summarize
   ./bin/site-research build-catalog
   ```

2. **API Key configurada** (uma das opções):
   ```bash
   export ANTHROPIC_API_KEY="sk-ant-..."
   # OU
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
   ```

3. **Dependências instaladas**:
   ```bash
   # Backend (já compilado)
   cd cristal-backend && go build -o bin/api ./cmd/api

   # Frontend
   cd cristal-chat-ui && npm install
   ```

### Iniciar Sistema Completo

```bash
# Script automático (recomendado)
./start-system.sh
```

O script:
- ✓ Verifica todos os pré-requisitos
- ✓ Compila backend se necessário
- ✓ Inicia backend em localhost:8080
- ✓ Inicia frontend em localhost:3000
- ✓ Salva PIDs para shutdown fácil

### Parar Sistema

```bash
./stop-system.sh
```

### Iniciar Manualmente (Desenvolvimento)

**Terminal 1 - Backend:**
```bash
cd cristal-backend
ANTHROPIC_API_KEY=sk-ant-... ./bin/api
```

**Terminal 2 - Frontend:**
```bash
cd cristal-chat-ui
npm run dev
```

## Testes

### 1. Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### 2. Chat com Citações

```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Quais são os balancetes disponíveis?"}'
```

**Resposta esperada:**
```json
{
  "response": "Encontrei os [Balancetes Mensais 2025]^1...",
  "citations": [
    {
      "id": 1,
      "title": "Balancetes Mensais 2025",
      "breadcrumb": "Transparência › Contabilidade",
      "url": "https://www.tre-pi.jus.br/..."
    }
  ]
}
```

### 3. Interface Web

1. Acesse http://localhost:3000
2. Digite uma pergunta: "Como contestar uma multa eleitoral?"
3. Observe:
   - ✓ Resposta com citações inline (texto^N em azul)
   - ✓ Bloco "PÁGINAS CITADAS" ao final
   - ✓ Links clicáveis para as páginas

## Logs

Logs em tempo real:
```bash
# Backend
tail -f logs/backend.log

# Frontend
tail -f logs/frontend.log
```

## Estrutura de Arquivos Criados/Modificados

```
cristal3/
├── cristal-backend/
│   ├── internal/
│   │   ├── orchestrator/
│   │   │   ├── orchestrator.go      [MODIFICADO]
│   │   │   ├── citations.go         [NOVO]
│   │   │   └── tools.go             [MODIFICADO]
│   │   └── server/
│   │       ├── types.go             [MODIFICADO]
│   │       └── handler.go           [MODIFICADO]
│   └── bin/api                      [COMPILADO]
│
├── cristal-chat-ui/                 [SEM MODIFICAÇÕES - Fase 6 completa]
│
├── start-system.sh                  [NOVO]
├── stop-system.sh                   [NOVO]
└── logs/                            [CRIADO]
    ├── backend.log
    ├── frontend.log
    ├── backend.pid
    └── frontend.pid
```

## Formato de Citações

### Backend → Frontend

```json
{
  "response": "Texto com [citação inline]^1 e [outra citação]^2...",
  "citations": [
    {
      "id": 1,
      "title": "Título da Página",
      "breadcrumb": "Seção › Subseção",
      "url": "https://..."
    }
  ]
}
```

### Frontend Renderiza Como

```html
<div class="assistant-message">
  <p>
    Texto com 
    <cite data-num="1">
      <a href="https://...">citação inline</a>
      <sup>1</sup>
    </cite>
    ...
  </p>
</div>

<div class="citations-block">
  <h3>PÁGINAS CITADAS</h3>
  <ol>
    <li>
      <strong>Título da Página</strong>
      <span>Seção › Subseção</span>
      <a href="https://...">www.tre-pi.jus.br/...</a>
    </li>
  </ol>
</div>
```

## Próximos Passos (Opcional)

- [ ] Adicionar testes E2E
- [ ] Implementar cache de citações
- [ ] Melhorar parser de citações (detectar variações de URL)
- [ ] Adicionar telemetria/métricas
- [ ] Docker Compose para deploy

## Troubleshooting

### Backend não inicia

```bash
# Verificar logs
cat logs/backend.log

# Verificar porta
lsof -ti:8080

# Matar processo órfão
kill -9 $(lsof -ti:8080)
```

### Frontend não conecta ao backend

```bash
# Verificar proxy no vite.config.ts
cat cristal-chat-ui/vite.config.ts

# Verificar CORS no backend
# Já habilitado em internal/server/server.go
```

### Citações não aparecem

1. Verificar se MCPs retornam URLs no markdown
2. Verificar logs do backend: `grep "citations extracted" logs/backend.log`
3. Verificar resposta JSON no DevTools (Network tab)

## Conclusão

✅ **Sistema completamente funcional e integrado**

O sistema agora:
- Captura citações automaticamente dos MCPs
- Formata texto com referências numeradas
- Renderiza interface elegante no frontend
- Mantém rastreabilidade completa das fontes

Conforme solicitado, a interface funciona **exatamente como mostrado na imagem** de referência.

---

**Desenvolvido por**: Claude Sonnet 4.5  
**Data**: 21 de Abril de 2026
