# FASE 6: Integração com Backend e Estados

## Status: ✅ COMPLETO

Integração completa com backend REST API usando Zustand para estado global e TanStack Query para gerenciamento de requisições.

---

## Estrutura Implementada

```
src/
├── api/
│   ├── client.ts           # Cliente HTTP para comunicação com backend
│   └── index.ts            # Barrel export
├── store/
│   └── chatStore.ts        # Zustand store global
├── hooks/
│   ├── useSendMessage.ts   # Hook de mutation para enviar mensagens
│   ├── useAutoScroll.ts    # Hook para auto-scroll
│   └── index.ts            # Barrel export
├── styles/
│   └── animations.css      # Keyframes para loading dots
└── types/
    └── chat.ts             # Interfaces TypeScript atualizadas
```

---

## 1. API Client (`src/api/client.ts`)

Cliente HTTP configurado para comunicação com backend REST API em `http://localhost:8080`.

### Endpoints

- **POST /api/chat** - Envia mensagem e recebe resposta com citações
- **GET /api/health** - Verifica saúde da API

### Características

- Timeout de 30 segundos
- Error handling apropriado
- Headers: `Content-Type: application/json`
- Suporte a variável de ambiente `VITE_API_BASE_URL`

### Uso

```typescript
import { apiClient } from '@/api/client';

const response = await apiClient.sendMessage('Qual o prazo para contestação?');
// { response: string, citations?: Citation[] }

const health = await apiClient.checkHealth();
// { status: string }
```

---

## 2. Zustand Store (`src/store/chatStore.ts`)

Estado global do chat gerenciado com Zustand.

### Estado

```typescript
interface ChatStore {
  messages: Message[];        // Lista de mensagens (user/assistant)
  citations: Citation[];      // Citações globais
  isLoading: boolean;         // Estado de loading
  error: string | null;       // Mensagem de erro

  addMessage: (message: Message) => void;
  addCitations: (citations: Citation[]) => void;
  clearChat: () => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}
```

### Estado Inicial

- `messages: []` (vazio - WelcomeCard visível)
- `citations: []`
- `isLoading: false`
- `error: null`

### Uso

```typescript
import { useChatStore } from '@/store/chatStore';

const messages = useChatStore(state => state.messages);
const addMessage = useChatStore(state => state.addMessage);
const clearChat = useChatStore(state => state.clearChat);
```

---

## 3. Hook useSendMessage (`src/hooks/useSendMessage.ts`)

Mutation hook usando TanStack Query para enviar mensagens.

### Fluxo

1. **onMutate**: Adiciona mensagem do usuário imediatamente
2. **onSuccess**: Adiciona resposta do assistente e citações
3. **onError**: Define mensagem de erro

### Uso

```typescript
import { useSendMessage } from '@/hooks/useSendMessage';

const { mutate: sendMessage } = useSendMessage();

sendMessage('Como contestar uma multa?');
```

---

## 4. Hook useAutoScroll (`src/hooks/useAutoScroll.ts`)

Auto-scroll suave para a última mensagem quando há mudanças.

### Uso

```typescript
import { useAutoScroll } from '@/hooks/useAutoScroll';

const scrollRef = useAutoScroll([messages, isLoading]);

<div ref={scrollRef}>
  {/* Chat content */}
</div>
```

---

## 5. LoadingDots Component

Componente de loading com 3 pontos pulsantes.

### Características

- Animação CSS com keyframes `pulse-dot`
- Texto padrão: "Consultando o portal..."
- Delays sequenciais: 0s, 0.2s, 0.4s

---

## 6. Configuração

### vite.config.ts

Proxy configurado para redirecionar requisições `/api` para o backend:

