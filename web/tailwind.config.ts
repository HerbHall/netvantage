import type { Config } from "tailwindcss";

/**
 * NetVantage Tailwind Configuration
 *
 * Brand palette: forest greens + earth tones (amber, sage, warm cream).
 * Dark mode is the default; light mode via [data-theme="light"].
 *
 * All custom colors are also available as CSS variables in
 * src/styles/design-tokens.css for non-Tailwind usage.
 */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: ["selector", '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        // Brand palette
        green: {
          50:  "#f0fdf4",
          100: "#dcfce7",
          200: "#bbf7d0",
          300: "#86efac",
          400: "#4ade80",
          500: "#22c55e",
          600: "#16a34a",
          700: "#15803d",
          800: "#166534",
          900: "#14532d",
          950: "#052e16",
        },
        earth: {
          50:  "#fdf8f0",
          100: "#f5edd8",
          200: "#e8d5b0",
          300: "#d4b87e",
          400: "#c4a77d",
          500: "#a3845a",
          600: "#92734e",
          700: "#7a5f3f",
          800: "#654d34",
          900: "#4a3826",
          950: "#2d2118",
        },
        sage: {
          50:  "#f5f5f0",
          100: "#e8e8df",
          200: "#d4d4c8",
          300: "#b8c4a0",
          400: "#9ca389",
          500: "#7d856e",
          600: "#6b7a56",
          700: "#5c6650",
          800: "#4a5240",
          900: "#3a4032",
          950: "#252922",
        },
        forest: {
          50:  "#f0f5f1",
          100: "#d6e4d8",
          200: "#a8c4ad",
          300: "#6a9972",
          400: "#3d7245",
          500: "#2a5630",
          600: "#1f4024",
          700: "#1a2e1c",
          800: "#162414",
          900: "#0f1a10",
          950: "#0c1a0e",
        },
        // Semantic aliases
        status: {
          online:   "#4ade80",
          degraded: "#c4a77d",
          offline:  "#ef4444",
          unknown:  "#9ca389",
        },
      },
      fontFamily: {
        sans: ["-apple-system", "BlinkMacSystemFont", "Segoe UI", "Inter", "Helvetica", "Arial", "sans-serif"],
        mono: ["JetBrains Mono", "Fira Code", "Cascadia Code", "Consolas", "monospace"],
      },
      borderRadius: {
        sm: "4px",
        md: "8px",
        lg: "12px",
        xl: "16px",
      },
      boxShadow: {
        glow: "0 0 20px rgba(74, 222, 128, 0.15)",
        "glow-sm": "0 0 8px rgba(74, 222, 128, 0.1)",
      },
    },
  },
  plugins: [],
} satisfies Config;
