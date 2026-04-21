import React from 'react';

interface IconProps {
  className?: string;
  size?: number;
}

const AttachIcon: React.FC<IconProps> = ({ className = '', size = 24 }) => {
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
      {/* Clipe de papel simples */}
      <path d="M21.44 11.05L12.25 20.24C11.1242 21.3658 9.59723 21.9983 8.005 21.9983C6.41277 21.9983 4.88579 21.3658 3.76 20.24C2.63421 19.1142 2.00166 17.5872 2.00166 15.995C2.00166 14.4028 2.63421 12.8758 3.76 11.75L12.33 3.18C13.0806 2.42944 14.0992 2.00662 15.1625 2.00662C16.2258 2.00662 17.2444 2.42944 17.995 3.18C18.7456 3.93056 19.1684 4.94918 19.1684 6.0125C19.1684 7.07582 18.7456 8.09444 17.995 8.845L9.41 17.41C9.03469 17.7853 8.52539 17.9967 7.995 17.9967C7.46461 17.9967 6.95531 17.7853 6.58 17.41C6.20469 17.0347 5.99329 16.5254 5.99329 15.995C5.99329 15.4646 6.20469 14.9553 6.58 14.58L15.07 6.1" />
    </svg>
  );
};

export default AttachIcon;
