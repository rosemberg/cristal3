import React from 'react';
import { AttachIcon, MicIcon } from '../icons';

interface ComposerToolbarProps {
  onAttachClick?: () => void;
  onMicClick?: () => void;
  className?: string;
}

const ComposerToolbar: React.FC<ComposerToolbarProps> = React.memo(({
  onAttachClick,
  onMicClick,
  className = '',
}) => {
  return (
    <div className={`flex items-center gap-0 ${className}`}>
      <button
        type="button"
        onClick={onAttachClick}
        className="w-9 h-9 flex items-center justify-center rounded-full transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-navy-300"
        style={{ color: 'var(--ink-500)' }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = 'var(--navy-050)';
          e.currentTarget.style.color = 'var(--navy-800)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = 'transparent';
          e.currentTarget.style.color = 'var(--ink-500)';
        }}
        aria-label="Anexar arquivo"
        title="Anexar"
      >
        <AttachIcon size={18} />
      </button>

      <button
        type="button"
        onClick={onMicClick}
        className="w-9 h-9 flex items-center justify-center rounded-full transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-navy-300"
        style={{ color: 'var(--ink-500)' }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = 'var(--navy-050)';
          e.currentTarget.style.color = 'var(--navy-800)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = 'transparent';
          e.currentTarget.style.color = 'var(--ink-500)';
        }}
        aria-label="Gravar áudio"
        title="Ditar por voz"
      >
        <MicIcon size={18} />
      </button>
    </div>
  );
});

ComposerToolbar.displayName = 'ComposerToolbar';

export default ComposerToolbar;
