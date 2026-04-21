import React from 'react';

interface LoadingDotsProps {
  text?: string;
  className?: string;
}

/**
 * Animação de loading com três pontos pulsantes
 * Usado enquanto o assistente está processando uma resposta
 */
const LoadingDots: React.FC<LoadingDotsProps> = ({
  text = 'Consultando o portal...',
  className = ''
}) => {
  return (
    <div
      className={`flex flex-col items-start gap-2 py-4 ${className}`}
      role="status"
      aria-label="Carregando resposta"
    >
      <div className="flex items-center gap-1">
        <span className="w-2 h-2 rounded-full bg-primary-blue animate-pulse-dot" style={{ animationDelay: '0s' }} />
        <span className="w-2 h-2 rounded-full bg-primary-blue animate-pulse-dot" style={{ animationDelay: '0.2s' }} />
        <span className="w-2 h-2 rounded-full bg-primary-blue animate-pulse-dot" style={{ animationDelay: '0.4s' }} />
      </div>
      {text && (
        <span className="text-sm text-text-secondary">
          {text}
        </span>
      )}
    </div>
  );
};

export default React.memo(LoadingDots);
