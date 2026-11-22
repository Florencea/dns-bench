---
trigger: glob
globs: "test/**, **/*.test.ts, **/*.test.tsx"
description: Dual-track testing architecture and Vitest standards across Chromium browser and Node environments.
---

# Testing Standards & Guidelines

Standards for unit, component, and integration testing across this codebase.

## 1. Dual-Track Testing Architecture

Tests are split into two Vitest workspace projects configured directly in `vite.config.ts`:

- **Renderer Project (`test/renderer/**`)**:
  - Runs in headless Chromium via `vitest/browser` and `@vitest/browser-playwright`.
  - Mounts components via `renderApp()` wrapped in `<QueryClientProvider>` with mock `window.api`.
  - Dual UI assertion requirement:
    - DOM presence: `await expect.element(el).toBeInTheDocument()`
    - Visibility: `await expect.element(el).toBeVisible()`
- **Main Project (`test/main/**`, `test/canary/**`)**:
  - Runs in native Node.js runtime (`npm run test:main`).
  - Tests DNS/DoH resolution algorithms, system menus, settings persistence, and static type contracts.

## 2. Invariants & Canary Type Contracts

- **Type Contract Tests**: `test/canary/e2e-type-contract.test.ts` asserts static contract fidelity using `expectTypeOf` from `vitest`. Any change to shared types, servers, or locales must update and pass these assertions.
- **Automated Verification**: Run `npm test` (or `npm test -- --reporter=tap-flat --no-color` for compact output) for unit/component tests and `npm run test:e2e` against packaged output for Electron CDP smoke verification.
