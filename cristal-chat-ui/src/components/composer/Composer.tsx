import React, { useState, useCallback, memo } from 'react';
import SendButton from './SendButton';
import ComposerToolbar from './ComposerToolbar';
import ComposerMeta from './ComposerMeta';
import ComposerInput from './ComposerInput';

export interface ComposerProps {
  onSendMessage: (message: string) => void;
  onClearChat?: () => void;
  isDisabled?: boolean;
  placeholder?: string;
  showMetaBar?: boolean;
  className?: string;
}

const Composer: React.FC<ComposerProps> = memo(({
  onSendMessage,
  onClearChat,
  isDisabled = false,
  placeholder = 'Digite sua pergunta...',
  showMetaBar = false,
  className = '',
}) => {
  const [inputValue, setInputValue] = useState<string>('');
  const [isFocused, setIsFocused] = useState<boolean>(false);

  const handleSubmit = useCallback(() => {
    if (inputValue.trim() && !isDisabled) {
      onSendMessage(inputValue.trim());
      setInputValue(''); // Limpar input após enviar
    }
  }, [inputValue, isDisabled, onSendMessage]);

  const handleClear = useCallback(() => {
    if (window.confirm('Deseja limpar toda a conversa?')) {
      onClearChat?.();
      setInputValue('');
    }
  }, [onClearChat]);

  const handleAttach = useCallback(() => {
    console.log('Anexar arquivo (implementar na Fase 7)');
  }, []);

  const handleMic = useCallback(() => {
    console.log('Gravar áudio (implementar na Fase 8)');
  }, []);

  return (
    <div className={`bg-surface border-t border-line ${className}`} style={{ paddingTop: '16px', paddingBottom: '18px' }}>
      {/* Composer pill container */}
      <div className="max-w-[880px] mx-auto px-6">
        {/* Composer pill */}
        <div
          className={`
            bg-white border rounded-full
            px-[18px] py-[6px]
            flex items-center gap-[6px]
            transition-all duration-150
            ${isFocused ? 'shadow-[0_0_0_4px_rgba(44,85,199,0.08)]' : 'shadow-[0_1px_2px_rgba(6,26,68,0.04)]'}
            ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''}
          `}
          style={{
            borderColor: isFocused ? 'var(--navy-300)' : 'var(--line-2)',
          }}
        >
          {/* Toolbar à esquerda */}
          <div className="flex-shrink-0">
            <ComposerToolbar
              onAttachClick={handleAttach}
              onMicClick={handleMic}
            />
          </div>

          {/* Input expansível no centro */}
          <div className="flex-1 min-w-0">
            <ComposerInput
              value={inputValue}
              onChange={setInputValue}
              onSubmit={handleSubmit}
              onFocus={() => setIsFocused(true)}
              onBlur={() => setIsFocused(false)}
              disabled={isDisabled}
              placeholder={placeholder}
            />
          </div>

          {/* Botão de enviar à direita */}
          <div className="flex-shrink-0">
            <SendButton
              onClick={handleSubmit}
              disabled={!inputValue.trim() || isDisabled}
            />
          </div>
        </div>

        {/* Meta bar (condicional, abaixo do composer) */}
        {showMetaBar && (
          <div className="mt-[10px]">
            <ComposerMeta
              onClearClick={onClearChat ? handleClear : undefined}
              showClearButton={!!onClearChat}
            />
          </div>
        )}
      </div>
    </div>
  );
});

Composer.displayName = 'Composer';

export default Composer;
