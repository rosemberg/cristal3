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
    <div className={`bg-card-bg rounded-lg border border-border-subtle p-4 ${className}`}>
      <h3 className="text-xs font-semibold text-text-secondary uppercase tracking-wide mb-3">
        Páginas Citadas
      </h3>
      <ol className="space-y-3">
        {citations.map((citation, idx) => (
          <CitationItem key={citation.id} citation={citation} number={idx + 1} />
        ))}
      </ol>
    </div>
  );
};

export default React.memo(CitationsBlock);
