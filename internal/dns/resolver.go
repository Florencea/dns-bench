package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultDNSTimeout = 2500 * time.Millisecond
	DefaultDoHTimeout = 3000 * time.Millisecond
	maxDoHBodyBytes   = 64 << 10 // 64 KiB
)

// ServerResult holds formatted IPv4 and IPv6 benchmark output for one resolver.
type ServerResult struct {
	Server Server
	IPv4   string
	IPv6   string
	Output string
}

// Engine executes concurrent DNS and DoH benchmarks.
type Engine struct {
	Servers          []Server
	DNSTimeout       time.Duration
	DoHTimeout       time.Duration
	DNSPort          string
	HTTPClient       *http.Client
	AllowInsecureDoH bool
	SystemLookup     func(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// NewEngine creates a DNS/DoH engine with safe default timeouts and preset servers.
func NewEngine(dnsTimeout, dohTimeout time.Duration) *Engine {
	if dnsTimeout <= 0 {
		dnsTimeout = DefaultDNSTimeout
	}
	if dohTimeout <= 0 {
		dohTimeout = DefaultDoHTimeout
	}
	return &Engine{
		Servers:    DefaultServers(),
		DNSTimeout: dnsTimeout,
		DoHTimeout: dohTimeout,
		DNSPort:    "53",
		HTTPClient: &http.Client{
			Timeout: dohTimeout,
		},
		SystemLookup: net.DefaultResolver.LookupNetIP,
	}
}

// FilteredServers returns the servers visible for the given mode.
func (e *Engine) FilteredServers(mode Mode) []Server {
	servers := e.Servers
	if len(servers) == 0 {
		servers = DefaultServers()
	}
	return FilterServers(servers, mode)
}

// QueryAll concurrently queries A and AAAA records across every server for mode.
func (e *Engine) QueryAll(ctx context.Context, mode Mode, rawDomain string) ([]ServerResult, error) {
	domain, err := ValidateDomain(rawDomain)
	if err != nil {
		return nil, err
	}
	targets := e.FilteredServers(mode)
	results := make([]ServerResult, len(targets))

	var wg sync.WaitGroup
	for i, srv := range targets {
		results[i].Server = srv
		wg.Add(2)

		go func(idx int, server Server) {
			defer wg.Done()
			results[idx].IPv4 = e.queryOne(ctx, mode, server, domain, TypeA)
		}(i, srv)

		go func(idx int, server Server) {
			defer wg.Done()
			results[idx].IPv6 = e.queryOne(ctx, mode, server, domain, TypeAAAA)
		}(i, srv)
	}
	wg.Wait()

	for i := range results {
		parts := make([]string, 0, 2)
		if results[i].IPv4 != "" {
			parts = append(parts, results[i].IPv4)
		}
		if results[i].IPv6 != "" {
			parts = append(parts, results[i].IPv6)
		}
		results[i].Output = strings.Join(parts, "\n\n")
	}

	return results, nil
}

func (e *Engine) queryOne(ctx context.Context, mode Mode, srv Server, domain string, rt RecordType) string {
	if mode == ModeDoH {
		return e.fetchDoH(ctx, srv.DoHEndpoint, domain, rt)
	}
	return e.resolveDNS(ctx, srv.IP, domain, rt)
}

func (e *Engine) resolveDNS(ctx context.Context, serverIP, domain string, rt RecordType) string {
	timeout := e.DNSTimeout
	if timeout <= 0 {
		timeout = DefaultDNSTimeout
	}
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var (
		addrs []netip.Addr
		err   error
	)

	if serverIP == "-" || serverIP == "" {
		lookup := e.SystemLookup
		if lookup == nil {
			lookup = net.DefaultResolver.LookupNetIP
		}
		network := "ip4"
		if rt == TypeAAAA {
			network = "ip6"
		}
		addrs, err = lookup(qctx, network, domain)
	} else {
		if _, parseErr := netip.ParseAddr(serverIP); parseErr != nil {
			return ""
		}
		qtype, codeErr := recordTypeCode(rt)
		if codeErr != nil {
			return ""
		}
		port := e.DNSPort
		if port == "" {
			port = "53"
		}
		addrs, err = queryWireDNS(qctx, net.JoinHostPort(serverIP, port), domain, qtype)
	}

	elapsed := time.Since(start)
	if err != nil {
		return ""
	}
	return formatResult(rt, elapsed, filterAddrs(addrs, rt))
}

type dohResponse struct {
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

func (e *Engine) fetchDoH(ctx context.Context, endpoint, domain string, rt RecordType) string {
	if endpoint == "" || endpoint == "-" {
		return ""
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	if u.Scheme != "https" && !(e.AllowInsecureDoH && u.Scheme == "http") {
		return ""
	}
	q := u.Query()
	q.Set("name", domain)
	q.Set("type", string(rt))
	u.RawQuery = q.Encode()

	timeout := e.DoHTimeout
	if timeout <= 0 {
		timeout = DefaultDoHTimeout
	}
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(qctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/dns-json")

	client := e.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ""
	}

	var parsed dohResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDoHBodyBytes)).Decode(&parsed); err != nil {
		return ""
	}
	if len(parsed.Answer) == 0 {
		return ""
	}

	addrs := make([]netip.Addr, 0, len(parsed.Answer))
	for _, ans := range parsed.Answer {
		addr, parseErr := netip.ParseAddr(strings.TrimSpace(ans.Data))
		if parseErr != nil {
			continue
		}
		if (rt == TypeA && addr.Is4()) || (rt == TypeAAAA && addr.Is6()) {
			addrs = append(addrs, addr)
		}
	}

	return formatResult(rt, elapsed, addrs)
}

func filterAddrs(addrs []netip.Addr, rt RecordType) []netip.Addr {
	out := make([]netip.Addr, 0, len(addrs))
	for _, a := range addrs {
		if (rt == TypeA && a.Is4()) || (rt == TypeAAAA && a.Is6()) {
			out = append(out, a)
		}
	}
	return out
}

func formatResult(rt RecordType, elapsed time.Duration, addrs []netip.Addr) string {
	label := "ipv4"
	if rt == TypeAAAA {
		label = "ipv6"
	}
	ms := float64(elapsed.Microseconds()) / 1000.0
	lines := make([]string, 0, 1+len(addrs))
	lines = append(lines, fmt.Sprintf("%s: %.2fms", label, ms))
	for _, a := range addrs {
		lines = append(lines, a.String())
	}
	return strings.Join(lines, "\n")
}
