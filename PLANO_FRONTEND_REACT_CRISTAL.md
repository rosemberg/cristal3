# Plano de Implementação: Frontend React Cristal

**Projeto**: Cristal Chat UI (React + TypeScript)  
**Versão**: 1.0  
**Data**: 2026-04-21  
**Autor**: Rosemberg Maia Gomes  
**Baseado em**: SPEC_CHAT_REACT.md + Identidade Visual TRE-PI

---

## Visão Geral

Implementar interface web do Cristal seguindo:
- **Stack técnico**: React 18 + TypeScript + Vite + Tailwind + Zustand + TanStack Query
- **Design**: Identidade visual institucional do TRE-PI (cores oficiais, bandeira brasileira)
- **Layout**: Moldura hospedeira + iframe simulado + componentes específicos
- **Integração**: Backend REST API (`localhost:8080`)

---

## Cores Institucionais (TRE-PI)

```css
:root {
  /* Cores principais */
  --dark-blue: #0C326F;        /* Header, rodapé, títulos */
  --primary-blue: #1351B4;     /* Botão principal, bolha usuário */
  --flag-yellow: #FFCD07;      /* CTA, divisória, sublinhado links */
  --urn-green: #1D9E75;        /* URLs em monospace */
  
  /* Cores secundárias */
  --light-blue-text: #B5D4F4;  /* Texto secundário header */
  --pale-blue-bg: #E6F1FB;     /* Chips de sugestão */
  
  /* Backgrounds */
  --chat-bg: #F7F7F5;          /* Área de rolagem */
  --card-bg: #FFFFFF;          /* Cards e input */
  
  /* Textos */
  --text-main: #1F2329;        /* Texto principal */
  --text-secondary: #5F5E5A;   /* Metadados */
  
  /* Bordas */
  --border-subtle: #D3D1C7;
}
```

---

## Arquitetura em Camadas

