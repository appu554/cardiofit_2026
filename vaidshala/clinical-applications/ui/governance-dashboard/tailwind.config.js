/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        // CSS variable-based colors for shadcn-style theming
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        // Clinical governance color palette
        governance: {
          primary: '#1e40af',    // Deep blue for trust
          secondary: '#7c3aed',  // Purple for review actions
          success: '#059669',    // Green for approved
          warning: '#d97706',    // Amber for pending review
          danger: '#dc2626',     // Red for rejected/critical
          info: '#0891b2',       // Cyan for information
        },
        priority: {
          critical: '#dc2626',
          high: '#ea580c',
          standard: '#2563eb',
          low: '#6b7280',
        },
        status: {
          draft: '#6b7280',
          pending: '#d97706',
          approved: '#059669',
          active: '#2563eb',
          rejected: '#dc2626',
          superseded: '#9333ea',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
};
