package bridge

import (
	"fmt"
	"net"
	"net/netip"
)

// ValidateBindAddress checks that addr is safe to bind. Loopback addresses are
// always allowed. Private (RFC-1918 / ULA) addresses require allowPrivate=true.
// Wildcard and public addresses are always rejected.
func ValidateBindAddress(addr string, allowPrivate bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid bind address %q: %w", addr, err)
	}

	// "localhost" is unambiguously loopback.
	if host == "localhost" {
		return nil
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		// Non-IP hostnames other than "localhost" are rejected.
		return fmt.Errorf("bind address %q: non-loopback hostname not allowed", addr)
	}

	if ip.IsLoopback() {
		return nil
	}

	if ip.IsUnspecified() {
		return fmt.Errorf("bind address %q: wildcard bind not allowed", addr)
	}

	if ip.IsPrivate() {
		if allowPrivate {
			return nil
		}
		return fmt.Errorf("bind address %q: private non-loopback bind requires allowPrivate=true", addr)
	}

	return fmt.Errorf("bind address %q: public bind not allowed", addr)
}

// LocalhostURLForBind returns an http:// URL for the given bind address.
func LocalhostURLForBind(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid bind address %q: %w", addr, err)
	}
	return fmt.Sprintf("http://%s:%s", host, port), nil
}
