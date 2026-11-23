package web

import (
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Florencea/dns-bench/internal/config"
	"github.com/Florencea/dns-bench/internal/i18n"
)

// ContentSecurityPolicy enforces strict zero-script, zero-framing, self-only styles.
const ContentSecurityPolicy = "default-src 'self'; script-src 'none'; style-src 'self'; img-src 'self' data:; form-action 'self'; frame-ancestors 'none'; base-uri 'self'"

// SecurityHeadersMiddleware sets strict CSP, clickjacking, MIME, and referrer headers.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", ContentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// LoopbackHostMiddleware rejects non-loopback Host headers to mitigate DNS rebinding.
func LoopbackHostMiddleware(defaultLocale string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := extractHostOnly(r.Host)
		if !config.IsLoopbackHost(host) {
			t := i18n.Catalog(i18n.ResolveLocale(r, defaultLocale))
			http.Error(w, t.Errors.HostRejected, http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractHostOnly(hostport string) string {
	trimmed := strings.TrimSpace(hostport)
	if trimmed == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		return host
	}
	return trimmed
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// RecoverAndLogMiddleware recovers handler panics and logs non-static HTTP requests.
func RecoverAndLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if recVal := recover(); recVal != nil {
				slog.Error("http panic recovered",
					"error", recVal,
					"path", r.URL.Path,
					"method", r.Method,
					"stack", string(debug.Stack()),
				)
				http.Error(rec, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(rec, r)

		if !strings.HasPrefix(r.URL.Path, "/static/") {
			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
	})
}
