import React from 'react';
import SuggestionChip from '../ui/SuggestionChip';
import { DiamondIcon } from '../icons';

interface WelcomeCardProps {
  onSuggestionClick: (suggestion: string) => void;
  className?: string;
}

/**
 * Card de boas-vindas exibido quando não há mensagens
 * Apresenta o Cristal e oferece sugestões de perguntas
 */
const WelcomeCard: React.FC<WelcomeCardProps> = ({
  onSuggestionClick,
  className = ''
}) => {
  const suggestions = [
    'Como contestar uma multa?',
    'Quais diárias foram pagas?',
    'Onde vejo as licitações?'
  ];

  return (
    <div className={`max-w-3xl mx-auto px-4 sm:px-0 ${className}`}>
      <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-6 sm:p-10">
        {/* Ícone + Título */}
        <div className="flex flex-col items-center text-center mb-8">
          {/* Ícone com círculo azul e borda amarela */}
          <div style={{
            width: '100px',
            height: '100px',
            borderRadius: '50%',
            backgroundColor: '#1e3a8a',
            border: '4px solid #FFCD07',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '24px',
            fontSize: '48px'
          }}>
            💎
          </div>

          <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 mb-3" style={{ color: '#1e3a8a' }}>
            Olá! Sou a Cristal. Sobre o que você quer saber do portal da transparência?
          </h2>
          <p className="text-base text-gray-600 max-w-2xl">
            Converse comigo em linguagem natural. Respondo com base nas páginas oficiais do TRE-PI e indico onde você encontra cada informação.
          </p>
        </div>

        {/* Sugestões */}
        <div className="flex flex-wrap justify-center gap-3">
          {suggestions.map((suggestion, idx) => (
            <SuggestionChip
              key={idx}
              text={suggestion}
              onClick={() => onSuggestionClick(suggestion)}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

export default React.memo(WelcomeCard);
