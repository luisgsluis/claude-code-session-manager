/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./static/**/*.{html,js}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        bg:    { DEFAULT: '#0f0f1a', card: '#1a1a2e', hover: '#252542' },
        fg:    { DEFAULT: '#e0e0e0', muted: '#8b8b9e' },
        accent:{ DEFAULT: '#6c5ce7', hover: '#7d6ff0', muted: '#4834d4' },
        danger:{ DEFAULT: '#e74c3c', hover: '#f06292' },
        success:{ DEFAULT: '#2ed573' },
        warn:  { DEFAULT: '#ffa502' },
      },
    },
  },
  plugins: [],
};
