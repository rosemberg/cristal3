import React, { useRef, useEffect } from 'react';

interface ComposerInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  onFocus?: () => void;
  onBlur?: () => void;
  placeholder?: string;
  disabled?: boolean;
  maxRows?: number;
  className?: string;
}

const ComposerInput: React.FC<ComposerInputProps> = React.memo(({
  value,
  onChange,
  onSubmit,
  onFocus,
  onBlur,
  placeholder = 'Digite sua pergunta...',
  disabled = false,
  maxRows = 6,
  className = '',
}) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto-resize: ajusta altura dinamicamente baseado no conteúdo
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    // Reset height para calcular scrollHeight correto
    textarea.style.height = 'auto';

    // Calcular nova altura (limitada por maxRows)
    const lineHeight = 20; // aproximadamente
    const maxHeight = maxRows * lineHeight;
    const newHeight = Math.min(textarea.scrollHeight, maxHeight);

    textarea.style.height = `${newHeight}px`;
  }, [value, maxRows]);

  // Keyboard handling: Enter para enviar, Shift+Enter para quebrar linha
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (value.trim() && !disabled) {
        onSubmit();
      }
    }

    if (e.key === 'Escape') {
      e.currentTarget.blur();
    }
  };

  return (
    <textarea
      ref={textareaRef}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onKeyDown={handleKeyDown}
      onFocus={onFocus}
      onBlur={onBlur}
      placeholder={placeholder}
      disabled={disabled}
      rows={1}
      aria-label="Campo de mensagem"
      aria-multiline="true"
      aria-disabled={disabled}
      className={`
        w-full
        resize-none
        overflow-y-auto
        bg-transparent
        text-text-main
        placeholder:text-text-secondary
        focus:outline-none
        disabled:opacity-50
        disabled:cursor-not-allowed
        ${className}
      `}
      style={{
        minHeight: '44px',
        maxHeight: `${maxRows * 20}px`,
        lineHeight: '20px',
      }}
    />
  );
});

ComposerInput.displayName = 'ComposerInput';

export default ComposerInput;
