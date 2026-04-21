import React from 'react';

/**
 * Header institucional do TRE-PI
 * Contém branding, acessibilidade e CTA da Ouvidoria
 */
const Header: React.FC = () => {
  return (
    <header style={{
      backgroundColor: '#1e3a8a',
      color: 'white',
      padding: '16px 24px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between'
    }}>
      {/* Logo + Título */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <div style={{
          width: '48px',
          height: '48px',
          borderRadius: '50%',
          border: '3px solid #FFCD07',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '24px'
        }}>
          💎
        </div>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: '700', margin: 0 }}>Cristal</h1>
          <p style={{ fontSize: '13px', color: '#cbd5e1', margin: 0, marginTop: '2px' }}>
            Inteligência e clareza nos dados públicos do TRE-PI
          </p>
        </div>
      </div>

      {/* Ações */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        {/* Ícone lua (dark mode) */}
        <button
          style={{
            width: '40px',
            height: '40px',
            background: 'rgba(255,255,255,0.1)',
            color: 'white',
            border: '1px solid rgba(255,255,255,0.2)',
            borderRadius: '6px',
            cursor: 'pointer',
            fontSize: '18px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}
          aria-label="Modo escuro"
        >
          🌙
        </button>

        <button
          style={{
            padding: '8px 14px',
            fontSize: '14px',
            background: 'rgba(255,255,255,0.1)',
            color: 'white',
            border: '1px solid rgba(255,255,255,0.2)',
            borderRadius: '6px',
            cursor: 'pointer',
            fontWeight: '600'
          }}
          aria-label="Alto contraste"
        >
          A+
        </button>

        <a
          href="https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/ouvidoria"
          target="_blank"
          rel="noopener noreferrer"
          style={{
            padding: '10px 20px',
            backgroundColor: '#FFCD07',
            color: '#1e3a8a',
            fontSize: '14px',
            fontWeight: '700',
            borderRadius: '6px',
            textDecoration: 'none',
            display: 'inline-block'
          }}
        >
          Ouvidoria / SIC
        </a>
      </div>
    </header>
  );
};

export default Header;
