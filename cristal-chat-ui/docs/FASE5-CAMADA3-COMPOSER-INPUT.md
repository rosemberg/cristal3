# FASE 5 - CAMADA 3: ComposerInput

## Status: ✅ COMPLETO

Implementação do componente `ComposerInput` - textarea inteligente com auto-resize e keyboard handling avançado.

---

## 📁 Arquivos Criados

### Componente Principal
- **`src/components/composer/ComposerInput.tsx`** - Textarea inteligente com auto-resize

### Arquivos de Exemplo
- **`src/examples/ComposerInputExample.tsx`** - Demonstração interativa

### Arquivos Atualizados
- **`src/components/composer/index.ts`** - Export do ComposerInput

---

## 🎯 Funcionalidades Implementadas

### 1. Auto-resize Dinâmico
```typescript
// Altura mínima: 44px (1 linha)
// Altura máxima: 120px (6 linhas)
// Scroll interno aparece após max-height

useEffect(() => {
  const textarea = textareaRef.current;
  if (!textarea) return;
  
  textarea.style.height = 'auto';
  const lineHeight = 20;
  const maxHeight = maxRows * lineHeight;
  const newHeight = Math.min(textarea.scrollHeight, maxHeight);
  textarea.style.height = `${newHeight}px`;
}, [value, maxRows]);
```

### 2. Keyboard Handling Inteligente
- **Enter** (sem Shift): Envia mensagem via `onSubmit()`
- **Shift+Enter**: Quebra linha normalmente
- **Escape**: Remove foco do input
- Validação: Só envia se `value.trim()` não estiver vazio

### 3. Estados Visuais
- **Disabled**: Opacidade reduzida + cursor `not-allowed`
- **Placeholder**: Cor `text-secondary` (#5F5E5A)
- **Focus/Blur**: Callbacks `onFocus` e `onBlur` para controle externo

### 4. Acessibilidade
- `aria-label="Campo de mensagem"`
- `aria-multiline="true"`
- `aria-disabled` quando desabilitado
- Focus visível gerenciado pelo container pai

---

## 📐 Interface TypeScript

```typescript
interface ComposerInputProps {
  value: string;                      // Valor controlado
  onChange: (value: string) => void;  // Callback de mudança (recebe string)
  onSubmit: () => void;               // Callback de envio (Enter)
  onFocus?: () => void;               // Callback de foco
  onBlur?: () => void;                // Callback de blur
  placeholder?: string;               // Texto placeholder (default: 'Digite sua pergunta...')
  disabled?: boolean;                 // Estado desabilitado (default: false)
  maxRows?: number;                   // Máximo de linhas visíveis (default: 6)
  className?: string;                 // Classes CSS adicionais
}
```

---

## 🎨 Estilos Tailwind

```tsx
<textarea
  className={`
    w-full                           // Largura total
    resize-none                      // Desabilita resize manual
    overflow-y-auto                  // Scroll vertical quando necessário
    bg-transparent                   // Fundo transparente
    text-text-main                   // Cor do texto (#1F2329)
    placeholder:text-text-secondary  // Cor do placeholder (#5F5E5A)
    focus:outline-none               // Remove outline padrão
    disabled:opacity-50              // Opacidade 50% quando disabled
    disabled:cursor-not-allowed      // Cursor not-allowed quando disabled
  `}
  style={{
    minHeight: '44px',               // Altura mínima (1 linha)
    maxHeight: '120px',              // Altura máxima (6 linhas)
    lineHeight: '20px',              // Altura de cada linha
  }}
/>
```

---

## 💡 Exemplo de Uso

```tsx
import { useState } from 'react';
import { ComposerInput } from '@/components/composer';

function MyComposer() {
  const [message, setMessage] = useState('');
  
  const handleSend = () => {
    if (!message.trim()) return;
    console.log('Enviando:', message);
    setMessage('');
  };
  
  return (
    <div className="p-4 border rounded-lg">
      <ComposerInput
        value={message}
        onChange={setMessage}
        onSubmit={handleSend}
        placeholder="Digite sua mensagem..."
        maxRows={6}
      />
    </div>
  );
}
```

---

## 🧪 Testando o Componente

### Testar Auto-resize
1. Digite texto curto → Altura = 44px (1 linha)
2. Pressione Shift+Enter 5 vezes → Textarea cresce até 120px
3. Continue adicionando linhas → Scroll interno aparece

### Testar Keyboard Handling
1. Digite "Olá, mundo!" e pressione **Enter** → Envia
2. Digite "Linha 1" + **Shift+Enter** + "Linha 2" → Quebra linha
3. Pressione **Esc** → Remove foco

### Testar Estados
1. Marque checkbox "Desabilitar" → Opacidade 50%, cursor not-allowed
2. Clique no input → onFocus dispara
3. Clique fora → onBlur dispara

---

## ✅ Checklist de Implementação

- [x] Criar `ComposerInput.tsx` com interface TypeScript completa
- [x] Implementar auto-resize com `useRef` + `useEffect`
- [x] Implementar keyboard handling (Enter, Shift+Enter, Escape)
- [x] Adicionar estados disabled, focus, blur
- [x] Adicionar atributos ARIA para acessibilidade
- [x] Atualizar `src/components/composer/index.ts` com export
- [x] Usar `React.memo` para otimização de re-renders
- [x] Garantir TypeScript sem erros
- [x] Passar em ESLint sem warnings
- [x] Criar exemplo interativo (`ComposerInputExample.tsx`)
- [x] Documentar funcionalidades e uso

---

## 🔍 Detalhes Técnicos

### Por que `resize-none`?
Evita que o usuário redimensione manualmente o textarea, garantindo comportamento consistente do auto-resize.

### Por que `rows={1}`?
Define a altura inicial mínima. O auto-resize cresce a partir daqui.

### Por que `style` inline?
`minHeight`, `maxHeight` e `lineHeight` são dinâmicos e dependem de `maxRows`. Tailwind não suporta valores dinâmicos.

### Por que `onChange` recebe string?
Simplifica a API. O componente já extrai `e.target.value` internamente:
```tsx
onChange={(e) => onChange(e.target.value)}
```

### Por que validar `value.trim()`?
Evita enviar mensagens vazias ou com apenas espaços em branco.

---

## 🚀 Próximas Etapas

A CAMADA 3 está completa. Próximos passos da **FASE 5**:

- **CAMADA 4**: `ComposerAttachments` - Chips de anexos/uploads
- **CAMADA 5**: `Composer` - Container principal que integra todos os componentes
- **CAMADA 6**: `ComposerContext` - Gerenciamento de estado global do composer

---

## 📊 Métricas

- **Linhas de código**: ~110 linhas
- **Componentes criados**: 1
- **Exemplos criados**: 1
- **TypeScript**: ✅ Sem erros
- **ESLint**: ✅ Sem warnings
- **Acessibilidade**: ✅ ARIA completo
- **Otimização**: ✅ React.memo aplicado

---

## 📝 Notas de Implementação

1. **Auto-resize** é calculado via `scrollHeight` do textarea
2. **maxRows** define limite superior (default 6 linhas = 120px)
3. **Scroll interno** aparece automaticamente quando ultrapassar maxHeight
4. **Enter** dispara `onSubmit()` apenas se houver texto válido
5. **Shift+Enter** preserva comportamento nativo de quebra de linha
6. **Escape** remove foco (útil para fechar modals ou sidebars)

---

**Implementado por**: Claude Sonnet 4.5  
**Data**: 2026-04-21  
**Status**: ✅ COMPLETO - Pronto para CAMADA 4
