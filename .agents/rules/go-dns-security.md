---
trigger: glob
globs: "cmd/**, internal/dns/**, internal/config/**, internal/web/**"
description: Go zero-dependency static binary invariants, loopback security boundaries, and RFC 1035 wire parsing guards.
---

# Go DNS and Localhost Security Rules

## Zero External Dependencies and Static Builds

- Keep `go.mod` free of external `require` dependencies; rely exclusively on the Go standard library.
- Keep production builds compatible with `CGO_ENABLED=0` across macOS, Linux, and Windows.

## Localhost Network Hardening

- Restrict server listening addresses to loopback hosts (`127.0.0.1`, `::1`, `localhost`).
- Enforce `LoopbackHostMiddleware` on all routes to reject DNS rebinding with `421 Misdirected Request`.
- Guard `GET /?host=...` queries with `Sec-Fetch-Site` and `Sec-Fetch-Dest` validation so cross-site or subresource requests cannot trigger local outbound DNS or DoH probes.
- Guard state-changing `POST` routes with `net/http.CrossOriginProtection` and `http.MaxBytesReader`, and validate redirect paths with `SafeRedirectPath`.
- Validate that automatic browser launch URLs are strictly `http://` loopback addresses and never pass URLs through a shell interpreter.

## DNS Wire and DoH Parsing Boundaries

- Verify randomized 16-bit Transaction IDs, QR flags, RCODE, and packet length bounds in `internal/dns/wire.go`.
- Enforce a jump ceiling on RFC 1035 compression pointers to prevent infinite loops from malformed packets.
- Bound DoH HTTP response reads with `io.LimitReader`, enforce `https://` endpoints in production, and validate returned IP strings with `net/netip.ParseAddr`.
