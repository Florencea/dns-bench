package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Florencea/dns-bench/internal/dns"
	"github.com/Florencea/dns-bench/internal/i18n"
)

const DefaultAddr = "127.0.0.1:8080"

// Config holds runtime settings for dns-bench.
type Config struct {
	Addr          string
	ExplicitAddr  bool
	DefaultLocale string
	NoOpen        bool
	DNSTimeout    time.Duration
	DoHTimeout    time.Duration
}

// LoadFromEnv loads configuration defaults and environment overrides.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		Addr:          DefaultAddr,
		DefaultLocale: i18n.LocaleZhTW,
		NoOpen:        false,
		DNSTimeout:    dns.DefaultDNSTimeout,
		DoHTimeout:    dns.DefaultDoHTimeout,
	}

	if v := strings.TrimSpace(os.Getenv("ADDR")); v != "" {
		cfg.Addr = v
		cfg.ExplicitAddr = true
	}
	if v := strings.TrimSpace(os.Getenv("DEFAULT_LOCALE")); v != "" {
		cfg.DefaultLocale = v
	}
	if v := strings.TrimSpace(os.Getenv("NO_OPEN")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid NO_OPEN %q: %w", v, err)
		}
		cfg.NoOpen = b
	}
	if v := strings.TrimSpace(os.Getenv("DNS_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid DNS_TIMEOUT %q: %w", v, err)
		}
		cfg.DNSTimeout = d
	}
	if v := strings.TrimSpace(os.Getenv("DOH_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid DOH_TIMEOUT %q: %w", v, err)
		}
		cfg.DoHTimeout = d
	}
	return cfg, nil
}

// Validate enforces loopback-only binding, supported locale, and positive timeouts.
func (c Config) Validate() error {
	host, portStr, err := net.SplitHostPort(strings.TrimSpace(c.Addr))
	if err != nil {
		return fmt.Errorf("invalid addr %q: %w", c.Addr, err)
	}
	if !IsLoopbackHost(host) {
		return fmt.Errorf("addr %q must bind to a loopback host (127.0.0.1, ::1, or localhost)", c.Addr)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return fmt.Errorf("invalid port in addr %q", c.Addr)
	}
	if _, ok := i18n.ParseLocale(c.DefaultLocale); !ok {
		return fmt.Errorf("unsupported DEFAULT_LOCALE %q (supported: %s, %s)", c.DefaultLocale, i18n.LocaleZhTW, i18n.LocaleEnUS)
	}
	if c.DNSTimeout <= 0 || c.DNSTimeout > 30*time.Second {
		return errors.New("DNS_TIMEOUT must be between 1ms and 30s")
	}
	if c.DoHTimeout <= 0 || c.DoHTimeout > 30*time.Second {
		return errors.New("DOH_TIMEOUT must be between 1ms and 30s")
	}
	return nil
}

// IsLoopbackHost reports whether host is localhost or a loopback IP address.
func IsLoopbackHost(host string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(strings.Trim(host, "[]")))
	if trimmed == "localhost" {
		return true
	}
	addr, err := netip.ParseAddr(trimmed)
	if err != nil {
		return false
	}
	return addr.IsLoopback()
}
