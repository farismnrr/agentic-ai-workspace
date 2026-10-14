package domain

import (
	"crypto/subtle"
	"errors"
	"net"
	"net/url"
	"slices"
)

// Client authentication methods (RFC 7591).
const (
	AuthMethodNone              = "none"
	AuthMethodClientSecretBasic = "client_secret_basic"
	AuthMethodClientSecretPost  = "client_secret_post"
)

// ClientSource records how a client became known.
type ClientSource string

// Client sources.
const (
	SourceDCR  ClientSource = "dcr"
	SourceCIMD ClientSource = "cimd"
)

// ErrBadRedirectURI means a redirect URI is not acceptable for any client.
var ErrBadRedirectURI = errors.New("redirect URI must be https (or http on a loopback address) without a fragment")

// Client is a registered OAuth client (public or confidential).
type Client struct {
	ID           string
	Name         string
	Source       ClientSource
	AuthMethod   string
	SecretHash   string
	RedirectURIs []string
}

// IsConfidential reports whether the client has registered client credentials.
func (c Client) IsConfidential() bool {
	return c.SecretHash != ""
}

// VerifySecret checks if secret matches the client's hashed secret in constant time.
func (c Client) VerifySecret(secret string, hasher func(string) string) bool {
	if c.SecretHash == "" || secret == "" || hasher == nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(hasher(secret)), []byte(c.SecretHash)) == 1
}

// AllowsRedirect reports whether uri exactly matches a registered redirect URI.
func (c Client) AllowsRedirect(uri string) bool {
	return slices.Contains(c.RedirectURIs, uri)
}

// ValidateRedirectURI enforces OAuth 2.1 §1.5: HTTPS, or http on loopback only.
func ValidateRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.Fragment != "" || parsed.User != nil {
		return ErrBadRedirectURI
	}

	if parsed.Scheme == "https" || (parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return nil
	}

	return ErrBadRedirectURI
}

// RedirectHost returns the host a redirect URI points to, for display.
func RedirectHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	return parsed.Host
}

// IsMetadataClientID reports whether id is a Client ID Metadata Document URL:
// https, with a path, and no fragment or credentials.
func IsMetadataClientID(id string) bool {
	parsed, err := url.Parse(id)

	return err == nil && parsed.Scheme == "https" && parsed.Host != "" &&
		parsed.Path != "" && parsed.Path != "/" && parsed.Fragment == "" && parsed.User == nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}
