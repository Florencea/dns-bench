package dns

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrDomainRequired = errors.New("domain_required")
	ErrDomainTooLong  = errors.New("domain_too_long")
	ErrDomainInvalid  = errors.New("domain_invalid")
)

var domainRegexp = regexp.MustCompile(`^(localhost|[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*)$`)

// ValidateDomain trims and validates a hostname against RFC label bounds.
func ValidateDomain(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrDomainRequired
	}
	if len(trimmed) > 253 {
		return "", ErrDomainTooLong
	}
	if !domainRegexp.MatchString(trimmed) {
		return "", ErrDomainInvalid
	}
	return trimmed, nil
}

// ParseMode normalizes a query mode string, defaulting to ModeDNS.
func ParseMode(raw string) Mode {
	if strings.TrimSpace(raw) == string(ModeDoH) {
		return ModeDoH
	}
	return ModeDNS
}

// IsValidMode reports whether raw is an exact supported mode identifier.
func IsValidMode(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed == string(ModeDNS) || trimmed == string(ModeDoH)
}
