import React from 'react';

/**
 * Footer institucional do TRE-PI
 * Contém copyright e links importantes
 */
const Footer: React.FC = () => {
  return (
    <footer style={{
      backgroundColor: '#1e3a8a',
      color: 'white',
      padding: '16px 24px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      fontSize: '13px',
      flexWrap: 'wrap',
      gap: '8px'
    }}>
      <p style={{ color: '#cbd5e1', margin: 0 }}>
        © TRE-PI
      </p>

      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        <a
          href="https://www.tre-pi.jus.br/politica-de-privacidade"
          target="_blank"
          rel="noopener noreferrer"
          style={{ color: '#FFCD07', textDecoration: 'none' }}
        >
          Política de privacidade
        </a>
        <span style={{ color: '#cbd5e1' }}>·</span>
        <a
          href="https://www.tre-pi.jus.br/ouvidoria"
          target="_blank"
          rel="noopener noreferrer"
          style={{ color: '#FFCD07', textDecoration: 'none' }}
        >
          Ouvidoria
        </a>
        <span style={{ color: '#cbd5e1' }}>·</span>
        <a
          href="https://www.tre-pi.jus.br/lgpd"
          target="_blank"
          rel="noopener noreferrer"
          style={{ color: '#FFCD07', textDecoration: 'none' }}
        >
          LGPD
        </a>
      </div>
    </footer>
  );
};

export default Footer;
