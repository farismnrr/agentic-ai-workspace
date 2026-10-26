package egress

import (
	"net/netip"
	"strings"

	"golang.org/x/net/idna"
)

const (
	maxHostLength  = 253
	maxLabelLength = 63
	minHostLabels  = 2
	// notHostChars rules out wildcards, ports, paths, user info, IPv6 and bracket forms, zones and spaces.
	notHostChars = "*/\\?#@:[]%<>\"'`,;=+ \t\r\n"
)

// CanonicalHost returns the one spelling of a DNS name that the egress proxy compares: lower case, IDNA in its
// ASCII form, one trailing dot removed. It reports false for anything that is not a bare name of at least two
// labels, including every IP address notation, so a name that a resolver could read as an address never matches.
// The extra-grant rules in modules/egressgrant are stricter; a test in bootstrap keeps the two from drifting apart
// in the unsafe direction (this function must give the same spelling for every name those rules accept).
func CanonicalHost(raw string) (string, bool) {
	if raw == "" || strings.ContainsAny(raw, notHostChars) {
		return "", false
	}

	ascii, err := idna.Lookup.ToASCII(raw)
	if err != nil {
		return "", false
	}

	host := strings.TrimSuffix(strings.ToLower(ascii), ".")

	return host, hostShapeOK(host)
}

func hostShapeOK(host string) bool {
	if host == "" || len(host) > maxHostLength {
		return false
	}

	if _, err := netip.ParseAddr(host); err == nil {
		return false
	}

	labels := strings.Split(host, ".")
	if len(labels) < minHostLabels {
		return false
	}

	for _, label := range labels {
		if label == "" || len(label) > maxLabelLength {
			return false
		}
	}

	last := labels[len(labels)-1]

	return !strings.HasPrefix(last, "0x") && strings.ContainsAny(last, "abcdefghijklmnopqrstuvwxyz")
}
