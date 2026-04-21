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
    <div className={`flex items-center gap-2 ${className}`}>
      <button
        type="button"
        onClick={onAttachClick}
        className="p-2 text-text-secondary hover:text-text-main hover:bg-chat-bg rounded-lg transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-flag-yellow"
        aria-label="Anexar arquivo"
        title="Anexar arquivo"
      >
        <AttachIcon size={20} />
      </button>

      <button
        type="button"
        onClick={onMicClick}
        className="p-2 text-text-secondary hover:text-text-main hover:bg-chat-bg rounded-lg transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-flag-yellow"
        aria-label="Gravar áudio"
        title="Gravar áudio"
      >
        <MicIcon size={20} />
      </button>
    </div>
  );
});

ComposerToolbar.displayName = 'ComposerToolbar';

export default ComposerToolbar;
