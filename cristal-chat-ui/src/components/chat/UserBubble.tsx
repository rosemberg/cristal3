import React from 'react';
import Avatar from '../ui/Avatar';
import { formatTime } from '../../utils/formatTime';

interface UserBubbleProps {
  content: string;
  timestamp: Date;
  className?: string;
}

/**
 * Bolha de mensagem do usuário
 * Fundo azul, texto branco, alinhada à direita com avatar "VC"
 */
const UserBubble: React.FC<UserBubbleProps> = ({
  content,
  timestamp,
  className = ''
}) => {
  return (
    <div
      className={`flex justify-end gap-3 mb-[22px] max-w-[880px] mx-auto ${className}`}
    >
      <div className="flex flex-col items-end" style={{ maxWidth: '560px' }}>
        <div
          className="px-[18px] py-[13px] shadow-[0_2px_0_rgba(6,26,68,0.12)]"
          style={{
            backgroundColor: 'var(--navy-800)',
            color: '#fff',
            borderRadius: 'var(--radius-bubble)',
            borderBottomRightRadius: '6px',
            fontSize: '14.5px',
            lineHeight: '1.45',
            fontWeight: '500',
          }}
        >
          <p style={{ margin: 0 }}>{content}</p>
        </div>
      </div>
      <Avatar type="user" />
    </div>
  );
};

export default React.memo(UserBubble);
