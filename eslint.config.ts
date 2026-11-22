import js from "@eslint/js";
import vitest from "@vitest/eslint-plugin";
import reactHooks from "eslint-plugin-react-hooks";
import { reactRefresh } from "eslint-plugin-react-refresh";
import { defineConfig, globalIgnores } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

export default defineConfig([
  globalIgnores([
    "dist",
    "dist-ssr",
    "release",
    "out",
    ".cache",
    ".vitest",
    "test-results",
    "playwright-report",
    "blob-report",
  ]),
  {
    linterOptions: {
      noInlineConfig: true,
      reportUnusedDisableDirectives: "error",
    },
  },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      tseslint.configs.strictTypeChecked,
      tseslint.configs.stylisticTypeChecked,
    ],
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      eqeqeq: ["error", "always"],
      "@typescript-eslint/ban-ts-comment": [
        "error",
        {
          "ts-ignore": true,
          "ts-expect-error": true,
          "ts-nocheck": true,
          "ts-check": false,
        },
      ],
      "@typescript-eslint/consistent-type-exports": [
        "error",
        { fixMixedExportsWithInlineTypeSpecifier: true },
      ],
      "@typescript-eslint/consistent-type-imports": [
        "error",
        { prefer: "type-imports", fixStyle: "inline-type-imports" },
      ],
      "@typescript-eslint/no-shadow": "error",
      "@typescript-eslint/no-unnecessary-qualifier": "error",
      "@typescript-eslint/no-unsafe-type-assertion": "error",
      "@typescript-eslint/no-useless-empty-export": "error",
      "@typescript-eslint/prefer-readonly": "error",
      "@typescript-eslint/promise-function-async": "error",
      "@typescript-eslint/require-array-sort-compare": "error",
      "@typescript-eslint/strict-boolean-expressions": "error",
      "@typescript-eslint/switch-exhaustiveness-check": "error",
    },
  },
  {
    files: ["src/renderer/**/*.{ts,tsx}"],
    extends: [
      reactHooks.configs.flat["recommended-latest"],
      reactRefresh.configs.vite(),
    ],
    languageOptions: {
      globals: globals.browser,
    },
    rules: {
      "react-hooks/exhaustive-deps": "error",
    },
  },
  {
    files: ["src/main/**/*.ts", "src/preload/**/*.ts"],
    languageOptions: {
      globals: globals.node,
    },
  },
  {
    files: ["src/shared/**/*.ts"],
    languageOptions: {
      globals: globals.es2023,
    },
  },
  {
    files: ["test/**/*.{ts,tsx}"],
    plugins: {
      vitest,
    },
    settings: {
      vitest: {
        typecheck: true,
      },
    },
    rules: {
      ...vitest.configs.recommended.rules,
    },
  },
  {
    files: ["test/main/**/*.ts", "test/canary/**/*.ts"],
    languageOptions: {
      globals: globals.node,
    },
  },
  {
    files: ["test/renderer/**/*.{ts,tsx}"],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
  {
    files: ["scripts/**/*.ts", "*.config.ts"],
    languageOptions: {
      globals: globals.node,
    },
  },
]);
