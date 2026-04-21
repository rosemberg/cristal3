import React from 'react';
import { DiamondIcon } from '../icons';

interface AvatarProps {
  type: 'user' | 'assistant';
  className?: string;
}

/**
 * Avatar reutilizável para usuário e assistente
 * - Usuário: círculo azul com "VC"
 * - Assistente: círculo branco com borda amarela e ícone DiamondIcon
 */
const Avatar: React.FC<AvatarProps> = ({
  type,
  className = ''
}) => {
  if (type === 'user') {
    return (
      <div
        style={{
          width: '32px',
          height: '32px',
          borderRadius: '50%',
          backgroundColor: '#3b82f6',
          color: 'white',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '12px',
          fontWeight: '600',
          flexShrink: 0
        }}
        className={className}
        aria-label="Avatar do usuário"
      >
        VC
      </div>
    );
  }

  return (
    <div
      style={{
        width: '32px',
        height: '32px',
        borderRadius: '50%',
        backgroundColor: 'white',
        border: '2px solid #FFCD07',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexShrink: 0
      }}
      className={className}
      aria-label="Avatar do assistente Cristal"
    >
      <span style={{ fontSize: '16px' }}>💎</span>
    </div>
  );
};

export default React.memo(Avatar);
