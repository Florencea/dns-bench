package config

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := Config{
		Addr:          "127.0.0.1:8080",
		DefaultLocale: "zh-TW",
		DNSTimeout:    2500 * time.Millisecond,
		DoHTimeout:    3000 * time.Millisecond,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	validIPv6 := valid
	validIPv6.Addr = "[::1]:0"
	if err := validIPv6.Validate(); err != nil {
		t.Fatalf("expected valid IPv6 loopback config, got %v", err)
	}

	nonLoopback := valid
	nonLoopback.Addr = "0.0.0.0:8080"
	if err := nonLoopback.Validate(); err == nil {
		t.Fatal("expected non-loopback bind 0.0.0.0 to be rejected")
	}

	badLocale := valid
	badLocale.DefaultLocale = "fr-FR"
	if err := badLocale.Validate(); err == nil {
		t.Fatal("expected unsupported locale to be rejected")
	}

	badTimeout := valid
	badTimeout.DNSTimeout = 0
	if err := badTimeout.Validate(); err == nil {
		t.Fatal("expected zero DNS timeout to be rejected")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	t.Parallel()

	for _, h := range []string{"127.0.0.1", "localhost", "LOCALHOST", "::1", "[::1]", "127.0.0.2"} {
		if !IsLoopbackHost(h) {
			t.Errorf("expected %q to be loopback", h)
		}
	}
	for _, h := range []string{"", "0.0.0.0", "192.168.1.10", "example.com", "127.0.0.1.evil.com"} {
		if IsLoopbackHost(h) {
			t.Errorf("expected %q to be non-loopback", h)
		}
	}
}
