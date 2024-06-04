/** @type {import('tailwindcss').Config} */
module.exports = {
  corePlugins: {
    preflight: false,
    container: false,
  },
  theme: {
    fontFamily: {
      inter: ["Inter", "sans-serif"],
      roboto: ["Roboto", "sans-serif"],
      mono: ["Space Mono", "monospace"],
    },
    colors: {
      black: '#1c1e21',
      darkBlue: "#263066",
      lightBlue: "#727BA8",
      lightGrey: "#E1E4F5",
    },
  },
  content: ["./src/**/*.{jsx,tsx,html}", "../md/**/*.{mdx,tsx}"],
  plugins: [],
};
