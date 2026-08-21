import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        void: "#070B10",
        steel: "#121A24",
        "steel-2": "#1B2633",
        line: "#2A3A4A",
        phosphor: "#7CFFB2",
        "phosphor-dim": "#1E4D38",
        amber: "#F5B942",
        fog: "#C5D0DA",
        mist: "#7A8B9A",
        danger: "#FF6B6B",
      },
      fontFamily: {
        display: ["Syne", "sans-serif"],
        body: ["Figtree", "sans-serif"],
        mono: ["Fragment Mono", "ui-monospace", "monospace"],
      },
      screens: {
        xs: "480px",
      },
    },
  },
  plugins: [],
} satisfies Config;
