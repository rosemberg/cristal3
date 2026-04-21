/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        'dark-blue': 'var(--dark-blue)',
        'primary-blue': 'var(--primary-blue)',
        'flag-yellow': 'var(--flag-yellow)',
        'urn-green': 'var(--urn-green)',
        'light-blue-text': 'var(--light-blue-text)',
        'pale-blue-bg': 'var(--pale-blue-bg)',
        'chat-bg': 'var(--chat-bg)',
        'card-bg': 'var(--card-bg)',
        'border-subtle': 'var(--border-subtle)',
        'text-main': 'var(--text-main)',
        'text-secondary': 'var(--text-secondary)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Courier New', 'monospace'],
      },
      borderRadius: {
        'pill': '24px',
      },
    },
  },
  plugins: [],
};
