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
  infoText = 'Baseado em páginas oficiais do TRE-PI',
  className = '',
}) => {
  return (
    <div className={`flex items-center justify-between gap-4 text-xs text-text-secondary ${className}`}>
      <div className="flex items-center gap-1.5">
        <InfoIcon size={16} className="text-text-secondary flex-shrink-0" />
        <span>{infoText}</span>
      </div>

      {showClearButton && onClearClick && (
        <button
          type="button"
          onClick={onClearClick}
          className="flex items-center gap-1 px-2 py-1 hover:bg-chat-bg rounded transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-flag-yellow"
          aria-label="Limpar conversa"
        >
          <TrashIcon size={14} />
          <span>Limpar</span>
        </button>
      )}
    </div>
  );
});

ComposerMeta.displayName = 'ComposerMeta';

export default ComposerMeta;
