/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./static/**/*.{html,js}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Values come from CSS custom properties (see input.css :root and
        // [data-skin=...] blocks) so a skin can be swapped at runtime by
        // setting data-skin on <html>, without a Tailwind rebuild per skin.
        bg:    { DEFAULT: 'rgb(var(--color-bg) / <alpha-value>)', card: 'rgb(var(--color-bg-card) / <alpha-value>)', hover: 'rgb(var(--color-bg-hover) / <alpha-value>)' },
        fg:    { DEFAULT: 'rgb(var(--color-fg) / <alpha-value>)', muted: 'rgb(var(--color-fg-muted) / <alpha-value>)' },
        accent:{ DEFAULT: 'rgb(var(--color-accent) / <alpha-value>)', hover: 'rgb(var(--color-accent-hover) / <alpha-value>)', muted: 'rgb(var(--color-accent-muted) / <alpha-value>)' },
        danger:{ DEFAULT: 'rgb(var(--color-danger) / <alpha-value>)', hover: 'rgb(var(--color-danger-hover) / <alpha-value>)' },
        success:{ DEFAULT: 'rgb(var(--color-success) / <alpha-value>)' },
        warn:  { DEFAULT: 'rgb(var(--color-warn) / <alpha-value>)' },
      },
    },
  },
  plugins: [],
};
