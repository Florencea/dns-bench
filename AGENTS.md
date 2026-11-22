# Agent Development Guidelines

Guidelines for AI agents and human contributors working on this repository.

This project uses standard `npm` with direct **Vite 8**, **Vitest 5**, **ESLint 10** (`typescript-eslint` v8), **Prettier 3**, and **TypeScript 6** without wrapper middleware layers or hidden compiler transforms.

## 1. Architecture Map

| Layer        | Path             | Responsibility                                                |
| :----------- | :--------------- | :------------------------------------------------------------ |
| **Main**     | `src/main/`      | Electron app lifecycle, window management, DNS & DoH engine   |
| **Preload**  | `src/preload/`   | Secure context bridge exposing typed `window.api`             |
| **Renderer** | `src/renderer/`  | React 19 UI with Adobe React Spectrum & Tailwind CSS v4       |
| **Shared**   | `src/shared/`    | Cross-process contracts (`ModeT`, `ServerT`) & DNS presets    |
| **Tests**    | `test/`          | Dual-track testing: Chromium browser (renderer) & Node (main) |
| **Tooling**  | `scripts/`       | Native Electron dev plugin and packaged CDP smoke test        |
| **Rules**    | `.agents/rules/` | Domain-specific modular rules activated via file globbing     |

---

## 2. Core SSOT & Invariants

- **Direct Proven Toolchain**: Manage dependencies with `npm` (`npm ci`, `npm install`) and invoke Vite, Vitest, ESLint, Prettier, TypeScript (`tsc -b`), and Electron Forge directly through shared `package.json` scripts. Avoid introducing experimental wrapper CLIs, 0.x native compiler transforms (no React Compiler), or redundant tool-specific script wrappers.
- **Strict `.npmrc`, `allowScripts` & Dependency Engines**: Keep `audit=false`, `engine-strict=true`, `fund=false`, `ignore-scripts=false`, `install-strategy=linked`, `save-exact=true`, and `strict-peer-deps=true` in `.npmrc`. Rely on npm's default-deny install-script policy governed by `package.json` `allowScripts` (permitting only `fs-xattr` and `macos-alias`), and let dependency `engines` ranges (enforced by `engine-strict=true`) govern Node.js compatibility without pinning fragile patch versions in root `package.json`. Install Playwright Chromium explicitly via `npm run test:setup`.
- **Strict TypeScript & Zero Suppression**: Single `typescript` (`6.0.2`) compiler across `tsc -b`, `typescript-eslint`, and VS Code. ESLint enforces `strictTypeChecked`, `stylisticTypeChecked`, `noInlineConfig: true`, `reportUnusedDisableDirectives: "error"`, and `--max-warnings 0`. Never use `// @ts-ignore`, `// @ts-expect-error`, `// @ts-nocheck`, `eslint-disable`, or unsafe `as` assertions. Run `npx knip` on demand for periodic dead-code audits.
- **Single Source of Truth**:
  - DNS presets reside strictly in `src/shared/servers.ts`.
  - Cross-process contracts reside in `src/shared/types.ts`.
  - Styling tokens reside in `src/renderer/index.css` via Tailwind CSS v4 `@theme`.
- **Security & Context Isolation**:
  - Main process never imports browser-only APIs; preload exposes only typed methods via `contextBridge`.
  - Raw Node.js modules or raw `ipcRenderer` must never be exposed to the renderer.
- **Domain Rules**: Path-specific rules live under `.agents/rules/` (`ci-workflows`, `electron-main`, `ui-styling`, `testing`).
- **CI & Actions Invariants**: All GitHub Actions in `.github/workflows/ci.yml` must pin exact SemVer tags and pass `npm run lint:ci`.

---

## 3. Shared Verification & Workflow Commands

Humans and agents share the same standard `npm run` scripts:

- **Inner Loop**: `npm run typecheck && npm run lint && npm run format:check`
- **Auto-Fix & Format**: `npm run lint:fix && npm run format`
- **Unit & Component Tests**: `npm test` (`npm run test:renderer` / `npm run test:main`; append `-- --reporter=tap-flat --no-color` for compact output)
- **Build & Package**: `npm run build`, `npm run app:pack`, `npm run app:mac`, `npm run app:linux`, `npm run app:win`
- **Packaged E2E Smoke Test**: `npm run test:e2e` (requires `npm run app:pack` first)
- **Full Verification Gate**: `npm run verify` (`typecheck` -> `lint` -> `format:check` -> `test` -> `app:pack` -> `test:e2e`)
- **CI Lint**: `npm run lint:ci` (`actionlint` 0 errors/warnings)
- **Development Server**: `npm run dev` (Vite 8 + `electronDevPlugin` hot-restart)

---

## 4. Git Workflow & Commit Restrictions

- **NEVER execute `git commit` directly**: Local environment uses 1Password SSH signing; non-interactive commit fails.
- **Protocol**: Stage changes with `git add <files>` and output `git commit -m "..."` in English for user to run locally.

---

## 5. Language & Planning Standards

- **Traditional Chinese for Plans & Responses**: All plans (`/plan`), walkthroughs, and chat responses must strictly be written in **Traditional Chinese (繁體中文)**.
- **Code Artifacts**: Source code, inline comments, commit messages, and automated tests must use concise English.
