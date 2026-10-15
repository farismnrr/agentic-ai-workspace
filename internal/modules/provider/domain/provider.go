package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// Kind is the wire format a provider speaks.
type Kind string

// Provider kinds. Gemini is spoken through Google's OpenAI-compatible endpoint, and
// "compatible" is any other OpenAI-compatible gateway, such as 9router.
const (
	KindAnthropic  Kind = "anthropic"
	KindOpenAI     Kind = "openai"
	KindGemini     Kind = "gemini"
	KindCompatible Kind = "compatible"
)

// Errors of the provider use cases.
var (
	ErrInvalidKind    = errors.New("provider kind must be anthropic, openai, gemini or compatible")
	ErrInvalidName    = errors.New("give the provider a name of up to 60 characters")
	ErrInvalidBaseURL = errors.New("address must be on this computer, for example http://localhost:20128/v1")
	ErrAddressFixed   = errors.New("this provider has a fixed address; leave the address empty")
	ErrInvalidKey     = errors.New("enter the API key")
	ErrNotFound       = errors.New("provider not found")
	ErrVerifyFailed   = errors.New("the provider did not accept this key")
	ErrTooMany        = errors.New("too many providers; remove one first")
)

const (
	maxNameLength = 60
	hintMinLength = 8
	hintTail      = 4
	hintShortTail = 2
)

// Provider is a model provider a person added. The API key is never part of it.
type Provider struct {
	CreatedAt time.Time
	ID        string
	UserID    string
	Name      string
	BaseURL   string
	KeyHint   string
	Kind      Kind
	// ContextWindows holds the sizes a person chose per model; see ContextWindow.
	ContextWindows map[string]int64
	// ReportedWindows are the sizes the provider itself listed when its key was verified.
	ReportedWindows map[string]int64
	Models          []string
	Verified        bool
	Enabled         bool
}

// Credential is what a model client needs to call the provider.
type Credential struct {
	Kind    Kind
	BaseURL string
	APIKey  string
}

// Valid reports whether k is a known kind.
func (k Kind) Valid() bool {
	return k == KindAnthropic || k == KindOpenAI || k == KindGemini || k == KindCompatible
}

// DefaultBaseURL is where the hosted providers live.
func DefaultBaseURL(kind Kind) string {
	switch kind {
	case KindAnthropic:
		return "https://api.anthropic.com"
	case KindOpenAI:
		return "https://api.openai.com/v1"
	case KindGemini:
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case KindCompatible:
		return ""
	}

	return ""
}

// NormalizeName trims and validates a display name.
func NormalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", ErrInvalidName
	}

	return name, nil
}

// CustomAddress reports whether people choose this kind's address themselves. Only
// gateways do, and only ones running on this computer; hosted providers are fixed.
func (k Kind) CustomAddress() bool {
	return k == KindCompatible
}

// NormalizeBaseURL fills in a hosted provider's fixed address and validates a
// gateway's. The server calls this address itself, so a gateway must be on this
// computer (the connection is also checked when it is opened; see platform/egress).
func NormalizeBaseURL(kind Kind, raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")

	if !kind.CustomAddress() {
		if fixed := DefaultBaseURL(kind); raw == "" || raw == fixed {
			return fixed, nil
		}

		return "", ErrAddressFixed
	}

	parsed, err := url.Parse(raw)
	if err != nil || !isLocalGateway(parsed) {
		return "", ErrInvalidBaseURL
	}

	return raw, nil
}

// isLocalGateway accepts a plain http(s) address on this computer, without credentials or extras.
func isLocalGateway(parsed *url.URL) bool {
	plain := parsed.User == nil && parsed.Fragment == "" && parsed.RawQuery == ""
	web := parsed.Scheme == "http" || parsed.Scheme == "https"

	return plain && web && isLocalHost(parsed.Hostname())
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// KeyHint is the only part of a key that is ever shown again.
func KeyHint(key string) string {
	runes := []rune(key)
	if len(runes) <= hintMinLength {
		return "…" + string(runes[len(runes)-min(len(runes), hintShortTail):])
	}

	return string(runes[:3]) + "…" + string(runes[len(runes)-hintTail:])
}
