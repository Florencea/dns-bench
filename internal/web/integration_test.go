package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Florencea/dns-bench/internal/config"
	"github.com/Florencea/dns-bench/internal/dns"
	"github.com/Florencea/dns-bench/internal/i18n"
	"github.com/Florencea/dns-bench/internal/web"
)

func newMockWebServer(t *testing.T, queryCounter *atomic.Int32) (*httptest.Server, func()) {
	t.Helper()

	dohServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if queryCounter != nil {
			queryCounter.Add(1)
		}
		w.Header().Set("Content-Type", "application/dns-json")
		if r.URL.Query().Get("type") == "AAAA" {
			_, _ = w.Write([]byte(`{"Answer":[{"type":28,"data":"2001:db8::99"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"Answer":[{"type":1,"data":"93.184.216.34"}]}`))
	}))

	eng := &dns.Engine{
		Servers: []dns.Server{
			{
				Title:       "Google Public DNS",
				Description: "Google resolver",
				Document:    "https://developers.google.com/speed/public-dns",
				DoHEndpoint: dohServer.URL,
				IP:          "-",
			},
			{
				Title:       "Legacy DNS Only",
				Description: "DNS without DoH",
				Document:    "-",
				DoHEndpoint: "-",
				IP:          "-",
			},
		},
		DNSTimeout:       time.Second,
		DoHTimeout:       time.Second,
		HTTPClient:       dohServer.Client(),
		AllowInsecureDoH: true,
		SystemLookup: func(_ context.Context, network, _ string) ([]netip.Addr, error) {
			if queryCounter != nil {
				queryCounter.Add(1)
			}
			if network == "ip4" {
				return []netip.Addr{netip.MustParseAddr("192.0.2.55")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("2001:db8::55")}, nil
		},
	}

	cfg := config.Config{
		Addr:          "127.0.0.1:8080",
		DefaultLocale: i18n.LocaleEnUS,
		DNSTimeout:    time.Second,
		DoHTimeout:    time.Second,
	}

	srv, err := web.NewServer(cfg, eng)
	if err != nil {
		dohServer.Close()
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	return ts, func() {
		ts.Close()
		dohServer.Close()
	}
}

func TestIndexModeFilteringAndQueryFlow(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	ts, cleanup := newMockWebServer(t, &calls)
	defer cleanup()

	// Default DNS mode shows both servers and executes zero outbound queries.
	resp, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body := readResponseBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d", resp.StatusCode)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected 0 outbound queries on idle GET /, got %d", calls.Load())
	}
	if !strings.Contains(body, "Google Public DNS") || !strings.Contains(body, "Legacy DNS Only") {
		t.Fatalf("expected both servers in DNS mode")
	}

	// Verify strict security headers.
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "script-src 'none'") {
		t.Errorf("CSP missing script-src 'none': %q", got)
	}

	// DoH mode filters out Legacy DNS Only.
	respDoH, err := ts.Client().Get(ts.URL + "/?mode=doh")
	if err != nil {
		t.Fatalf("GET /?mode=doh: %v", err)
	}
	bodyDoH := readResponseBody(t, respDoH)
	if !strings.Contains(bodyDoH, "Google Public DNS") || strings.Contains(bodyDoH, "Legacy DNS Only") {
		t.Fatalf("DoH mode failed to filter non-DoH server")
	}

	// Valid DNS query executes lookups and renders IPv4/IPv6 results.
	respQuery, err := ts.Client().Get(ts.URL + "/?mode=dns&host=example.com")
	if err != nil {
		t.Fatalf("GET /?mode=dns&host=example.com: %v", err)
	}
	bodyQuery := readResponseBody(t, respQuery)
	if respQuery.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on valid query, got %d", respQuery.StatusCode)
	}
	if !strings.Contains(bodyQuery, "192.0.2.55") || !strings.Contains(bodyQuery, "2001:db8::55") {
		t.Fatalf("expected resolved IPs in rendered page")
	}

	// Invalid domain returns 400 with Bootstrap .is-invalid and aria-invalid.
	respInvalid, err := ts.Client().Get(ts.URL + "/?mode=dns&host=bad%20domain!")
	if err != nil {
		t.Fatalf("GET invalid domain: %v", err)
	}
	bodyInvalid := readResponseBody(t, respInvalid)
	if respInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid domain, got %d", respInvalid.StatusCode)
	}
	if !strings.Contains(bodyInvalid, "is-invalid") || !strings.Contains(bodyInvalid, `aria-invalid="true"`) {
		t.Fatalf("expected .is-invalid and aria-invalid on invalid domain response")
	}
}

