# Agent Development Guidelines

Architecture and engineering invariants for `dns-bench`.

## Architecture Map

- `cmd/dns-bench/`: Single executable entrypoint with loopback listener fallback, automatic OS browser launch, and graceful shutdown.
- `internal/config/`: Runtime configuration loading and loopback-only binding validation.
- `internal/dns/`: Preset DNS/DoH server SSOT (`servers.go`), domain/mode validation (`validate.go`), pure standard-library RFC 1035 UDP/TCP wire codec (`wire.go`), and concurrent IPv4/IPv6 benchmark engine (`resolver.go`).
- `internal/i18n/`: Compile-time checked `zh-TW` and `en-US` translation catalogs with cookie and `Accept-Language` resolution.
- `internal/web/`: Full-page server-rendered HTML handlers, strict security headers, loopback `Host` header check, `net/http.CrossOriginProtection`, cross-site query fetch guard, and automated Bootstrap/a11y lint tests.
- `web/`: Embedded Bootstrap 5.3.9 CSS (`web/static/bootstrap.min.css`), SVG DNS icon (`web/static/dns-icon.svg`), favicon, and HTML5 templates (`web/templates/*.html`).
- `.agents/rules/`: Domain rules for Bootstrap CSS-only zero-JS UI, Go DNS/network security invariants, and GitHub Actions CI.

## Core Invariants

- Single Static Binary with Zero External Go Dependencies (`CGO_ENABLED=0`): Use only the Go standard library in `go.mod` and embed frontend assets via `web/embed.go`. Do not introduce Node.js, npm, Electron, CGo, or frontend bundlers.
- Strict Bootstrap 5 CSS-Only UI and Zero JavaScript: Never add `<script>`, `<style>`, `<iframe>`, `style=`, `on*=` attributes, `javascript:` URIs, `data-bs-*` attributes, or custom CSS files. Only class names present in `web/static/bootstrap.min.css` are permitted, enforced by `internal/web/lint_test.go`.
- Full-Page Server Rendering and Address-Bar Query URLs: Mode switching (`GET /?mode=dns|doh`) and DNS queries (`GET /?mode=...&host=...`) use standard HTML forms and full-page rendering. Locale switches use `POST /locale` with `303 See Other` redirects.
- Localhost Security Boundaries: Bind strictly to loopback (`127.0.0.1`, `::1`, or `localhost`), enforce `Host` loopback allowlisting against DNS rebinding (`421 Misdirected Request`), enforce `Sec-Fetch-Site` and `Sec-Fetch-Dest` checks on `GET /?host=...` against cross-site local DNS triggering, enforce `net/http.CrossOriginProtection` on `POST /locale`, and serve `script-src 'none'` with `X-Frame-Options: DENY` on every route.
- Third-Party Licensing: Preserve `NOTICE` and `THIRD_PARTY_LICENSES/`. Do not add copyleft dependencies.

## Verification Commands

Run the full local gate after changes:

```bash
test -z "$(gofmt -l .)"
go vet ./...
CGO_ENABLED=1 go test -race -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/dns-bench ./cmd/dns-bench
actionlint
```

## Git Workflow Restrictions

- Never execute `git commit` directly because local commits rely on interactive SSH signing.
- Stage changes with `git add` and output the suggested `git commit -m "..."` command in English for the user to execute.

## Language Standards

- Write all dialogue, walkthroughs, and plans in Traditional Chinese (Taiwan usage).
- Write all source code, comments, templates, tests, standalone documentation, and commit messages in concise English (except localized UI strings in `zh-TW` catalogs).
