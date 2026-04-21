# Quick Start - FASE 6

## Comandos Rápidos

### 1. Criar .env
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/cristal-chat-ui
cp .env.example .env
```

### 2. Instalar dependências (se necessário)
```bash
npm install
```

### 3. Iniciar frontend
```bash
npm run dev
```

### 4. Build produção
```bash
npm run build
```

---

## Verificação de Endpoints Backend

### Testar Health Check
```bash
curl http://localhost:8080/api/health
```

Resposta esperada:
```json
{"status":"ok"}
```

### Testar Chat
```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Como contestar uma multa?"}'
```

Resposta esperada:
```json
{
  "response": "Para contestar uma multa...",
  "citations": [
    {
      "id": 1,
      "title": "Título",
      "breadcrumb": "Path › No › Portal",
      "url": "https://exemplo.com"
    }
  ]
}
```

---

## Estrutura de Resposta do Backend

```typescript
// Request
POST /api/chat
{
  "message": string
}

// Response
{
  "response": string,        // Texto da resposta (markdown suportado)
  "citations": Citation[]    // Array de citações (opcional)
}

// Citation
{
  "id": number,             // ID sequencial (1, 2, 3...)
  "title": string,          // Título da página
  "breadcrumb": string,     // Caminho (ex: "Transparência › Licitações")
  "url": string            // URL completa
}
```

---

## URLs Importantes

- **Frontend Dev**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost:8080/api/health
- **Chat Endpoint**: http://localhost:8080/api/chat

---

## Arquivos de Documentação

1. **README-FASE6.md** - Resumo completo da implementação
2. **INSTRUCOES-FASE6.md** - Guia passo a passo com troubleshooting
3. **FASE6-INTEGRATION.md** - Documentação técnica detalhada
4. **FASE6-SUMMARY.txt** - Resumo executivo rápido
5. **CHECKLIST-FASE6.md** - Checklist de validação
6. **QUICKSTART.md** - Este arquivo

---

## Estrutura de Pastas

```
cristal-chat-ui/
├── src/
│   ├── api/              # Cliente HTTP
│   ├── store/            # Zustand store
│   ├── hooks/            # Custom hooks
│   ├── components/       # Componentes React
│   ├── types/            # TypeScript interfaces
│   └── App.tsx           # App principal
├── .env                  # Variáveis de ambiente (criar)
├── .env.example          # Template
└── package.json          # Dependências
```

---

## Checklist Rápido

- [ ] Criar .env (cp .env.example .env)
- [ ] Backend rodando em localhost:8080
- [ ] Endpoints /api/chat e /api/health funcionando
- [ ] npm install (se necessário)
- [ ] npm run dev
- [ ] Testar no navegador (localhost:3000)

---

## Exemplo de Backend Mínimo (FastAPI)

```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

app = FastAPI()

# CORS para desenvolvimento
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

class ChatRequest(BaseModel):
    message: str

class Citation(BaseModel):
    id: int
    title: str
    breadcrumb: str
    url: str

class ChatResponse(BaseModel):
    response: str
    citations: list[Citation] = []

@app.post("/api/chat")
async def chat(request: ChatRequest):
    # Processar mensagem e gerar resposta
    response_text = f"Você perguntou: {request.message}"
    
    # Citações de exemplo
    citations = [
        Citation(
            id=1,
            title="Portal de Transparência",
            breadcrumb="Home › Transparência",
            url="https://www.tre-pi.jus.br/transparencia"
        )
    ]
    
    return ChatResponse(
        response=response_text,
        citations=citations
    )

@app.get("/api/health")
async def health():
    return {"status": "ok"}

# Executar: uvicorn main:app --reload --port 8080
```

---

## Troubleshooting Rápido

### Erro: "Erro ao enviar mensagem"
- Backend está rodando?
- Endpoints corretos?
- CORS configurado?

### Erro: "Tempo limite de resposta excedido"
- Timeout é 30s
- Backend está demorando muito?
- Rede lenta?

### WelcomeCard não aparece
- Estado inicial: messages = []
- Limpe o cache do navegador
- Verifique console do navegador

### Citações não aparecem
- Backend retorna "citations" array?
- Formato correto? (id, title, breadcrumb, url)
- Veja Network tab no DevTools

---

## Comandos de Debug

```bash
# Ver logs do frontend
# Abrir DevTools (F12) > Console

# Testar backend
curl http://localhost:8080/api/health

# Ver versão do Node
node --version

# Ver versão do npm
npm --version

# Limpar cache
rm -rf node_modules package-lock.json
npm install

# Build limpo
rm -rf dist
npm run build
```

---

## Próximo Passo

Após iniciar, teste o fluxo:

1. Acesse http://localhost:3000
2. Veja WelcomeCard
3. Digite uma mensagem
4. Clique em "Enviar"
5. Veja LoadingDots
6. Veja resposta do assistente
7. Clique nas citações
8. Teste "Limpar conversa"

✅ Se tudo funcionar, FASE 6 está completa!

---

Criado em: 21 de Abril de 2026
