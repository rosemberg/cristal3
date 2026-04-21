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
    <div className={`bg-card-bg border-t border-border-subtle ${className}`}>
      {/* Meta bar (condicional) */}
      {showMetaBar && (
        <div className="px-3 sm:px-4 pt-3 pb-2">
          <ComposerMeta
            onClearClick={onClearChat ? handleClear : undefined}
            showClearButton={!!onClearChat}
          />
        </div>
      )}

      {/* Input área com border pill */}
      <div className={`
        mx-3 sm:mx-4 mb-3 sm:mb-4 px-3 sm:px-4 py-3
        ${showMetaBar ? 'mt-0' : 'mt-3 sm:mt-4'}
        border-2 rounded-pill
        flex items-end gap-2 sm:gap-3
        transition-all duration-200
        ${isFocused ? 'border-primary-blue shadow-md' : 'border-border-subtle'}
        ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''}
      `}>
        {/* Toolbar à esquerda */}
        <div className="flex-shrink-0 self-end pb-0.5">
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
        <div className="flex-shrink-0 self-end pb-0.5">
          <SendButton
            onClick={handleSubmit}
            disabled={!inputValue.trim() || isDisabled}
          />
        </div>
      </div>
    </div>
  );
});

Composer.displayName = 'Composer';

export default Composer;
