import React from 'react';
import { SendIcon } from '../icons';

interface SendButtonProps {
  onClick: () => void;
  disabled?: boolean;
  isLoading?: boolean;
  className?: string;
}

const SendButton: React.FC<SendButtonProps> = React.memo(({
  onClick,
  disabled = false,
  isLoading = false,
  className = ''
}) => {
  const isDisabled = disabled || isLoading;

  return (
    <button
      onClick={onClick}
      disabled={isDisabled}
      aria-label="Enviar mensagem"
      className={`
        w-10 h-10
        rounded-full
        flex items-center justify-center
        transition-all duration-200
        focus:outline-none
        focus-visible:ring-2 focus-visible:ring-flag-yellow
        ${isDisabled
          ? 'bg-border-subtle cursor-not-allowed'
          : 'bg-primary-blue hover:bg-[#0F428C]'
        }
        ${className}
      `.trim().replace(/\s+/g, ' ')}
    >
      <SendIcon
        className="text-white"
        size={20}
      />
    </button>
  );
});

SendButton.displayName = 'SendButton';

export default SendButton;
