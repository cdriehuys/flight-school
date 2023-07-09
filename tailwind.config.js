/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["content/**/*.md", "data/**/*.yml", "layouts/**/*.html"],
  theme: {
    extend: {},
  },
  plugins: [require("@tailwindcss/typography")],
};
