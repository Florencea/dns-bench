# VP Electron DNS

[![CI](https://github.com/Florencea/vp-electron-dns/actions/workflows/ci.yml/badge.svg)](https://github.com/Florencea/vp-electron-dns/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A DNS and DNS-over-HTTPS (DoH) benchmarking desktop application powered by **Electron**, **Vite 8**, **React 19**, **Adobe React Spectrum**, **Tailwind CSS v4**, and **TanStack Query**.

Designed for deterministic DNS evaluation across preset and custom resolvers, featuring direct multi-target Vite/Rolldown builds, transparent HMR with automated main-process hot-restart, and zero third-party Electron wrapper meta-frameworks.

---

## Highlights

- **Strict Quality Gate**: Unified verification gate (`npm run verify`) executing strict TypeScript (`tsc -b`), ESLint (`strictTypeChecked` / `stylisticTypeChecked`, `noInlineConfig: true`, `--max-warnings 0`), Prettier formatting, dual-track Vitest tests, Electron Forge packaging, and CDP E2E smoke validation.
- **Direct Vite 8 & Rolldown**: Built directly with Vite 8 and Rolldown across main, preload, and renderer targets without wrapper CLIs or hidden compiler layers.
- **Hot-Restart & Dev Runner**: Transparent Vite dev plugin (`scripts/electron-dev-plugin.ts`) providing process hot-restart on main process edits, full reload on preload changes, and React HMR for the renderer.
- **Dual-Track Testing**: Renderer UI tests execute in headless Chromium browser mode via `@vitest/browser-playwright`, while main process, DNS resolver, and static type contract tests run in Node.js.
- **Clear Architecture & Governance**: Explicit process boundaries, Zod boundary validation, and contributor guidelines documented in [AGENTS.md](AGENTS.md).
- **Process Isolation & Security**: Safe context bridge exposition ensuring raw Node.js internals and IPC channels remain isolated from the renderer.

---

## Tech Stack

| Layer                | Technology                                                      |
| :------------------- | :-------------------------------------------------------------- |
| **Runtime & Shell**  | Electron (Context Isolation, Safe IPC Bridge)                   |
| **Frontend UI**      | React 19, Adobe React Spectrum, Tailwind CSS v4                 |
| **State & Fetching** | TanStack Query                                                  |
| **Build & Dev**      | Vite 8 (Rolldown), Native Electron Dev Plugin                   |
| **Testing**          | Vitest 5 (Chromium Browser Mode + Node Backend Tests)           |
| **Packaging**        | Electron Forge v8 (ASAR, Fuses, DMG, ZIP)                       |
| **Code Quality**     | TypeScript 6 (strict), ESLint 10, typescript-eslint, Prettier 3 |

---

## Getting Started

### Prerequisites

- **Node.js** - Active LTS runtime satisfying the `engines` ranges declared by project dependencies (enforced by `.npmrc` `engine-strict=true`).

### Setup

1. **Install dependencies:**

   ```sh
   npm ci
   ```

2. **Setup Playwright Chromium binary:**

   ```sh
   npm run test:setup
   ```

3. **Run in development mode:**

   ```sh
   npm run dev
   ```

---

## Available Scripts

| Command                 | Description                                                                    |
| :---------------------- | :----------------------------------------------------------------------------- |
| `npm run dev`           | Start development environment with Electron hot-restart runner (`vite`)        |
| `npm run build`         | Build all targets (main, preload, renderer) with Vite                          |
| `npm run typecheck`     | Run strict TypeScript project-reference check (`tsc -b --pretty false`)        |
| `npm run lint`          | Lint codebase with ESLint (`eslint --max-warnings 0 .`)                        |
| `npm run lint:fix`      | Automatically fix ESLint issues (`eslint --fix .`)                             |
| `npm run format`        | Format files and sort Tailwind classes with Prettier (`prettier --write .`)    |
| `npm run format:check`  | Verify formatting without modifying files (`prettier --check .`)               |
| `npm run lint:ci`       | Validate GitHub Actions workflows locally with `actionlint`                    |
| `npm test`              | Run all Vitest projects (Chromium browser renderer + Node main)                |
| `npm run test:renderer` | Run Chromium browser renderer tests (`vitest run --project renderer`)          |
| `npm run test:main`     | Run Node main and type contract tests (`vitest run --project main`)            |
| `npm run test:e2e`      | Run packaged Electron CDP smoke test (`node scripts/smoke-test.ts`)            |
| `npm run test:setup`    | Install Playwright Chromium browser binary                                     |
| `npm run app:pack`      | Build and package unpacked Electron application via Electron Forge             |
| `npm run app:mac`       | Package macOS application into `out/`                                          |
| `npm run app:win`       | Package Windows application into `out/`                                        |
| `npm run app:linux`     | Package Linux application into `out/`                                          |
| `npm run verify`        | Full gate: `typecheck`, `lint`, `format:check`, `test`, `app:pack`, `test:e2e` |

---

## CI/CD Pipeline Architecture

- **Tiered Daily CI (`.github/workflows/ci.yml`)**:
  - **Tier 1 (`gatekeeper`)**: Runs on `ubuntu-24.04` with Node.js Active LTS (`lts/*`). Executes `npm ci`, Playwright browser caching and `test:setup`, static verification (`typecheck`, `lint`, `format:check`), unit/component suites (`npm test -- --reporter=tap-flat --no-color`), unpacked Electron packaging (`app:pack`), Linux distribution packaging (`app:linux`), and headless Electron E2E smoke testing (`xvfb-run --auto-servernum npm run test:e2e`).
  - **Tier 2 (`platform-package`)**: Executes after `gatekeeper` succeeds across `macos-26` and `windows-2025`. Validates runtime path compatibility (`npm test -- --reporter=tap-flat --no-color`), production builds, and platform distribution packaging (`app:mac`, `app:win`).

---

## Guidelines

For architectural rules, security standards, and the contributor workflow, see [AGENTS.md](AGENTS.md).

---

## Note

- To run `npm run app:win` on macOS ARM (Apple Silicon), Rosetta 2 is required.

---

## License

This project is licensed under the MIT License.