func TestSecurityGuardsCrossSiteQueryCSRFAndDNSRebinding(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	ts, cleanup := newMockWebServer(t, &calls)
	defer cleanup()

	// 1. Cross-site GET with host param must be rejected with 403 and zero DNS calls.
	reqCrossSite, _ := http.NewRequest(http.MethodGet, ts.URL+"/?mode=dns&host=example.com", nil)
	reqCrossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	reqCrossSite.Header.Set("Sec-Fetch-Dest", "document")
	respCross, err := ts.Client().Do(reqCrossSite)
	if err != nil {
		t.Fatalf("cross-site GET: %v", err)
	}
	_ = readResponseBody(t, respCross)
	if respCross.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on cross-site query, got %d", respCross.StatusCode)
	}
	if calls.Load() != 0 {
		t.Fatalf("cross-site query triggered %d outbound lookups", calls.Load())
	}

	// 2. Subresource image GET with host param must also be rejected.
	reqImg, _ := http.NewRequest(http.MethodGet, ts.URL+"/?mode=dns&host=example.com", nil)
	reqImg.Header.Set("Sec-Fetch-Site", "same-origin")
	reqImg.Header.Set("Sec-Fetch-Dest", "image")
	respImg, err := ts.Client().Do(reqImg)
	if err != nil {
		t.Fatalf("subresource GET: %v", err)
	}
	_ = readResponseBody(t, respImg)
	if respImg.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on image dest query, got %d", respImg.StatusCode)
	}

	// 3. Cross-origin POST /locale must be blocked by CrossOriginProtection.
	form := url.Values{"locale": {"zh-TW"}, "next": {"/"}}
	reqCSRF, _ := http.NewRequest(http.MethodPost, ts.URL+"/locale", strings.NewReader(form.Encode()))
	reqCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCSRF.Header.Set("Sec-Fetch-Site", "cross-site")
	reqCSRF.Header.Set("Origin", "https://evil.example")
	respCSRF, err := ts.Client().Do(reqCSRF)
	if err != nil {
		t.Fatalf("CSRF POST: %v", err)
	}
	_ = readResponseBody(t, respCSRF)
	if respCSRF.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on cross-origin POST /locale, got %d", respCSRF.StatusCode)
	}

	// 4. Non-loopback Host header (DNS rebinding) must be rejected with 421.
	reqRebind, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	reqRebind.Host = "evil.example.com:8080"
	respRebind, err := ts.Client().Do(reqRebind)
	if err != nil {
		t.Fatalf("rebinding GET: %v", err)
	}
	_ = readResponseBody(t, respRebind)
	if respRebind.StatusCode != http.StatusMisdirectedRequest {
		t.Fatalf("expected 421 on non-loopback Host, got %d", respRebind.StatusCode)
	}
}

func TestLocaleSwitchPreservesSafeQueryURLAndBlocksOpenRedirect(t *testing.T) {
	t.Parallel()

	ts, cleanup := newMockWebServer(t, nil)
	defer cleanup()

	noRedirectClient := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{"locale": {"zh-TW"}, "next": {"/?mode=doh&host=example.com"}}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/locale", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		t.Fatalf("POST /locale: %v", err)
	}
	_ = readResponseBody(t, resp)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/?mode=doh&host=example.com" {
		t.Fatalf("Location = %q, want /?mode=doh&host=example.com", loc)
	}

	// Open redirect attempt must collapse to "/".
	if got := web.SafeRedirectPath("https://evil.example/phish"); got != "/" {
		t.Fatalf("SafeRedirectPath allowed external URL: %q", got)
	}
	if got := web.SafeRedirectPath("//evil.example/phish"); got != "/" {
		t.Fatalf("SafeRedirectPath allowed protocol-relative URL: %q", got)
	}
}

func readResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}