```
┌─────────────────────────────────────────────┐
│  Moldura Hospedeira (Site Institucional)    │
│  ┌───────────────────────────────────────┐  │
│  │  Browser Chrome (URL fictícia)        │  │
│  ├───────────────────────────────────────┤  │
│  │  Iframe Cristal (max-width: 720px)    │  │
│  │  ┌─────────────────────────────────┐  │  │
│  │  │  Header (--dark-blue)           │  │  │
│  │  │  Diamante | Cristal | Botões    │  │  │
│  │  ├─────────────────────────────────┤  │  │
│  │  │  Barra Amarela (3px)            │  │  │
│  │  ├─────────────────────────────────┤  │  │
│  │  │  Área de Chat (--chat-bg)       │  │  │
│  │  │  - Card Boas-Vindas             │  │  │
│  │  │  - Turnos de Conversa           │  │  │
│  │  │  - Citações e Referências       │  │  │
│  │  ├─────────────────────────────────┤  │  │
│  │  │  Composer (--card-bg)           │  │  │
│  │  │  Input + Ícones + Enviar        │  │  │
│  │  ├─────────────────────────────────┤  │  │
│  │  │  Rodapé (--dark-blue)           │  │  │
│  │  └─────────────────────────────────┘  │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

---

## Estrutura de Arquivos

```
cristal-chat-ui/
├── public/
│   ├── favicon.ico
│   └── diamond-icon.svg              # Ícone do cristal
├── src/
│   ├── components/
│   │   ├── layout/
│   │   │   ├── HostFrame.tsx         # Moldura hospedeira
│   │   │   ├── BrowserChrome.tsx     # Barra URL fictícia
│   │   │   ├── Header.tsx            # Header institucional
│   │   │   ├── YellowDivider.tsx     # Barra amarela 3px
│   │   │   └── Footer.tsx            # Rodapé institucional
│   │   ├── chat/
│   │   │   ├── ChatInterface.tsx     # Container principal
│   │   │   ├── ChatArea.tsx          # Área de rolagem
│   │   │   ├── WelcomeCard.tsx       # Card boas-vindas
│   │   │   ├── MessageTurn.tsx       # 1 turno completo
│   │   │   ├── UserBubble.tsx        # Bolha usuário
│   │   │   ├── AssistantMessage.tsx  # Resposta IA (sem bolha)
│   │   │   ├── CitationInline.tsx    # Link com superscript
│   │   │   ├── CitationsBlock.tsx    # "PÁGINAS CITADAS"
│   │   │   ├── CitationItem.tsx      # 1 item da lista
│   │   │   └── LoadingDots.tsx       # Animação "..."
│   │   ├── composer/
│   │   │   ├── Composer.tsx          # Área de input completa
│   │   │   ├── ComposerInput.tsx     # Textarea estilizada
│   │   │   ├── ComposerToolbar.tsx   # Anexar + Microfone
│   │   │   ├── SendButton.tsx        # Botão circular azul
│   │   │   └── ComposerMeta.tsx      # Barra "Baseado em..." + Limpar
│   │   ├── ui/
│   │   │   ├── Button.tsx            # Botões reutilizáveis
│   │   │   ├── IconButton.tsx        # Botões de ícone
│   │   │   ├── SuggestionChip.tsx    # Chips clicáveis
│   │   │   └── Avatar.tsx            # Avatares (VC, IA)
│   │   └── icons/
│   │       ├── DiamondIcon.tsx       # SVG diamante
│   │       ├── SendIcon.tsx          # SVG avião papel
│   │       ├── AttachIcon.tsx        # SVG clipe
│   │       ├── MicIcon.tsx           # SVG microfone
│   │       ├── TrashIcon.tsx         # SVG lixeira
│   │       └── InfoIcon.tsx          # SVG informação
│   ├── hooks/
│   │   ├── useChat.ts                # Lógica de chat
│   │   ├── useSendMessage.ts         # Enviar mensagens
│   │   └── useAutoScroll.ts          # Auto-scroll chat
│   ├── api/
│   │   └── client.ts                 # Cliente REST
│   ├── types/
│   │   ├── chat.ts                   # Mensagens, turnos
│   │   └── citation.ts               # Tipos de citação
│   ├── store/
│   │   └── chatStore.ts              # Zustand store
│   ├── data/
│   │   └── welcomeContent.ts         # Perguntas sugeridas (chips)
│   ├── styles/
│   │   ├── variables.css             # CSS vars das cores
│   │   └── animations.css            # Keyframes (loading dots)
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css                     # Tailwind + Custom CSS
├── .env
├── .env.example
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.js
├── postcss.config.js
└── README.md
```

---

## Plano de Implementação (6 Fases)

### **FASE 1: Setup e Estrutura Base** (2-3 horas)

#### Objetivos:
- Criar projeto Vite + React + TypeScript
- Configurar Tailwind com cores institucionais
- Estrutura de diretórios
- CSS vars no :root

#### Tarefas:
1. ✅ Criar projeto: `npm create vite@latest cristal-chat-ui -- --template react-ts`
2. ✅ Instalar dependências:
   ```bash
   npm install @tanstack/react-query zustand react-markdown lucide-react uuid
   npm install -D @types/uuid tailwindcss postcss autoprefixer
   ```
3. ✅ Setup Tailwind: `npx tailwindcss init -p`
4. ✅ Criar `src/styles/variables.css` com cores TRE-PI
5. ✅ Configurar `tailwind.config.js` para usar CSS vars
6. ✅ Criar estrutura de pastas
7. ✅ Configurar `vite.config.ts` com proxy para `:8080`

#### Entregáveis:
- Projeto rodando (`npm run dev`)
- Cores institucionais disponíveis via Tailwind
- Estrutura de pastas criada

---

### **FASE 2: Layout Institucional** (3-4 horas)

#### Objetivos:
- Implementar moldura hospedeira
- Header institucional completo
- Rodapé institucional
- Barra amarela divisória

#### Tarefas:
1. ✅ **HostFrame.tsx**: Wrapper cinza com borda tracejada
2. ✅ **BrowserChrome.tsx**: 
   - 3 bolinhas (vermelho, amarelo, verde)
   - URL fictícia `www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/cristal`
3. ✅ **Header.tsx**:
   - Fundo `--dark-blue`
   - À esquerda: DiamondIcon + "Cristal" + subtítulo (--light-blue-text)
   - À direita: Botões acessibilidade + CTA "Ouvidoria / SIC" (--flag-yellow)
4. ✅ **YellowDivider.tsx**: Barra 3px amarela
5. ✅ **Footer.tsx**:
   - Fundo `--dark-blue`
   - Texto "© TRE-PI"
   - Links "Política de privacidade · Ouvidoria · LGPD" (hover amarelo)

#### Entregáveis:
- Layout completo renderizando
- Responsivo (max-width: 720px)
- Cores institucionais aplicadas

---

### **FASE 3: Componentes de Chat (UI Pura)** (4-5 horas)

#### Objetivos:
- Área de chat com scroll customizado
- Card de boas-vindas
- Bolhas de mensagens (usuário e IA)
- Avatares

#### Tarefas:
1. ✅ **ChatArea.tsx**:
   - Fundo `--chat-bg`
   - Scroll customizado (styled scrollbar)
   - Padding adequado
2. ✅ **WelcomeCard.tsx**:
   - Card branco
   - Título "Olá! Sou a Cristal..."
   - 3 chips clicáveis:
     - "Como contestar uma multa?"
     - "Quais diárias foram pagas?"
     - "Onde vejo as licitações?"
3. ✅ **SuggestionChip.tsx**:
   - Fundo `--pale-blue-bg`
   - Borda sutil
   - Hover state
4. ✅ **UserBubble.tsx**:
   - Fundo `--primary-blue`
   - Texto branco
   - Alinhado à direita
   - Borda inferior direita reta
   - Avatar "VC"
5. ✅ **AssistantMessage.tsx**:
   - Sem bolha (texto flutuando)
   - Avatar "IA" (borda amarela)
   - Cor de texto `--text-main`
6. ✅ **Avatar.tsx**:
   - Versão "VC" (usuário)
   - Versão "IA" (borda `--flag-yellow`)

#### Entregáveis:
- Chat área renderizando
- Componentes de mensagem funcionais
- Mock inicial visível

---

### **FASE 4: Sistema de Citações** (3-4 horas)

#### Objetivos:
- Links inline com superscript
- Bloco "PÁGINAS CITADAS"
- Formatação de referências

#### Tarefas:
1. ✅ **CitationInline.tsx**:
   - Link com borda inferior `--flag-yellow` (2px)
   - Superscript `<sup>1</sup>` em azul
   - Hover: cor mais escura
2. ✅ **CitationsBlock.tsx**:
   - Card branco
   - Título "PÁGINAS CITADAS" (uppercase, --text-secondary)
   - Lista numerada
3. ✅ **CitationItem.tsx**:
   - Círculo azul com número
   - Título do link (--primary-blue)
   - Breadcrumb (--text-secondary, menor)
   - URL (--urn-green, font-family: monospace)
4. ✅ **Tipos TypeScript** (`types/citation.ts`):
   ```typescript
   interface Citation {
     id: number;
     title: string;
     breadcrumb: string;
     url: string;
   }
   ```

#### Entregáveis:
- Sistema de citações completo
- Mock data com 2-3 referências
- Links clicáveis e estilizados

---

### **FASE 5: Composer (Área de Input)** (3-4 horas)

#### Objetivos:
- Input em formato de pílula
- Ícones integrados
- Botão de enviar circular
- Barra de metadados

#### Tarefas:
1. ✅ **Composer.tsx**: Container principal
2. ✅ **ComposerInput.tsx**:
   - `border-radius: 24px`
   - Fundo `--card-bg`
   - Borda `--border-subtle`
   - `:focus-within` → destaque visual
   - Auto-resize do textarea
3. ✅ **ComposerToolbar.tsx**:
   - Ícones SVG: AttachIcon + MicIcon
   - Botões sem background (ghost)
   - Hover state sutil
4. ✅ **SendButton.tsx**:
   - Botão circular
   - Fundo `--primary-blue`
   - Ícone avião de papel branco
   - Disabled state (cinza)
5. ✅ **ComposerMeta.tsx**:
   - InfoIcon + "Baseado em páginas oficiais do TRE-PI"
   - Botão "Limpar" com TrashIcon
   - Texto `--text-secondary`, tamanho pequeno
6. ✅ **Ícones SVG** (criar todos):
   - DiamondIcon (cristal geométrico)
   - SendIcon (avião de papel)
   - AttachIcon (clipe)
   - MicIcon (microfone)
   - TrashIcon (lixeira)
   - InfoIcon (círculo com "i")

#### Entregáveis:
- Composer completo e funcional
- Ícones SVG customizados
- Input responsivo

---

### **FASE 6: Integração e Estados** (4-5 horas)

#### Objetivos:
- Conectar com backend
- Loading states
- Error handling
- Estado inicial vazio (só card de boas-vindas)

#### Tarefas:
1. ✅ **API Client** (`api/client.ts`):
   ```typescript
   sendMessage(message: string) → ChatResponse
   checkHealth() → { status: string }
   ```
2. ✅ **Zustand Store** (`store/chatStore.ts`):
   ```typescript
   interface ChatStore {
     messages: Message[];
     citations: Citation[];
     isLoading: boolean;
     error: string | null;
     addMessage: (msg: Message) => void;
     addCitations: (cites: Citation[]) => void;
     clearChat: () => void;
   }
   ```
   **Estado inicial**: `messages: []` (vazio)
3. ✅ **Hook useSendMessage** (`hooks/useSendMessage.ts`):
   - TanStack Query mutation
   - onMutate: adicionar mensagem usuário
   - onSuccess: adicionar resposta IA + citações
   - onError: exibir erro
4. ✅ **LoadingDots.tsx**:
   - Animação CSS (3 pontinhos piscando)
   - Texto "Consultando o portal..."
   - Keyframes:
   ```css
   @keyframes pulsando {
     0%, 100% { opacity: 0.3; }
     50% { opacity: 1; }
   }
   ```
5. ✅ **ChatInterface.tsx**: Orquestrar tudo
6. ✅ **useAutoScroll.ts**: Scroll para última mensagem
7. ✅ **Estado Inicial da UI**:
   - Chat vazio (sem mensagens)
   - Apenas WelcomeCard visível
   - Chips clicáveis funcionais (enviam pergunta ao clicar)

#### Entregáveis:
- Frontend funcionando end-to-end
- Mock content renderizado corretamente
- Loading e error states
- Integração com backend testada

---

## Mock Content (Referência de Design)

> **IMPORTANTE**: Este conteúdo é apenas **referência visual** para ilustrar como cada tipo de mensagem deve ser renderizada. A interface deve **começar vazia** (só com o card de boas-vindas) e as mensagens só aparecem quando o usuário interage com o backend real.

### Exemplo: Pergunta Complexa com Citações

**Usuário:**
```
"Como funciona a restituição de multa eleitoral?"
```

**IA:**
```
Para pedir restituição de uma multa eleitoral já paga, o TRE-PI exige 
requerimento formal, documentação comprobatória e análise conforme as 
resoluções vigentes.

