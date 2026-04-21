import React from 'react';

/**
 * Barra divisória amarela de 3px
 * Representa a bandeira brasileira na identidade visual TRE-PI
 */
const YellowDivider: React.FC = () => {
  return (
    <div
      style={{
        width: '100%',
        height: '3px',
        backgroundColor: '#FFCD07'
      }}
      role="separator"
      aria-hidden="true"
    />
  );
};

export default YellowDivider;
