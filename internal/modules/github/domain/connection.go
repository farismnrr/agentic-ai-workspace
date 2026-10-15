// Package domain holds what a connected GitHub account means to the app.
package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNotConfigured means the administrator has not set up the GitHub App.
	ErrNotConfigured = errors.New("github is not configured")
	// ErrNotConnected means the person has no usable GitHub connection.
	ErrNotConnected = errors.New("github is not connected")
	// ErrState means the sign-in answer does not match a sign-in this person started.
	ErrState = errors.New("github sign-in state is unknown or expired")
	// ErrRejected means GitHub refused a code or token: it is wrong, expired or revoked.
	ErrRejected = errors.New("github rejected the credentials")
)

// Profile is the GitHub account a person connected.
type Profile struct {
	Login string
	Name  string
	ID    int64
}

// DisplayName is the name commits are written under.
func (p Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}

	return p.Login
}

// NoreplyEmail is the private address GitHub accepts for commits, so no real email is needed.
func (p Profile) NoreplyEmail() string {
	return fmt.Sprintf("%d+%s@users.noreply.github.com", p.ID, p.Login)
}

// Tokens are the credentials GitHub issued. A zero expiry means the token never expires.
type Tokens struct {
	AccessExpires  time.Time
	RefreshExpires time.Time
	Access         string
	Refresh        string
}

// ExpiresWithin reports whether the access token ends before now+margin.
func (t Tokens) ExpiresWithin(now time.Time, margin time.Duration) bool {
	return !t.AccessExpires.IsZero() && t.AccessExpires.Before(now.Add(margin))
}

// Connection is a person's connected account without any secret.
type Connection struct {
	Profile Profile
}

// Credential is what a command needs to act as the person on GitHub.
type Credential struct {
	Token   string
	Profile Profile
}