O [procedimento completo, com formulários e requisitos]¹ fica na seção 
de serviços eleitorais. As regras de aplicação e cobrança seguem as 
[resoluções do TRE-PI sobre multas]², que definem hipóteses de 
restituição e prazos.
```

**Formato das Citações:**
1. **Título**: Procedimento de restituição de multas eleitorais  
   **Breadcrumb**: Transparência › Serviços › Multas  
   **URL**: `tre-pi.jus.br/.../restituicao-multas.html`

2. **Título**: Resoluções sobre aplicação de multas — TRE-PI  
   **Breadcrumb**: Transparência › Normas › Resoluções  
   **URL**: `tre-pi.jus.br/.../resolucoes-multas-2025.pdf`

**Disclaimer:**
```
⚠️ Confirme sempre prazos e documentos atualizados no portal oficial.
```

---

### Exemplo: Resposta Simples

**Usuário:**
```
"E se eu quiser contestar antes de pagar, qual o prazo?"
```

**IA:**
```
A contestação deve ser apresentada em até 10 dias após a notificação. 
O [formulário de defesa e os documentos necessários]¹ estão disponíveis 
na seção de recursos e contestações.
```

**Formato da Citação:**
1. **Título**: Recursos e contestações de multas eleitorais  
   **Breadcrumb**: Transparência › Serviços › Recursos  
   **URL**: `tre-pi.jus.br/.../recursos-contestacoes.html`

---

### Exemplo: Loading State

**Usuário:**
```
"Como falo com a Ouvidoria?"
```

**IA:**
```
[LoadingDots animando]
Consultando o portal...
```

---

## Requisitos Técnicos

### Semântica HTML
- `<main>` para área de chat
- `<header>` e `<footer>` institucionais
- `aria-label` em todos os botões de ícone
- `role="button"` em chips clicáveis

### Acessibilidade
- Contraste WCAG AAA
- Botões "Alto contraste" e "A+" funcionais (futuro)
- Keyboard navigation (Tab, Enter, Esc)

### Animações
- Transições suaves (`transition: all 0.2s ease`)
- Hover states elegantes
- Loading dots com `@keyframes`

### Responsividade
- Desktop: `max-width: 720px`
- Mobile: Full width com padding lateral
- Composer: Ajusta altura automaticamente

### Performance
- Lazy loading de componentes pesados
- Memoization onde necessário (`React.memo`)
- Virtual scrolling para muitas mensagens (futuro)

---

## Configuração Tailwind Customizada

```javascript
// tailwind.config.js
module.exports = {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        'dark-blue': 'var(--dark-blue)',
        'primary-blue': 'var(--primary-blue)',
        'flag-yellow': 'var(--flag-yellow)',
        'urn-green': 'var(--urn-green)',
        'light-blue-text': 'var(--light-blue-text)',
        'pale-blue-bg': 'var(--pale-blue-bg)',
        'chat-bg': 'var(--chat-bg)',
        'card-bg': 'var(--card-bg)',
        'border-subtle': 'var(--border-subtle)',
        'text-main': 'var(--text-main)',
        'text-secondary': 'var(--text-secondary)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Courier New', 'monospace'],
      },
      borderRadius: {
        'pill': '24px',
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
};
```

---

## Comandos Úteis

```bash
# Iniciar projeto
npm create vite@latest cristal-chat-ui -- --template react-ts
cd cristal-chat-ui
npm install

