import React from 'react';
import CitationItem from './CitationItem';
import type { Citation } from '../../types/citation';

interface CitationsBlockProps {
  citations: Citation[];
  className?: string;
}

/**
 * Bloco "PÁGINAS CITADAS"
 * Renderiza lista de referências usadas na resposta
 */
const CitationsBlock: React.FC<CitationsBlockProps> = ({
  citations,
  className = ''
}) => {
  if (!citations || citations.length === 0) {
    return null;
  }

  return (
    <div
      className={`bg-white rounded-2xl overflow-hidden border ${className}`}
      style={{ borderColor: 'var(--line)' }}
    >
      <div
        className="flex items-center gap-2 px-4 py-2.5 border-b"
        style={{
          backgroundColor: 'var(--paper)',
          borderBottomColor: 'var(--line)',
        }}
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" width="13" height="13" style={{ color: 'var(--ink-500)' }}>
          <path d="M4 5a2 2 0 0 1 2-2h12v16H6a2 2 0 0 0-2 2V5z" />
          <path d="M4 19a2 2 0 0 1 2-2h12" />
        </svg>
        <h3
          className="uppercase tracking-wide"
          style={{
            fontSize: '11px',
            fontWeight: '700',
            color: 'var(--ink-500)',
            letterSpacing: '0.14em',
          }}
        >
          Páginas citadas no portal
        </h3>
      </div>
      <div>
        {citations.map((citation, idx) => (
          <CitationItem key={citation.id} citation={citation} number={idx + 1} />
        ))}
      </div>
    </div>
  );
};

export default React.memo(CitationsBlock);
