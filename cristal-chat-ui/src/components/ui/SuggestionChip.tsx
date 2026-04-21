import React from 'react';

interface SuggestionChipProps {
  text: string;
  number?: number;
  onClick: () => void;
  className?: string;
}

/**
 * Chip clicável de sugestão
 * Usado no card de boas-vindas para perguntas pré-definidas
 */
const SuggestionChip: React.FC<SuggestionChipProps> = ({
  text,
  number,
  onClick,
  className = ''
}) => {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`
        inline-flex items-center gap-2
        px-3.5 py-2.5 rounded-full
        transition-all duration-150
        focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-navy-300
        cursor-pointer
        ${className}
      `}
      style={{
        backgroundColor: 'var(--navy-050)',
        color: 'var(--navy-800)',
        border: '1px solid var(--navy-100)',
        fontSize: '13px',
        fontWeight: '500',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = 'var(--navy-100)';
        e.currentTarget.style.transform = 'translateY(-1px)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = 'var(--navy-050)';
        e.currentTarget.style.transform = 'translateY(0)';
      }}
      role="button"
    >
      {number && (
        <span
          className="inline-flex items-center justify-center rounded px-1.5 py-0.5 border"
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '10.5px',
            color: 'var(--navy-600)',
            backgroundColor: '#fff',
            borderColor: 'var(--navy-100)',
          }}
        >
          {number}
        </span>
      )}
      <span>{text}</span>
    </button>
  );
};

export default React.memo(SuggestionChip);
