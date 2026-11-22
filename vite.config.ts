import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { playwright } from "@vitest/browser-playwright";
import { defineConfig } from "vitest/config";
import { electronDevPlugin } from "./scripts/electron-dev-plugin.ts";

export default defineConfig(({ mode }) => {
  if (mode === "main") {
    return {
      build: {
        ssr: "src/main/index.ts",
        outDir: "dist/main",
      },
    };
  }

  if (mode === "preload") {
    return {
      build: {
        ssr: "src/preload/index.ts",
        outDir: "dist/preload",
        rolldownOptions: {
          output: {
            format: "cjs",
            entryFileNames: "index.cjs",
          },
          external: ["electron"],
        },
      },
      ssr: {
        noExternal: true,
      },
    };
  }

  return {
    base: "./",
    server: {
      watch: {
        ignored: ["**/src/main/**", "**/src/preload/**"],
      },
    },
    build: {
      outDir: "dist/renderer",
      chunkSizeWarningLimit: 1000,
    },
    plugins: [react(), tailwindcss(), electronDevPlugin()],
    test: {
      silent: "passed-only",
      allowOnly: (process.env.CI ?? "") === "",
      projects: [
        {
          extends: true,
          test: {
            name: "renderer",
            include: ["test/renderer/**/*.{test,spec}.{ts,tsx}"],
            setupFiles: ["./test/renderer/setup.ts"],
            browser: {
              enabled: true,
              provider: playwright(),
              headless: true,
              instances: [{ browser: "chromium" }],
            },
          },
        },
        {
          extends: true,
          test: {
            name: "main",
            include: [
              "test/main/**/*.{test,spec}.ts",
              "test/canary/**/*.{test,spec}.ts",
            ],
            environment: "node",
          },
        },
      ],
    },
  };
});
