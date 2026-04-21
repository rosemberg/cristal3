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
    <li className={`flex gap-3 ${className}`}>
      {/* Número */}
      <div className="flex-shrink-0 w-6 h-6 rounded-full bg-primary-blue text-white flex items-center justify-center text-xs font-semibold">
        {number}
      </div>

      {/* Conteúdo */}
      <div className="flex-1 min-w-0">
        <a
          href={citation.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm font-medium text-primary-blue hover:text-[#0F428C] hover:underline transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-flag-yellow rounded"
        >
          {citation.title}
        </a>
        <p className="text-xs text-text-secondary mt-0.5">
          {citation.breadcrumb}
        </p>
        <p className="text-xs text-urn-green font-mono mt-1 truncate">
          {citation.url}
        </p>
      </div>
    </li>
  );
};

export default React.memo(CitationItem);
