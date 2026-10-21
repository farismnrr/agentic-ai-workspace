// Package egress limits where the server may open outgoing connections. Several
// features call addresses a person chose (model providers, client metadata), so
// without a guard a person could aim the server at services only it can reach
// (SSRF). The check runs when the connection is opened, on the address DNS
// actually returned, so a hostname that later resolves elsewhere cannot slip by.
package egress

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// Rule says which destinations a connection may reach.
type Rule int

const (
	// PublicOnly allows the public internet and nothing internal.
	PublicOnly Rule = iota
	// LocalOnly allows only this computer (loopback), never this server's own port.
	LocalOnly
	// PrivateAllowed allows public, private, and loopback destinations.
	PrivateAllowed
)

// Policy is a Rule plus the port this server listens on, which is always off limits.
type Policy struct {
	OwnPort string
	Rule    Rule
}

var (
	// ErrBlocked means the destination is not allowed by the policy.
	ErrBlocked = errors.New("connection blocked: that address is not allowed")

	// special lists ranges that are routable-looking but never a public service.
	special = mustPrefixes(
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
		"64:ff9b::/96", "64:ff9b:1::/48", "2001:db8::/32", "2002::/16",
		"fd00:ec2::254/128",
	)
)

const (
	dialTimeout = 10 * time.Second
	idleTimeout = 90 * time.Second
	tlsTimeout  = 10 * time.Second
)

// Transport returns an HTTP transport that enforces the policy on every connection.
// It ignores proxy environment variables, which would route around the check.
func Transport(policy Policy) *http.Transport {
	return &http.Transport{
		Proxy:               nil,
		DialContext:         Dialer(policy).DialContext,
		ForceAttemptHTTP2:   true,
		IdleConnTimeout:     idleTimeout,
		TLSHandshakeTimeout: tlsTimeout,
	}
}

// Dialer returns a net.Dialer that enforces policy against the resolved IP at
// connection time. Use its DialContext for non-HTTP clients such as DB drivers.
func Dialer(policy Policy) *net.Dialer {
	return &net.Dialer{
		Timeout: dialTimeout,
		Control: func(_, address string, _ syscall.RawConn) error { return Check(policy, address) },
	}
}

// Check reports whether a resolved "ip:port" destination is allowed by the policy.
func Check(policy Policy, address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBlocked, err)
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBlocked, err)
	}

	ip = ip.Unmap()
	if ownPort(policy, port) {
		return ErrBlocked
	}
	return checkAddressRule(policy.Rule, ip)
}

func ownPort(policy Policy, port string) bool { return policy.OwnPort != "" && port == policy.OwnPort }

func checkAddressRule(rule Rule, ip netip.Addr) error {
	switch rule {
	case LocalOnly:
		if ip.IsLoopback() {
			return nil
		}
		return ErrBlocked
	case PrivateAllowed:
		if privateAllowed(ip) {
			return nil
		}
		return ErrBlocked
	case PublicOnly:
		return checkPublic(ip)
	default:
		return ErrBlocked
	}
}

func privateAllowed(ip netip.Addr) bool {
	return (ip.IsGlobalUnicast() || ip.IsLoopback()) && !ip.IsLinkLocalUnicast() && !isSpecial(ip)
}

func checkPublic(ip netip.Addr) error {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || isSpecial(ip) {
		return ErrBlocked
	}

	return nil
}

func isSpecial(ip netip.Addr) bool {
	for _, prefix := range special {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}

func mustPrefixes(values ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		out = append(out, netip.MustParsePrefix(value))
	}

	return out
}
