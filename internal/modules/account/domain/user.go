package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MinPasswordLength matches the registration form.
	MinPasswordLength = 8
	// MaxPasswordLength bounds hashing cost for hostile input.
	MaxPasswordLength = 128
	maxNameLength     = 100
)

var (
	// ErrInvalidName means the display name is empty or too long.
	ErrInvalidName = errors.New("invalid name")
	// ErrInvalidEmail means the address is not a plain email address.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrWeakPassword means the password breaks the length policy.
	ErrWeakPassword = errors.New("password must be 8 to 128 characters")
	// ErrEmailTaken means an account already uses the email.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidCredentials is returned for any failed sign-in, never revealing which part was wrong.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserNotFound means no account matches the lookup.
	ErrUserNotFound = errors.New("user not found")
	// ErrNoSession means the session token is unknown or expired.
	ErrNoSession = errors.New("no active session")
)

// User is a registered person.
type User struct {
	CreatedAt time.Time
	ID        string
	Name      string
	Email     string
}

// NormalizeName trims and validates a display name.
func NormalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", ErrInvalidName
	}

	return name, nil
}

// NormalizeEmail lowercases and validates a bare email address.
func NormalizeEmail(raw string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil || address.Name != "" {
		return "", ErrInvalidEmail
	}

	return strings.ToLower(address.Address), nil
}

// ValidatePassword enforces the length policy.
func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < MinPasswordLength || len(password) > MaxPasswordLength {
		return ErrWeakPassword
	}

	return nil
}
