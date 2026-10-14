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

// Provider kinds. "compatible" is any OpenAI-compatible gateway, such as 9router.
const (
	KindAnthropic  Kind = "anthropic"
	KindOpenAI     Kind = "openai"
	KindCompatible Kind = "compatible"
)

// Errors of the provider use cases.
var (
	ErrInvalidKind    = errors.New("provider kind must be anthropic, openai or compatible")
	ErrInvalidName    = errors.New("give the provider a name of up to 60 characters")
	ErrInvalidBaseURL = errors.New("base URL must be an http(s) address, for example http://localhost:20128/v1")
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
	Models    []string
	Verified  bool
	Enabled   bool
}

// Credential is what a model client needs to call the provider.
type Credential struct {
	Kind    Kind
	BaseURL string
	APIKey  string
}

// Valid reports whether k is a known kind.
func (k Kind) Valid() bool {
	return k == KindAnthropic || k == KindOpenAI || k == KindCompatible
}

// DefaultBaseURL is where the hosted providers live.
func DefaultBaseURL(kind Kind) string {
	switch kind {
	case KindAnthropic:
		return "https://api.anthropic.com"
	case KindOpenAI:
		return "https://api.openai.com/v1"
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

// NormalizeBaseURL fills in the default address and validates a custom one.
func NormalizeBaseURL(kind Kind, raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		raw = DefaultBaseURL(kind)
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" {
		return "", ErrInvalidBaseURL
	}

	return raw, nil
}

// KeyHint is the only part of a key that is ever shown again.
func KeyHint(key string) string {
	runes := []rune(key)
	if len(runes) <= hintMinLength {
		return "…" + string(runes[len(runes)-min(len(runes), hintShortTail):])
	}

	return string(runes[:3]) + "…" + string(runes[len(runes)-hintTail:])
}
