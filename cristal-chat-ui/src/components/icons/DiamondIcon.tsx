import React from 'react';

interface DiamondIconProps {
  className?: string;
  size?: number;
}

const DiamondIcon: React.FC<DiamondIconProps> = ({
  className = '',
  size = 24
}) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden="true"
    >
      {/* Diamante facetado - cristal geométrico */}
      <path
        d="M12 2L4 9L12 22L20 9L12 2Z"
        fill="currentColor"
        opacity="0.9"
      />
      {/* Facetas internas */}
      <path
        d="M12 2L8 9H16L12 2Z"
        fill="currentColor"
        opacity="0.7"
      />
      <path
        d="M4 9L12 13L20 9H4Z"
        fill="currentColor"
        opacity="0.5"
      />
      <path
        d="M12 13L8 9L12 22L12 13Z"
        fill="currentColor"
        opacity="0.3"
      />
      <path
        d="M12 13L16 9L12 22L12 13Z"
        fill="currentColor"
        opacity="0.3"
      />
    </svg>
  );
};

export default DiamondIcon;
