import React from 'react';
import type { Citation } from '../../types/citation';

interface CitationItemProps {
  citation: Citation;
  number: number;
  className?: string;
}

/**
 * Item individual na lista de citações
 * Exibe número, título, breadcrumb e URL da fonte
 */
const CitationItem: React.FC<CitationItemProps> = ({ citation, number, className = '' }) => {
  return (
    <div
      className={`grid gap-3 px-4 py-3 border-b last:border-b-0 cursor-pointer transition-all duration-100 ${className}`}
      style={{
        gridTemplateColumns: '28px 1fr auto',
        alignItems: 'start',
        borderBottomColor: 'var(--line)',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = 'var(--navy-050)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = 'transparent';
      }}
      onClick={() => window.open(citation.url, '_blank')}
    >
      {/* Número */}
      <div
        className="flex items-center justify-center rounded-full mt-0.5"
        style={{
          width: '24px',
          height: '24px',
          backgroundColor: 'var(--navy-700)',
          color: '#fff',
          fontSize: '11.5px',
          fontWeight: '700',
        }}
      >
        {number}
      </div>

      {/* Conteúdo */}
      <div className="flex-1 min-w-0">
        <div
          className="font-semibold leading-tight"
          style={{
            fontSize: '14px',
            color: 'var(--navy-800)',
            lineHeight: '1.3',
          }}
        >
          {citation.title}
        </div>
        <div
          className="mt-0.5"
          style={{
            fontSize: '11.5px',
            color: 'var(--ink-500)',
          }}
        >
          {citation.breadcrumb?.split(' › ').map((crumb, i, arr) => (
            <React.Fragment key={i}>
              <span>{crumb}</span>
              {i < arr.length - 1 && <span style={{ color: 'var(--ink-300)' }}> › </span>}
            </React.Fragment>
          ))}
        </div>
        <div
          className="mt-1 truncate"
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '11.5px',
            color: 'var(--green-700)',
            wordBreak: 'break-all',
          }}
        >
          {citation.url}
        </div>
      </div>

      {/* Botão Abrir */}
      <button
        className="self-center px-2.5 py-1.5 rounded-lg border transition-all duration-150 whitespace-nowrap focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-navy-300"
        style={{
          fontSize: '11px',
          color: 'var(--navy-700)',
          fontWeight: '600',
          letterSpacing: '0.04em',
          textTransform: 'uppercase',
          borderColor: 'var(--navy-100)',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = 'var(--navy-050)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = 'transparent';
        }}
        onClick={(e) => {
          e.stopPropagation();
          window.open(citation.url, '_blank');
        }}
      >
        Abrir →
      </button>
    </div>
  );
};

export default React.memo(CitationItem);
