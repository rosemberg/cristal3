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
    'Como contestar uma multa eleitoral?',
    'Quais diárias foram pagas em março?',
    'Onde vejo as licitações em andamento?'
  ];

  return (
    <section className={`max-w-[880px] mx-auto px-7 ${className}`}>
      <div
        className="relative overflow-hidden bg-white rounded-3xl p-8 sm:p-9 text-center border"
        style={{
          borderColor: 'var(--line)',
          borderRadius: 'var(--radius-xl)',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        {/* Gradient background effect */}
        <div
          className="absolute inset-x-0 bottom-0 pointer-events-none"
          style={{
            height: '120px',
            background: 'radial-gradient(ellipse at 50% 0%, rgba(245,197,24,0.18), transparent 60%)',
            left: '-40px',
            right: '-40px',
            bottom: '-60px',
          }}
        />

        {/* Diamond Icon */}
        <div
          className="inline-flex items-center justify-center rounded-full mb-[18px] relative"
          style={{
            width: '78px',
            height: '78px',
            backgroundColor: 'var(--navy-800)',
            border: '3px solid var(--gold-400)',
          }}
        >
          <DiamondIcon size={38} style={{ color: '#fff' }} />
        </div>

        <h2
          className="font-medium mb-1.5 mx-auto max-w-[680px]"
          style={{
            fontFamily: 'var(--font-display)',
            fontSize: '26px',
            color: 'var(--navy-800)',
            letterSpacing: '-0.01em',
            lineHeight: '1.25',
          }}
        >
          Olá! Sou a Cristal. Sobre o que você quer saber do portal da transparência?
        </h2>
        <p
          className="mx-auto max-w-[560px] mb-[22px]"
          style={{
            color: 'var(--ink-500)',
            fontSize: '14.5px',
            lineHeight: '1.55',
          }}
        >
          Converse comigo em linguagem natural. Respondo com base nas páginas oficiais do portal e indico exatamente onde cada informação está publicada.
        </p>

        {/* Sugestões */}
        <div className="flex flex-wrap justify-center gap-2.5 relative">
          {suggestions.map((suggestion, idx) => (
            <SuggestionChip
              key={idx}
              text={suggestion}
              number={idx + 1}
              onClick={() => onSuggestionClick(suggestion)}
            />
          ))}
        </div>
      </div>
    </section>
  );
};

export default React.memo(WelcomeCard);
