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
          width: '30px',
          height: '30px',
          borderRadius: '50%',
          backgroundColor: 'var(--ink-300)',
          color: '#fff',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '11px',
          fontWeight: '700',
          letterSpacing: '0.04em',
          flexShrink: 0,
          marginTop: '4px',
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
        width: '34px',
        height: '34px',
        borderRadius: '50%',
        backgroundColor: 'var(--navy-800)',
        border: '2px solid var(--gold-400)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexShrink: 0,
      }}
      className={className}
      aria-label="Avatar do assistente Cristal"
    >
      <DiamondIcon size={17} style={{ color: '#fff' }} />
    </div>
  );
};

export default React.memo(Avatar);
