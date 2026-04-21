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
      className={`text-primary-blue hover:text-[#0F428C] border-b-2 border-flag-yellow transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-flag-yellow rounded ${className}`}
      aria-label={`Citação ${number}`}
    >
      {children}
      <sup className="text-primary-blue font-semibold ml-0.5">[{number}]</sup>
    </a>
  );
};

export default React.memo(CitationInline);
