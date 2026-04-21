# Instruções para Executar FASE 6

## 1. Criar arquivo .env

O arquivo `.env` precisa ser criado manualmente na raiz do projeto:

```bash
cd /Users/rosemberg/projetos-gemini/cristal3/cristal-chat-ui
cp .env.example .env
```

Ou crie manualmente com o conteúdo:
```
VITE_API_BASE_URL=http://localhost:8080
```

## 2. Verificar dependências

Todas as dependências já estão no package.json:

```bash
npm install
```

## 3. Iniciar o backend

Antes de iniciar o frontend, certifique-se de que o backend está rodando em `http://localhost:8080`.

O backend deve ter os seguintes endpoints:

### POST /api/chat
```json
Request:
{
  "message": "Como contestar uma multa?"
}

Response:
{
  "response": "Para contestar uma multa...",
  "citations": [
    {
      "id": 1,
      "title": "Título da página",
      "breadcrumb": "Path › No › Portal",
      "url": "https://exemplo.com"
    }
  ]
}
```

### GET /api/health
```json
Response:
{
  "status": "ok"
}
```

## 4. Iniciar o frontend

```bash
npm run dev
```

Acesse: http://localhost:3000

## 5. Testar o fluxo

1. Você verá o WelcomeCard (estado inicial vazio)
2. Digite uma mensagem no Composer
3. Clique em "Enviar" ou pressione Enter
4. A mensagem do usuário aparece imediatamente
5. LoadingDots aparece ("Consultando o portal...")
6. Resposta do assistente aparece com citações (se houver)
7. Citações clicáveis aparecem abaixo da resposta
8. Auto-scroll para a última mensagem
9. Botão "Limpar conversa" fica disponível após primeira mensagem

## 6. Verificar integração

Abra o DevTools do navegador:

### Console
- Verifique se não há erros no console
- Veja as requisições POST /api/chat sendo enviadas

### Network
- Verifique as requisições para /api/chat
- Veja os payloads e respostas

### React DevTools (se instalado)
- Inspecione o estado do Zustand store
- Veja messages, citations, isLoading, error

## Troubleshooting

### Backend não está respondendo

Se aparecer erro:
```
Erro ao enviar mensagem
```

Verifique:
1. Backend está rodando? `curl http://localhost:8080/api/health`
2. Endpoint correto? Deve ser `/api/chat` não `/chat`
3. CORS configurado no backend?

### Citações não aparecem

Verifique:
1. Backend está retornando `citations` no response?
2. Formato do Citation está correto?
3. Citations tem `id`, `title`, `breadcrumb`, `url`?

### WelcomeCard não aparece

Verifique:
1. Estado inicial do store: `messages: []`
2. ChatArea renderiza WelcomeCard quando `messages.length === 0`

### Loading infinito

Verifique:
1. Backend respondeu?
2. Houve erro na requisição?
3. Veja o error banner no topo da tela

## Estrutura esperada no Backend

```python
# Exemplo em Python/FastAPI
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

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
    # Processar mensagem
    response_text = process_message(request.message)
    
    # Gerar citações
    citations = get_citations_for_response(response_text)
    
    return ChatResponse(
        response=response_text,
        citations=citations
    )

@app.get("/api/health")
async def health():
    return {"status": "ok"}
```

## Próximos passos

Após validar que tudo está funcionando:

1. Testar com diferentes tipos de perguntas
2. Verificar formatação de citações
3. Testar error handling (backend offline)
4. Testar timeout (resposta > 30s)
5. Testar Clear chat
6. Testar sugestões do WelcomeCard

## Suporte

Se encontrar problemas:

1. Verifique FASE6-SUMMARY.txt para visão geral
2. Leia FASE6-INTEGRATION.md para documentação completa
3. Verifique logs do console do navegador
4. Verifique logs do backend
5. Teste endpoints do backend com curl/Postman primeiro

