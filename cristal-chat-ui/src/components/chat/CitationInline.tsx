import React from 'react';

interface CitationInlineProps {
  href: string;
  number: string | number;
  children: React.ReactNode;
  className?: string;
}

/**
 * Link inline com superscript numérico
 * Usado para referenciar citações no texto da resposta do assistente
 */
const CitationInline: React.FC<CitationInlineProps> = ({
  href,
  number,
  children,
  className = ''
}) => {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={`cursor-pointer transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-navy-300 rounded ${className}`}
      style={{
        color: 'var(--navy-700)',
        fontWeight: '500',
        textDecoration: 'underline',
        textDecorationColor: 'var(--gold-400)',
        textDecorationThickness: '2px',
        textUnderlineOffset: '3px',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.color = 'var(--navy-900)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.color = 'var(--navy-700)';
      }}
      aria-label={`Citação ${number}`}
    >
      {children}
      <sup
        className="ml-0.5"
        style={{
          color: 'var(--navy-600)',
          fontSize: '11px',
          fontFamily: 'var(--font-mono)',
        }}
      >
        {number}
      </sup>
    </a>
  );
};

export default React.memo(CitationInline);
