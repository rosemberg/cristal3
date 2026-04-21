import React from 'react';

interface SuggestionChipProps {
  text: string;
  onClick: () => void;
  className?: string;
}

/**
 * Chip clicável de sugestão
 * Usado no card de boas-vindas para perguntas pré-definidas
 */
const SuggestionChip: React.FC<SuggestionChipProps> = ({
  text,
  onClick,
  className = ''
}) => {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`
        w-full px-4 py-3 text-left
        bg-pale-blue-bg border border-border-subtle
        rounded-lg text-sm text-text-main
        hover:bg-primary-blue/10 hover:border-primary-blue hover:shadow-sm
        transition-all duration-200
        focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-flag-yellow
        cursor-pointer
        ${className}
      `}
      role="button"
    >
      {text}
    </button>
  );
};

export default React.memo(SuggestionChip);
