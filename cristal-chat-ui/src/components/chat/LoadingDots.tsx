import React from 'react';
import Avatar from '../ui/Avatar';

interface LoadingDotsProps {
  text?: string;
  className?: string;
}

/**
 * Animação de loading com três pontos pulsantes
 * Usado enquanto o assistente está processando uma resposta
 */
const LoadingDots: React.FC<LoadingDotsProps> = ({
  text = 'Consultando o portal',
  className = ''
}) => {
  return (
    <div
      className={`flex gap-3 mb-[22px] max-w-[880px] mx-auto ${className}`}
      role="status"
      aria-label="Carregando resposta"
    >
      <Avatar type="assistant" />
      <div className="flex items-center gap-2.5" style={{ fontSize: '14.5px', color: 'var(--ink-500)' }}>
        <span>{text}</span>
        <span className="inline-flex gap-1">
          <span className="w-1.5 h-1.5 rounded-full animate-pulse-dot" style={{ backgroundColor: 'var(--ink-300)', animationDelay: '0s' }} />
          <span className="w-1.5 h-1.5 rounded-full animate-pulse-dot" style={{ backgroundColor: 'var(--ink-300)', animationDelay: '0.15s' }} />
          <span className="w-1.5 h-1.5 rounded-full animate-pulse-dot" style={{ backgroundColor: 'var(--ink-300)', animationDelay: '0.3s' }} />
        </span>
      </div>
    </div>
  );
};

export default React.memo(LoadingDots);
