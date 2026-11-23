package dns

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestDefaultServersInvariants(t *testing.T) {
	t.Parallel()

	servers := DefaultServers()
	if len(servers) != 12 {
		t.Fatalf("expected 12 preset servers, got %d", len(servers))
	}

	for _, s := range servers {
		if strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Description) == "" ||
			strings.TrimSpace(s.Document) == "" || strings.TrimSpace(s.DoHEndpoint) == "" ||
			strings.TrimSpace(s.IP) == "" {
			t.Fatalf("server %+v has empty required field", s)
		}
		if s.DoHEndpoint != "-" && !strings.HasPrefix(s.DoHEndpoint, "https://") {
			t.Fatalf("server %q has non-https DoH endpoint %q", s.Title, s.DoHEndpoint)
		}
		if s.IP != "-" {
			if _, err := netip.ParseAddr(s.IP); err != nil {
				t.Fatalf("server %q has invalid IP %q: %v", s.Title, s.IP, err)
			}
		}
	}

	dohServers := FilterServers(servers, ModeDoH)
	if len(dohServers) != 4 {
		t.Fatalf("expected 4 DoH servers, got %d", len(dohServers))
	}
}

func TestValidateDomain(t *testing.T) {
	t.Parallel()

	valid := []struct {
		in   string
		want string
	}{
		{"example.com", "example.com"},
		{"  sub.example.co.uk  ", "sub.example.co.uk"},
		{"localhost", "localhost"},
		{"a-b.c1.io", "a-b.c1.io"},
	}
	for _, tc := range valid {
		got, err := ValidateDomain(tc.in)
		if err != nil {
			t.Errorf("ValidateDomain(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ValidateDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	invalid := []struct {
		in   string
		want error
	}{
		{"", ErrDomainRequired},
		{"   ", ErrDomainRequired},
		{strings.Repeat("a", 254), ErrDomainTooLong},
		{"-bad.example.com", ErrDomainInvalid},
		{"bad domain.com", ErrDomainInvalid},
		{"http://example.com", ErrDomainInvalid},
		{"example.com/path", ErrDomainInvalid},
	}
	for _, tc := range invalid {
		if _, err := ValidateDomain(tc.in); !errors.Is(err, tc.want) {
			t.Errorf("ValidateDomain(%q) error = %v, want %v", tc.in, err, tc.want)
		}
	}
}

func TestParseDNSResponseSecurityGuards(t *testing.T) {
	t.Parallel()

	// Transaction ID mismatch must be rejected.
	resp := make([]byte, 12)
	binary.BigEndian.PutUint16(resp[0:2], 0x1234)
	binary.BigEndian.PutUint16(resp[2:4], 0x8180)
	if _, _, err := parseDNSResponse(resp, 0x9999, qtypeA); !errors.Is(err, errTransactionID) {
		t.Fatalf("expected errTransactionID, got %v", err)
	}

	// Self-referencing compression pointer loop must be rejected without hanging.
	loopPkt := make([]byte, 18)
	binary.BigEndian.PutUint16(loopPkt[0:2], 0x4242)
	binary.BigEndian.PutUint16(loopPkt[2:4], 0x8180)
	binary.BigEndian.PutUint16(loopPkt[4:6], 1)        // QDCOUNT=1
	binary.BigEndian.PutUint16(loopPkt[12:14], 0xC00C) // points to offset 12 (self)
	if _, _, err := parseDNSResponse(loopPkt, 0x4242, qtypeA); !errors.Is(err, errCompressionLoop) {
		t.Fatalf("expected errCompressionLoop, got %v", err)
	}

	// Non-zero RCODE (NXDOMAIN=3) must fail.
	nx := make([]byte, 12)
	binary.BigEndian.PutUint16(nx[0:2], 0x0007)
	binary.BigEndian.PutUint16(nx[2:4], 0x8183)
	if _, _, err := parseDNSResponse(nx, 0x0007, qtypeA); !errors.Is(err, errResponseCode) {
		t.Fatalf("expected errResponseCode, got %v", err)
	}
}

func TestWireDNSUDPAndTCPFallback(t *testing.T) {
	t.Parallel()

	udpConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer func() { _ = udpConn.Close() }()

	udpAddr := udpConn.LocalAddr().(*net.UDPAddr)
	tcpListener, err := net.Listen("tcp", udpAddr.String())
	if err != nil {
		t.Fatalf("listen tcp on same port %s: %v", udpAddr.String(), err)
	}
	defer func() { _ = tcpListener.Close() }()

	// UDP server: answers A directly, returns TC=1 for AAAA to exercise TCP fallback.
	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, readErr := udpConn.ReadFrom(buf)
			if readErr != nil {
				return
			}
			req := buf[:n]
			qtype := binary.BigEndian.Uint16(req[n-4 : n-2])
			var reply []byte
			if qtype == qtypeA {
				reply = buildMockAnswer(req, qtypeA, []byte{93, 184, 216, 34}, false)
			} else {
				reply = buildMockAnswer(req, qtypeAAAA, nil, true) // truncated -> triggers TCP
			}
			_, _ = udpConn.WriteTo(reply, addr)
		}
	}()

	// TCP server: answers AAAA with 2001:db8::1.
	go func() {
		for {
			conn, acceptErr := tcpListener.Accept()
			if acceptErr != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				var lb [2]byte
				if _, rerr := c.Read(lb[:]); rerr != nil {
					return
				}
				reqLen := int(binary.BigEndian.Uint16(lb[:]))
				req := make([]byte, reqLen)
				if _, rerr := c.Read(req); rerr != nil {
					return
				}
				ip6 := netip.MustParseAddr("2001:db8::1").As16()
				reply := buildMockAnswer(req, qtypeAAAA, ip6[:], false)
				framed := make([]byte, 2+len(reply))
				binary.BigEndian.PutUint16(framed[0:2], uint16(len(reply)))
				copy(framed[2:], reply)
				_, _ = c.Write(framed)
			}(conn)
		}
	}()

	_, port, err := net.SplitHostPort(udpAddr.String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}

	eng := &Engine{
		Servers: []Server{
			{
				Title:       "Local Mock DNS",
				Description: "Mock UDP/TCP resolver",
				Document:    "-",
				DoHEndpoint: "-",
				IP:          "127.0.0.1",
			},
			{
				Title:       "System Default",
				Description: "System resolver",
				Document:    "-",
				DoHEndpoint: "-",
				IP:          "-",
			},
		},
		DNSTimeout: 2 * time.Second,
		DNSPort:    port,
		SystemLookup: func(_ context.Context, network, _ string) ([]netip.Addr, error) {
			if network == "ip4" {
				return []netip.Addr{netip.MustParseAddr("192.0.2.10")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("2001:db8::10")}, nil
		},
	}

	results, err := eng.QueryAll(context.Background(), ModeDNS, "example.com")
	if err != nil {
		t.Fatalf("QueryAll failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !strings.Contains(results[0].Output, "ipv4:") || !strings.Contains(results[0].Output, "93.184.216.34") {
		t.Errorf("unexpected UDP A result: %q", results[0].Output)
	}
	if !strings.Contains(results[0].Output, "ipv6:") || !strings.Contains(results[0].Output, "2001:db8::1") {
		t.Errorf("unexpected TCP fallback AAAA result: %q", results[0].Output)
	}
	if !strings.Contains(results[1].Output, "192.0.2.10") || !strings.Contains(results[1].Output, "2001:db8::10") {
		t.Errorf("unexpected System Default result: %q", results[1].Output)
	}
}

func TestDoHQueriesAndErrorHandling(t *testing.T) {
	t.Parallel()

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/dns-json" {
			http.Error(w, "missing accept header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/dns-json")
		if r.URL.Query().Get("type") == "AAAA" {
			_, _ = w.Write([]byte(`{"Answer":[{"type":28,"data":"2001:db8::42"},{"type":5,"data":"not-an-ip"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"Answer":[{"type":1,"data":"93.184.216.34"},{"type":1,"data":"bad-ip"}]}`))
	}))
	defer okServer.Close()

	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusInternalServerError)
	}))
	defer errServer.Close()

	eng := &Engine{
		Servers: []Server{
			{
				Title:       "Mock DoH OK",
				Description: "Healthy DoH",
				Document:    "-",
				DoHEndpoint: okServer.URL,
				IP:          "1.1.1.1",
			},
			{
				Title:       "Mock DoH Fail",
				Description: "Failing DoH",
				Document:    "-",
				DoHEndpoint: errServer.URL,
				IP:          "8.8.8.8",
			},
			{
				Title:       "Legacy DNS Only",
				Description: "No DoH",
				Document:    "-",
				DoHEndpoint: "-",
				IP:          "9.9.9.9",
			},
		},
		DoHTimeout:       2 * time.Second,
		HTTPClient:       okServer.Client(),
		AllowInsecureDoH: true,
	}

	results, err := eng.QueryAll(context.Background(), ModeDoH, "example.com")
	if err != nil {
		t.Fatalf("QueryAll DoH failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 DoH-filtered results, got %d", len(results))
	}
	if !strings.Contains(results[0].Output, "93.184.216.34") || !strings.Contains(results[0].Output, "2001:db8::42") {
		t.Errorf("expected both IPv4 and IPv6 DoH records, got %q", results[0].Output)
	}
	if strings.Contains(results[0].Output, "bad-ip") || strings.Contains(results[0].Output, "not-an-ip") {
		t.Errorf("non-IP data leaked into DoH output: %q", results[0].Output)
	}
	if results[1].Output != "" {
		t.Errorf("expected empty output for HTTP 500 DoH server, got %q", results[1].Output)
	}
}

func buildMockAnswer(req []byte, rrType uint16, rdata []byte, truncated bool) []byte {
	out := make([]byte, len(req))
	copy(out, req)
	flags := uint16(0x8180)
	if truncated {
		flags |= 0x0200
	}
	binary.BigEndian.PutUint16(out[2:4], flags)
	if truncated || len(rdata) == 0 {
		binary.BigEndian.PutUint16(out[6:8], 0)
		return out
	}
	binary.BigEndian.PutUint16(out[6:8], 1)
	// Compression pointer to QNAME at offset 12.
	out = binary.BigEndian.AppendUint16(out, 0xC00C)
	out = binary.BigEndian.AppendUint16(out, rrType)
	out = binary.BigEndian.AppendUint16(out, qclassIN)
	out = binary.BigEndian.AppendUint32(out, 60)
	out = binary.BigEndian.AppendUint16(out, uint16(len(rdata)))
	out = append(out, rdata...)
	return out
}
