---
trigger: glob
globs: "web/**, internal/web/**"
description: Strict Bootstrap 5 CSS-only constraints, zero JavaScript enforcement, full-page SSR, and HTML accessibility standards.
---

# Bootstrap 5 CSS-Only and Zero JavaScript Rules

## Allowed Assets and Styling

- Link only `/static/bootstrap.min.css` as a stylesheet in templates.
- Never add custom `.css` files, `<style>` blocks, inline `style="..."` attributes, or CSS preprocessors.
- Use only class names defined in `web/static/bootstrap.min.css`.
- All rendered states must pass `internal/web/lint_test.go`.

## Zero JavaScript and Full-Page SSR Policy

- Never add `<script>` tags, external `.js` bundles (including `bootstrap.bundle.min.js`), `<iframe>` elements, inline event handlers (`onclick=`, `onchange=`, `onsubmit=`, etc.), or `javascript:` URIs.
- Never use Bootstrap JavaScript trigger attributes (`data-bs-toggle`, `data-bs-dismiss`, `data-bs-target`, `data-bs-ride`, `data-bs-slide`).
- Use full-page server rendering (`GET /?mode=...&host=...` for queries and `POST /locale` with `303 See Other` for language switching) and native `<details>`/`<summary>` for disclosure UI.
- Keep `script-src 'none'` and `frame-ancestors 'none'` active in `Content-Security-Policy` alongside `X-Frame-Options: DENY`.

## HTML5 Accessibility

- Set `lang="{{.Lang}}"` on `<html>`.
- Include landmark elements (`<header>`, `<nav aria-label="...">`, `<main>`).
- Pair every non-hidden, non-submit `<input>`, `<select>`, and `<textarea>` with a `<label for="...">` or `aria-label`.
- Mark active mode navigation links with `aria-current="page"`.
- Mark invalid form controls with `.is-invalid`, `aria-invalid="true"`, and `<div class="invalid-feedback d-block" role="alert">`.
