import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        surface: "#101820",
        panel: "#17212b",
        accent: "#00eaff",
      },
    },
  },
  plugins: [],
};

export default config;
