import { type Config } from "prettier";

const config: Config = {
  plugins: ["prettier-plugin-packagejson", "prettier-plugin-tailwindcss"],
  tailwindStylesheet: "./src/renderer/index.css",
  tailwindAttributes: ["UNSAFE_className"],
};

export default config;