# Instalar dependências principais
npm install @tanstack/react-query zustand react-markdown uuid

# Instalar Tailwind
npm install -D tailwindcss postcss autoprefixer @types/uuid
npx tailwindcss init -p

# Dev server
npm run dev  # http://localhost:3000

# Build
npm run build

# Preview build
npm run preview

# Lint
npm run lint
```

---

## Testes de Aceitação

### Fase 2 (Layout):
- [ ] Moldura hospedeira renderiza corretamente
- [ ] Header azul escuro com diamante e botões
- [ ] Barra amarela de 3px visível
- [ ] Rodapé com links hover amarelo

### Fase 3 (Chat UI):
- [ ] Card de boas-vindas com 3 chips
- [ ] Bolha do usuário azul alinhada à direita
- [ ] Mensagem da IA sem bolha, com avatar "IA"
- [ ] Scroll customizado funciona

### Fase 4 (Citações):
- [ ] Links inline com superscript e borda amarela
- [ ] Bloco "PÁGINAS CITADAS" renderiza
- [ ] URLs em verde e monospace
- [ ] Hover states funcionam

### Fase 5 (Composer):
- [ ] Input em formato de pílula
- [ ] Ícones anexar e microfone visíveis
- [ ] Botão enviar circular azul
- [ ] Barra de metadados abaixo do input
- [ ] Botão "Limpar" funcional

### Fase 6 (Integração):
- [ ] Interface inicia vazia (só card de boas-vindas)
- [ ] Clicar em chip envia pergunta ao backend
- [ ] Enviar mensagem adiciona ao chat
- [ ] Loading dots animam corretamente
- [ ] Resposta do backend renderiza com citações
- [ ] Error handling exibe mensagem clara
- [ ] Auto-scroll para última mensagem
- [ ] Botão "Limpar" esvazia o chat

---

## Cronograma Estimado

| Fase | Descrição | Tempo | Dependências |
|------|-----------|-------|--------------|
| 1 | Setup e estrutura | 2-3h | - |
| 2 | Layout institucional | 3-4h | Fase 1 |
| 3 | Componentes de chat | 4-5h | Fase 2 |
| 4 | Sistema de citações | 3-4h | Fase 3 |
| 5 | Composer | 3-4h | Fase 3 |
| 6 | Integração e estados | 4-5h | Todas |

**Total**: 19-25 horas (~3-4 dias de trabalho)

---

## Próximos Passos Após MVP

### Features Avançadas:
- [ ] Histórico persistente (localStorage)
- [ ] Sessões múltiplas
- [ ] Syntax highlighting em code blocks
- [ ] Copy to clipboard
- [ ] Export chat (PDF/MD)
- [ ] Dark mode toggle (manter cores institucionais)
- [ ] Voice input (Web Speech API)
- [ ] PWA (service worker)

### Performance:
- [ ] Virtual scrolling (react-window)
- [ ] Code splitting por rota
- [ ] Image lazy loading
- [ ] Prefetch de links

### UX:
- [ ] Animações com framer-motion
- [ ] Toast notifications
- [ ] Skeleton loading states
- [ ] Empty state ilustrado
- [ ] Sugestões contextuais

---

## Referências

- **Spec Técnica**: `/Users/rosemberg/projetos-gemini/cristal-data/SPEC_CHAT_REACT.md`
- **Backend**: `/Users/rosemberg/projetos-gemini/cristal-data/cristal-backend/`
- **Design System**: Este documento (cores institucionais TRE-PI)
- **React**: https://react.dev
- **Vite**: https://vitejs.dev
- **Tailwind**: https://tailwindcss.com
- **TanStack Query**: https://tanstack.com/query
- **Zustand**: https://github.com/pmndrs/zustand

---

## Status

- ✅ Plano de implementação completo
- ⏳ Aguardando início da Fase 1
- 🎯 Objetivo: MVP funcional em 3-4 dias

**Próxima ação**: Iniciar Fase 1 (Setup e Estrutura Base)

---

**Data**: 2026-04-21  
**Versão**: 1.0  
**Pronto para implementação!** 🚀
