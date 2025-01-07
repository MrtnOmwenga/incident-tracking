/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        red_dark: "#7f1d1d",
        red_mid: "#b91c1c",
        red_light: "#ef4444",
      },

      fontFamily: {
        mono: ['"Roboto Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
}