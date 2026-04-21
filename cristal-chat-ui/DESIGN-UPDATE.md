# Atualização do Design - Cristal Chat UI

## Resumo

Implementação do design elegante do Cristal Chat com base no arquivo de design fornecido. O novo design apresenta uma interface limpa e moderna com as cores institucionais navy blue e gold, otimizada para exibição em iframe.

## Mudanças Implementadas

### 1. Sistema de Cores e Tipografia

**Arquivo:** `src/styles/variables.css`

- Implementado novo sistema de cores baseado na paleta Navy + Gold:
  - Navy: `--navy-900` até `--navy-050` (tons de azul marinho)
  - Gold/Yellow: `--gold-600` até `--gold-100` (tons de dourado/amarelo)
  - Ink: `--ink-900` até `--ink-300` (tons de texto)
  - Surface: `--paper`, `--surface`, `--line` (fundos e bordas)
  - Green: `--green-700`, `--green-500` (URLs e status)

- Adicionadas variáveis de tipografia:
  - `--font-sans`: Inter Tight, Inter, system-ui
  - `--font-display`: Fraunces (para títulos)
  - `--font-mono`: JetBrains Mono (para código/URLs)

- Variáveis de raio de borda:
  - `--radius-sm` até `--radius-xl`
  - `--radius-bubble`: 22px (para bolhas de mensagem)

- Shadows:
  - `--shadow-sm`, `--shadow-md`, `--shadow-lg`

### 2. Fontes Google

**Arquivo:** `index.html`

- Adicionadas Google Fonts: Inter Tight, Fraunces, JetBrains Mono
- Atualizado lang para "pt-BR"
- Título: "Cristal — Chat do Portal da Transparência"

### 3. Estilos Base

**Arquivo:** `src/index.css`

- Atualizado body para usar `var(--font-sans)`
- Melhorada renderização de texto com `text-rendering: optimizeLegibility`
- Atualizado scrollbar com novos estilos usando variáveis CSS
- Adicionada utility class `.rounded-pill` para bordas totalmente arredondadas

### 4. Animações

**Arquivo:** `src/styles/animations.css`

- Melhorada animação `pulse-dot` com transform translateY
- Adicionada animação `fadeUp` para mensagens
- Adicionada animação `fadeIn` para modais
- Duração ajustada para 1.2s no pulse-dot

### 5. Componentes Atualizados

#### Composer (`src/components/composer/Composer.tsx`)

- Design de pill totalmente arredondado
- Centralizado com max-width de 880px
- Melhor focus state com shadow azul
- Meta bar movida para baixo do composer
- Padding e espaçamento refinados

#### SendButton (`src/components/composer/SendButton.tsx`)

- Cor atualizada para `--navy-700` (azul marinho)
- Hover com `--navy-800`
- Efeito de translateY no hover
- Tamanho do ícone: 18px
- Estilo inline para melhor controle de cores

#### ComposerInput (`src/components/composer/ComposerInput.tsx`)

- Altura mínima: 40px
- Line height: 22px
- Font size: 15px
- Padding: 9px 6px
- Cor do texto usando variável `--ink-900`

#### ComposerToolbar (`src/components/composer/ComposerToolbar.tsx`)

- Botões circulares (36px x 36px)
- Cor base: `--ink-500`
- Hover: fundo `--navy-050`, texto `--navy-800`
- Gap removido (0)
- Ícones: 18px

#### ComposerMeta (`src/components/composer/ComposerMeta.tsx`)

- Font size: 12.5px
- Cor: `--ink-500`
- Texto padrão: "Baseado em páginas oficiais do portal"
- Hover refinado com transições suaves

#### WelcomeCard (`src/components/chat/WelcomeCard.tsx`)

- Max-width: 880px
- Border radius: `--radius-xl`
- Gradiente radial de fundo (efeito gold)
- Ícone diamante grande (78px) com borda gold
- Título com `font-display` (Fraunces)
- Sugestões com números (chips)
- Animação fade-up

#### SuggestionChip (`src/components/ui/SuggestionChip.tsx`)

- Design de pill com número opcional
- Cores: `--navy-050` background, `--navy-800` text
- Borda: `--navy-100`
- Número em mono com fundo branco
- Hover: translateY(-1px)

#### Avatar (`src/components/ui/Avatar.tsx`)

- Usuário: 30px, cor `--ink-300`
- Assistente: 34px, fundo `--navy-800`, borda `--gold-400`
- Ícone DiamondIcon em vez de emoji
- Tamanho do ícone: 17px

#### UserBubble (`src/components/chat/UserBubble.tsx`)

- Background: `--navy-800`
- Border radius: `--radius-bubble` (22px)
- Bottom right radius: 6px
- Padding: 18px
- Font size: 14.5px
- Font weight: 500
- Max-width: 560px
- Shadow: 0 2px 0 rgba(6,26,68,0.12)

#### AssistantMessage (`src/components/chat/AssistantMessage.tsx`)

- Max-width: 720px
- Font size: 14.5px
- Line height: 1.6
- Cor: `--ink-900`
- Links com underline gold
- Code blocks com fundo `--navy-050`
- Margins refinados

#### CitationInline (`src/components/chat/CitationInline.tsx`)

- Cor: `--navy-700`
- Underline: 2px, cor `--gold-400`
- Offset: 3px
- Sup em mono, cor `--navy-600`, 11px
- Hover: cor `--navy-900`

#### CitationsBlock (`src/components/chat/CitationsBlock.tsx`)

- Header com ícone de livro
- Background: `--paper`
- Border: `--line`
- Título em uppercase, tracking wide
- Font size: 11px
- Border radius: 14px

#### CitationItem (`src/components/chat/CitationItem.tsx`)

- Layout em grid (28px badge, 1fr content, auto button)
- Badge circular 24px, fundo `--navy-700`
- Breadcrumb com separadores `›`
- URL em mono, cor `--green-700`
- Botão "Abrir →" com uppercase
- Hover: fundo `--navy-050`

#### LoadingDots (`src/components/chat/LoadingDots.tsx`)

- Com avatar do assistente
- Texto: "Consultando o portal"
- Pontos: 1.5px, cor `--ink-300`
- Layout horizontal
- Font size: 14.5px

#### ChatArea (`src/components/chat/ChatArea.tsx`)

- Background: `--paper`
- Padding: 22px 24px 28px
- Animação fade-up nas mensagens
- Scroll suave

## Resultado

O design agora apresenta:

1. **Visual Elegante**: Cores navy blue e gold que remetem à instituição
2. **Tipografia Refinada**: Três fontes complementares (Inter Tight, Fraunces, JetBrains Mono)
3. **Composer Moderno**: Pill totalmente arredondado com botão azul circular
4. **Hierarquia Clara**: Espaçamentos consistentes e tamanhos de fonte apropriados
5. **Micro-interações**: Hover states suaves e animações sutis
6. **Acessibilidade**: Focus states visíveis e estrutura semântica
7. **Responsividade**: Layout que se adapta bem a diferentes tamanhos

## Como Testar

1. Execute o servidor de desenvolvimento:
   ```bash
   npm run dev
   ```

2. Abra http://localhost:5173 no navegador

3. Teste os elementos:
   - Welcome card com sugestões
   - Envio de mensagens
   - Visualização de citações
   - Composer com anexo e mic
   - Loading state
   - Hover states

## Próximos Passos (Opcional)

- Implementar modo escuro (dark mode)
- Adicionar mais micro-animações
- Otimizar para mobile/tablet
- Implementar temas customizáveis
- Adicionar mais estados visuais (erro, sucesso, etc.)
