// Package domain holds the rules of sites: web apps the AI starts in the sandbox and
// the person opens at /run/<name>/<token>/.
package domain

import (
	"errors"
	"strings"
	"time"
)

// Visibility says who may open a site's pages.
type Visibility string

// Visibilities. A new site is always private.
const (
	Private Visibility = "private"
	Public  Visibility = "public"
)

// Status is whether the site's container is up.
type Status string

// Statuses.
const (
	Running Status = "running"
	Stopped Status = "stopped"
)

// Site errors.
var (
	ErrNotFound    = errors.New("site not found")
	ErrBadPort     = errors.New("use a port between 1024 and 65535")
	ErrBadCommand  = errors.New("give the command that starts the app")
	ErrBadChoice   = errors.New("visibility must be private or public")
	ErrTooMany     = errors.New("you already have the most sites running; stop one first")
	ErrServerBusy  = errors.New("the server is running as many sites as it can; try again later")
	ErrNotStarted  = errors.New("the app did not start answering in time")
	ErrNoSandboxes = errors.New("sites need Docker, which the server cannot use yet")
)

// Port limits: unprivileged ports only.
const (
	MinPort = 1024
	MaxPort = 65535
)

// Site is one web app of a project. Token is a random UUID that is part of its address and
// changes on every start, so an old link stops working.
type Site struct {
	StartedAt   time.Time
	ProjectID   string
	UserID      string
	Slug        string
	Token       string
	ContainerID string
	Visibility  Visibility
	Status      Status
	Port        int
}

// ValidPort checks a port the app will listen on.
func ValidPort(port int) error {
	if port < MinPort || port > MaxPort {
		return ErrBadPort
	}

	return nil
}

// ParseVisibility reads a visibility the person chose.
func ParseVisibility(value string) (Visibility, error) {
	switch v := Visibility(strings.ToLower(strings.TrimSpace(value))); v {
	case Private, Public:
		return v, nil
	}

	return "", ErrBadChoice
}

// BasePath is the address prefix the app is served under, without a trailing slash.
func (s Site) BasePath() string {
	return "/run/" + s.Slug + "/" + s.Token
}

// CanView decides whether a request may be answered. document is true for a page the
// browser navigates to and false for what a page loads itself (scripts, styles, fetches).
//
// Pages of a private site need the signed-in owner. A page's own loads cannot: the browser
// runs it in a sandbox without the person's login, so they are protected by the token in
// the address instead, which only someone who opened the site has.
func (s Site) CanView(userID string, document bool) bool {
	switch {
	case s.Status != Running:
		return false
	case s.Visibility == Public:
		return true
	case !document:
		return true
	}

	return userID != "" && userID == s.UserID
}
