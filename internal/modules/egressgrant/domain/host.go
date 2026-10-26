// Package domain holds the rules for letting one task's sandbox reach an extra HTTPS host: what a valid host is,
// who may approve, how long an approval lasts, and when it stops counting.
package domain

import (
	"errors"
	"net/netip"
	"strings"

	"golang.org/x/net/idna"
)

// ErrInvalidHost means the text is not a public DNS name a grant may name.
var ErrInvalidHost = errors.New("that is not a valid public host name")

const (
	maxHostLength  = 253
	maxLabelLength = 63
	minLabels      = 2
)

// forbiddenChars rules out wildcards, ports, paths, user info, IPv6 and bracket forms, zones and spaces before any
// parsing, so nothing but a bare name can get through.
const forbiddenChars = "*/\\?#@:[]%<>\"'`,;=+ \t\r\n"

// internalSuffixes are names that point inside a network, never at the public internet.
var internalSuffixes = []string{
	".localhost", ".local", ".internal", ".lan", ".home.arpa", ".localdomain", ".intranet", ".private",
}

// NormalizeHost returns the one spelling of a host that grants are stored and matched by: lower case, IDNA in its
// ASCII form, one trailing dot removed. Surrounding spaces are refused, not trimmed: callers trim what a person
// typed. It refuses wildcards, IP addresses in any notation, single labels and
// internal-looking names. The proxy runs the same function on every request, so a grant never depends on how the
// caller spelled the name.
func NormalizeHost(raw string) (string, error) {
	host := raw
	if host == "" || strings.ContainsAny(host, forbiddenChars) {
		return "", ErrInvalidHost
	}

	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return "", ErrInvalidHost
	}

	ascii = strings.TrimSuffix(strings.ToLower(ascii), ".")

	if !validShape(ascii) || isInternal(ascii) {
		return "", ErrInvalidHost
	}

	return ascii, nil
}

// validShape checks lengths, empty labels and that the name cannot be read as an IP address by any resolver.
func validShape(host string) bool {
	if host == "" || len(host) > maxHostLength {
		return false
	}

	if _, err := netip.ParseAddr(host); err == nil {
		return false
	}

	labels := strings.Split(host, ".")
	if len(labels) < minLabels {
		return false
	}

	for _, label := range labels {
		if label == "" || len(label) > maxLabelLength {
			return false
		}
	}

	return realTLD(labels[len(labels)-1])
}

// realTLD rejects last labels that numeric parsers read as part of an address ("127.1", "0x7f.1", "010.1.1.1"):
// a real top-level domain has a letter and is not a hexadecimal number.
func realTLD(label string) bool {
	if strings.HasPrefix(label, "0x") || !strings.ContainsAny(label, "abcdefghijklmnopqrstuvwxyz") {
		return false
	}

	return true
}

func isInternal(host string) bool {
	if host == "localhost" {
		return true
	}

	for _, suffix := range internalSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}
