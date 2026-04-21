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
    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', marginBottom: '16px' }} className={className}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', maxWidth: '80%' }}>
        <div style={{
          backgroundColor: '#3b82f6',
          color: 'white',
          padding: '12px 16px',
          borderRadius: '16px',
          borderBottomRightRadius: '4px',
          wordBreak: 'break-word'
        }}>
          <p style={{ fontSize: '14px', lineHeight: '1.6', margin: 0 }}>{content}</p>
        </div>
        <span style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>
          {formatTime(timestamp)}
        </span>
      </div>
      <Avatar type="user" />
    </div>
  );
};

export default React.memo(UserBubble);
