package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	MaxVariables       = 50
	MaxValueSize       = 4096
	MaxSecretValueSize = 16 * 1024
	MaxNameSize        = 64
)

var (
	ErrInvalidName  = errors.New("invalid environment variable name")
	ErrReserved     = errors.New("reserved environment variable name")
	ErrInvalidValue = errors.New("invalid environment variable value")
	ErrLimit        = errors.New("environment variable limit reached")
	ErrNotFound     = errors.New("environment variable not found")
	namePattern     = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

// Variable is the safe, decrypted form used by application consumers.
type Variable struct {
	UpdatedAt time.Time
	Name      string
	Value     string
	IsSecret  bool
}

// ValidateName checks the portable name format and blocks base-environment overrides.
func ValidateName(name string) error {
	if reserved(name) {
		return ErrReserved
	}
	if len(name) > MaxNameSize || !namePattern.MatchString(name) {
		return ErrInvalidName
	}

	return nil
}

// ValidateValue limits values by their encoded byte length.
func ValidateValue(value string) error {
	if len(value) > MaxValueSize || hasForbiddenControl(value) {
		return ErrInvalidValue
	}

	return nil
}

// ValidateSecretValue allows bounded server-managed secrets such as PEM material.
func ValidateSecretValue(value string) error {
	if len(value) > MaxSecretValueSize || hasForbiddenSecretControl(value) {
		return ErrInvalidValue
	}
	return nil
}

func hasForbiddenSecretControl(value string) bool {
	for _, char := range value {
		if char != '\t' && char != '\n' && char != '\r' && unicode.IsControl(char) {
			return true
		}
	}
	return false
}

func hasForbiddenControl(value string) bool {
	for _, char := range value {
		if char != '\t' && unicode.IsControl(char) {
			return true
		}
	}
	return false
}

func reserved(name string) bool {
	lower := strings.ToLower(name)
	if lower == "path" || lower == "home" || lower == "http_proxy" || lower == "https_proxy" ||
		lower == "no_proxy" || lower == "all_proxy" {
		return true
	}

	return strings.HasPrefix(lower, "ld_") || dangerousGoName(name) ||
		strings.HasPrefix(lower, "docker_")
}

func dangerousGoName(name string) bool {
	if strings.HasPrefix(name, "CGO_") {
		return true
	}
	switch name {
	case "GOFLAGS", "GOPATH", "GOROOT", "GOPROXY", "GOENV", "GOMODCACHE", "GOCACHE",
		"GOTOOLCHAIN", "GONOSUMDB", "GONOPROXY", "GOPRIVATE", "GOINSECURE", "GOSUMDB",
		"GOWORK", "GOEXPERIMENT", "GODEBUG", "GOOS", "GOARCH":
		return true
	default:
		return false
	}
}
