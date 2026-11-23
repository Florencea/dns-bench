package main

import (
	"net"
	"testing"
)

func TestListenLoopbackFallbackWhenDefaultOccupied(t *testing.T) {
	t.Parallel()

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer func() { _ = occupied.Close() }()

	// Explicit addr must fail when occupied.
	if ln, err := listenLoopback(occupied.Addr().String(), true); err == nil {
		_ = ln.Close()
		t.Fatal("expected explicit occupied addr to fail")
	}

	// Non-explicit default must fall back to an ephemeral loopback port.
	fallback, err := listenLoopback(occupied.Addr().String(), false)
	if err != nil {
		t.Fatalf("expected fallback listener, got error: %v", err)
	}
	defer func() { _ = fallback.Close() }()

	if fallback.Addr().String() == occupied.Addr().String() {
		t.Fatalf("fallback bound to the same occupied address %s", occupied.Addr().String())
	}
}

func TestOpenBrowserRejectsNonLoopback(t *testing.T) {
	t.Parallel()

	for _, bad := range []string{"https://example.com/", "http://example.com/", "javascript:alert(1)", "file:///etc/passwd"} {
		if err := openBrowser(bad); err == nil {
			t.Errorf("expected openBrowser(%q) to reject non-loopback URL", bad)
		}
	}
}
