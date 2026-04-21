import React from 'react';

interface IconProps {
  className?: string;
  size?: number;
}

const SendIcon: React.FC<IconProps> = ({ className = '', size = 24 }) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden="true"
    >
      {/* Avião de papel inclinado 45° para nordeste */}
      <line x1="22" y1="2" x2="11" y2="13" />
      <path d="M22 2L15 22L11 13L2 9L22 2Z" />
    </svg>
  );
};

export default SendIcon;
