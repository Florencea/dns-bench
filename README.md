# DNS Bench

Self-contained DNS and DNS-over-HTTPS (DoH) benchmarking tool compiled to a single static Go binary (`CGO_ENABLED=0`) with zero external Go dependencies, serving a full-page HTML5 interface styled exclusively with embedded Bootstrap 5.3.9 CSS (zero JavaScript, zero custom CSS).

## Architecture Overview

- Single Static Executable: Embeds HTML5 templates, Bootstrap 5.3.9 CSS, and SVG/ICO icons via `embed.FS`. Requires no Node.js, Electron, C compiler, or external runtime.
- Zero-Dependency DNS and DoH Engine: Queries 12 preset DNS/DoH resolvers concurrently for IPv4 (`A`) and IPv6 (`AAAA`) records. Uses a pure standard-library RFC 1035 UDP/TCP wire client for explicit resolver IPs (consistent behavior across macOS, Linux, and Windows), `net.DefaultResolver` for System Default, and `net/http` `application/dns-json` for DoH endpoints.
- Strict Bootstrap 5 CSS-Only and Zero JavaScript: Uses full-page server rendering with bookmarkable query URLs (`/?mode=dns&host=example.com`), native `<details>` resolver metadata disclosure, compile-time checked `zh-TW` and `en-US` translations, and automated class/a11y verification in `internal/web/lint_test.go`.
- Localhost Security Hardening: Binds strictly to loopback (`127.0.0.1`, `::1`, or `localhost`), rejects non-loopback `Host` headers (`421 Misdirected Request`), blocks cross-site and non-document `GET /?host=...` lookups via `Sec-Fetch-Site` and `Sec-Fetch-Dest`, protects `POST /locale` via `net/http.CrossOriginProtection`, and enforces `Content-Security-Policy` (`script-src 'none'`; `frame-ancestors 'none'`) with `X-Frame-Options: DENY`.

## Prerequisites

- Go 1.27 or newer (`go.mod` targets `go 1.27.4`) for building from source.
- No Node.js, CGo, or packaging framework required.

## Build and Run

Build the static binary for your current platform:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/dns-bench ./cmd/dns-bench
```

Cross-compile static binaries for Windows, macOS, and Linux:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/dns-bench.exe ./cmd/dns-bench
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/dns-bench-darwin-arm64 ./cmd/dns-bench
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/dns-bench-linux-amd64 ./cmd/dns-bench
```

Start the local server (automatically binds `127.0.0.1:8080`, falls back to an available ephemeral loopback port if `8080` is occupied, and opens the default web browser):

```bash
./bin/dns-bench
```

On Windows, double-click `dns-bench.exe` to start the server and open the browser automatically. Close the console window or press `Ctrl+C` to stop.

Run without opening a browser automatically:

```bash
./bin/dns-bench --no-open --addr 127.0.0.1:8080
```

## Configuration Reference

Environment variables and matching CLI flags:

- `ADDR` / `--addr`: Loopback HTTP listen address (default `127.0.0.1:8080`; when omitted and port `8080` is occupied, automatically falls back to `127.0.0.1:0`).
- `DEFAULT_LOCALE` / `--locale`: Fallback UI language when neither cookie nor `Accept-Language` matches (`zh-TW` or `en-US`, default `zh-TW`).
- `NO_OPEN` / `--no-open`: Disable automatic system browser launch on startup (default `false`).
- `DNS_TIMEOUT`: Per-query UDP/TCP DNS timeout duration (default `2500ms`, maximum `30s`).
- `DOH_TIMEOUT`: Per-query DoH HTTP timeout duration (default `3000ms`, maximum `30s`).
- `--version`: Print version and Go runtime information and exit.

## Verification and Quality Gate

```bash
test -z "$(gofmt -l .)"
go vet ./...
CGO_ENABLED=1 go test -race -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/dns-bench ./cmd/dns-bench
actionlint
```

## Licensing

- Project license: MIT (`LICENSE`).
- Third-party notice and license text for embedded Bootstrap 5.3.9 CSS are included in `NOTICE` and `THIRD_PARTY_LICENSES/bootstrap-5.3.9-LICENSE.txt`.
