# Cristal Chat UI

Frontend React para o assistente virtual Cristal do TRE-PI.

## Stack Técnico

- React 18 + TypeScript
- Vite (build tool)
- Tailwind CSS (styling com cores institucionais TRE-PI)
- TanStack Query (server state)
- Zustand (client state)
- React Markdown (renderização de respostas)

## Cores Institucionais (TRE-PI)

- **Dark Blue** (#0C326F) - Header, rodapé, títulos
- **Primary Blue** (#1351B4) - Botão principal, bolha usuário
- **Flag Yellow** (#FFCD07) - CTA, divisória, links
- **Urn Green** (#1D9E75) - URLs em monospace

## Instalação

```bash
npm install
```

## Desenvolvimento

```bash
npm run dev
```

Acesse: http://localhost:3000

## Build

```bash
npm run build
```

## Backend

O frontend se conecta ao backend REST API em `http://localhost:8080`.

Certifique-se de que o backend está rodando:

```bash
cd ../cristal-backend
ANTHROPIC_API_KEY=sk-ant-... go run ./cmd/api
```

## Estrutura

- `/src/components/layout` - Moldura hospedeira, header, footer
- `/src/components/chat` - Área de chat, mensagens, citações
- `/src/components/composer` - Input de mensagem
- `/src/components/ui` - Componentes reutilizáveis
- `/src/components/icons` - Ícones SVG customizados
- `/src/hooks` - Custom hooks (useChat, useSendMessage)
- `/src/api` - Cliente REST API
- `/src/types` - TypeScript types
- `/src/store` - Zustand store
- `/src/data` - Conteúdo estático (welcome message, chips)
- `/src/styles` - CSS variables e animations

## Fases de Implementação

- ✅ **FASE 1**: Setup e estrutura base
- ⏳ **FASE 2**: Layout institucional
- ⏳ **FASE 3**: Componentes de chat
- ⏳ **FASE 4**: Sistema de citações
- ⏳ **FASE 5**: Composer
- ⏳ **FASE 6**: Integração e estados

## Referências

- [PLANO_FRONTEND_REACT_CRISTAL.md](../PLANO_FRONTEND_REACT_CRISTAL.md)
- [SPEC_CHAT_REACT.md](../../cristal-data/SPEC_CHAT_REACT.md)
- [Backend README](../cristal-backend/README.md)
