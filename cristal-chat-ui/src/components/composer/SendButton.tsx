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
      title="Enviar (Enter)"
      className={`
        w-10 h-10
        rounded-full
        flex items-center justify-center
        transition-all duration-150
        focus:outline-none
        focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-navy-300
        ${isDisabled
          ? 'bg-navy-200 cursor-not-allowed'
          : 'bg-navy-700 hover:bg-navy-800 hover:-translate-y-0.5 active:translate-y-0'
        }
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      style={{
        backgroundColor: isDisabled ? 'var(--navy-200)' : 'var(--navy-700)',
      }}
      onMouseEnter={(e) => {
        if (!isDisabled) {
          e.currentTarget.style.backgroundColor = 'var(--navy-800)';
        }
      }}
      onMouseLeave={(e) => {
        if (!isDisabled) {
          e.currentTarget.style.backgroundColor = 'var(--navy-700)';
        }
      }}
    >
      <SendIcon
        className="text-white"
        size={18}
      />
    </button>
  );
});

SendButton.displayName = 'SendButton';

export default SendButton;
