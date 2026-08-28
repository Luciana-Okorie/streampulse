import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#0B0E14",
        surface: "#131824",
        surfaceRaised: "#1A2130",
        line: "#232C3D",
        text: "#E8ECF1",
        textMuted: "#6B7684",
        signal: "#FF8A3D",
        ok: "#3DDC84",
        error: "#FF5470",
        info: "#5FB4E8",
      },
      fontFamily: {
        display: ["var(--font-display)", "sans-serif"],
        mono: ["var(--font-mono)", "monospace"],
      },
    },
  },
  plugins: [],
};
export default config;
