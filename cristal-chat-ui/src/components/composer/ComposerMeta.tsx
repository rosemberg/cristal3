import React from 'react';
import { InfoIcon, TrashIcon } from '../icons';

interface ComposerMetaProps {
  onClearClick?: () => void;
  showClearButton?: boolean;
  infoText?: string;
  className?: string;
}

const ComposerMeta: React.FC<ComposerMetaProps> = React.memo(({
  onClearClick,
  showClearButton = false,
  infoText = 'Baseado em páginas oficiais do portal',
  className = '',
}) => {
  return (
    <div className={`flex items-center justify-between gap-4 px-1.5 ${className}`} style={{ fontSize: '12.5px', color: 'var(--ink-500)' }}>
      <div className="flex items-center gap-1.5">
        <InfoIcon size={13} style={{ color: 'var(--ink-500)' }} className="flex-shrink-0" />
        <span>{infoText}</span>
      </div>

      {showClearButton && onClearClick && (
        <button
          type="button"
          onClick={onClearClick}
          className="flex items-center gap-1.5 px-2 py-1 rounded-lg transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-navy-300"
          style={{ fontSize: '12.5px', color: 'var(--ink-500)' }}
          onMouseEnter={(e) => {
            e.currentTarget.style.backgroundColor = 'var(--navy-050)';
            e.currentTarget.style.color = 'var(--navy-800)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = 'transparent';
            e.currentTarget.style.color = 'var(--ink-500)';
          }}
          aria-label="Limpar conversa"
        >
          <TrashIcon size={13} />
          <span>Limpar</span>
        </button>
      )}
    </div>
  );
});

ComposerMeta.displayName = 'ComposerMeta';

export default ComposerMeta;