```typescript
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

### .env

Variável de ambiente para base URL do backend:

```env
VITE_API_BASE_URL=http://localhost:8080
```

---

## 7. App.tsx

Componente principal atualizado com:

- `QueryClientProvider` para TanStack Query
- Integração com Zustand store
- Hook `useSendMessage` para mutations
- Error banner para exibir erros
- Estado inicial vazio (WelcomeCard visível)

---

## 8. Tipos TypeScript

### Message

```typescript
interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}
```

### Citation

```typescript
interface Citation {
  id: number;
  title: string;
  breadcrumb: string;
  url: string;
}
```

### ChatResponse

```typescript
interface ChatResponse {
  response: string;
  citations?: Citation[];
}
```

---

## Como Testar

### 1. Instalar dependências

```bash
npm install
```

### 2. Configurar variáveis de ambiente

Crie o arquivo `.env` na raiz do projeto:

```bash
cp .env.example .env
```

### 3. Iniciar o backend

Certifique-se de que o backend está rodando em `http://localhost:8080` com os endpoints:
- POST /api/chat
- GET /api/health

### 4. Iniciar o frontend

```bash
npm run dev
```

Acesse: http://localhost:3000

---

## Checklist de Implementação

- [x] API Client com sendMessage e checkHealth
- [x] Zustand Store com estado inicial vazio
- [x] Hook useSendMessage com TanStack Query
- [x] LoadingDots component com animação
- [x] animations.css com keyframes
- [x] useAutoScroll hook
- [x] types/chat.ts com interfaces
- [x] App.tsx com integração completa
- [x] vite.config.ts com proxy
- [x] .env.example com variáveis
- [x] WelcomeCard aparece quando messages.length === 0
- [x] Loading state desabilita Composer
- [x] Error handling apropriado
- [x] Auto-scroll suave

---

## Próximos Passos

### FASE 7 (Sugerida): Melhorias e Otimizações

- Adicionar retry automático em caso de falha
- Implementar cache de mensagens no localStorage
- Adicionar suporte a websockets para streaming de respostas
- Implementar paginação de mensagens antigas
- Adicionar testes unitários e de integração

---

## Estrutura de Pastas Final

```
cristal-chat-ui/
├── src/
│   ├── api/
│   │   ├── client.ts
│   │   └── index.ts
│   ├── components/
│   │   ├── chat/
│   │   ├── composer/
│   │   ├── icons/
│   │   ├── layout/
│   │   └── ui/
│   ├── data/
│   ├── hooks/
│   │   ├── useAutoScroll.ts
│   │   ├── useSendMessage.ts
│   │   └── index.ts
│   ├── store/
│   │   └── chatStore.ts
│   ├── styles/
│   │   ├── animations.css
│   │   └── variables.css
│   ├── types/
│   │   ├── chat.ts
│   │   └── citation.ts
│   ├── utils/
│   ├── App.tsx
│   ├── index.css
│   └── main.tsx
├── .env
├── .env.example
├── package.json
├── vite.config.ts
└── tsconfig.json
```

---

## Dependências Utilizadas

- **React 19.2.5** - Framework UI
- **Zustand 5.0.12** - Estado global
- **TanStack Query 5.99.2** - Gerenciamento de requisições
- **UUID 14.0.0** - Geração de IDs únicos
- **Vite 8.0.9** - Build tool
- **TypeScript 6.0.2** - Type safety

---

## Observações Importantes

1. **Estado Inicial Vazio**: O store inicia com `messages: []` para garantir que o WelcomeCard seja exibido inicialmente.

2. **Citações Globais**: As citações são armazenadas separadamente no store e passadas para os componentes via props, não ficam mais dentro do objeto Message.

3. **Error Handling**: Erros são capturados e exibidos em um banner no topo da tela com animação.

4. **Loading State**: Durante o loading, o Composer fica desabilitado e um LoadingDots é exibido no chat.

5. **Auto-scroll**: A cada nova mensagem ou mudança de estado de loading, o chat faz scroll automático para o final.

---

Desenvolvido por Claude Sonnet 4.5 para o Projeto Cristal Chat UI - TRE-PI
