/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        navy: {
          900: '#0A192F',
          800: '#0F2438',
          700: '#1E293B',
          600: '#1F3A5F',
        },
        gold: {
          DEFAULT: '#F5A623',
          light: '#FFA94D',
        },
      },
    },
  },
  plugins: [],
};
