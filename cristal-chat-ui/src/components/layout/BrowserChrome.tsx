import React from 'react';

/**
 * Barra de navegador fictícia
 * Simula uma janela do navegador para dar contexto de iframe
 */
const BrowserChrome: React.FC = () => {
  return (
    <div style={{
      backgroundColor: '#f3f4f6',
      borderBottom: '1px solid #d1d5db',
      padding: '8px 16px',
      display: 'flex',
      alignItems: 'center',
      gap: '12px'
    }}>
      {/* Traffic lights */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
        <div style={{ width: '12px', height: '12px', borderRadius: '50%', backgroundColor: '#ef4444' }} />
        <div style={{ width: '12px', height: '12px', borderRadius: '50%', backgroundColor: '#eab308' }} />
        <div style={{ width: '12px', height: '12px', borderRadius: '50%', backgroundColor: '#22c55e' }} />
      </div>

      {/* URL bar */}
      <div style={{
        flex: 1,
        backgroundColor: 'white',
        borderRadius: '4px',
        padding: '4px 12px',
        fontSize: '14px',
        color: '#4b5563',
        fontFamily: 'monospace'
      }}>
        www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/cristal
      </div>
    </div>
  );
};

export default BrowserChrome;
