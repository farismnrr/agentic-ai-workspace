// Package domain holds the rules of sites: web apps the AI starts in the sandbox and
// the person opens at /run/<project-id>/.
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
	ErrNotFound     = errors.New("site not found")
	ErrBadPort      = errors.New("use a port between 1024 and 65535")
	ErrBadCommand   = errors.New("give the command that starts the app")
	ErrNeverStarted = errors.New("this app has not been started before; ask the AI to start it once")
	ErrBadChoice    = errors.New("visibility must be private or public")
	ErrTooMany      = errors.New("you already have the most apps running; stop one on the Sites page first")
	ErrServerBusy   = errors.New("the server is running as many apps as it can; stop one on the Sites page or try later")
	ErrNotStarted   = errors.New("the app did not start answering in time")
	ErrNoSandboxes  = errors.New("sites need Docker, which the server cannot use yet")
)

// Port limits: unprivileged ports only.
const (
	MinPort = 1024
	MaxPort = 65535
)

// Site is one web app of a project. Slug is its address and is the project's ID, which is unique across the
// server and never changes, so the address stays the same whenever the app is stopped and started again or the
// project is renamed. Token is a random UUID that never appears in the address: it keys the tickets and still
// opens addresses made before names were unique. Command is how the app was last started.
type Site struct {
	StartedAt   time.Time
	ProjectID   string
	UserID      string
	Slug        string
	Command     string
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
	return "/run/" + s.Slug
}

// Access describes one request to a site.
type Access struct {
	// UserID is the signed-in person, or empty.
	UserID string
	// Page is true for a GET the browser navigates to, a page the person opens; false for
	// what a page loads itself (scripts, styles, fetches) and for form posts.
	Page bool
	// Ticket is true when the address carries a valid, unexpired ticket.
	Ticket bool
}

// Verdict is what to do with a request.
type Verdict int

// Verdicts.
const (
	// Allow forwards the request to the app.
	Allow Verdict = iota
	// AskSignIn answers a page request with a page that asks the person to sign in.
	AskSignIn
	// Deny answers 403.
	Deny
	// Missing answers 404: the site is not running.
	Missing
)

// Decide applies the rules. A public site answers anyone. A private site answers its
// signed-in owner, and otherwise only a request carrying a ticket that is not a page: the
// browser runs a page in a sandbox without the person's login, so what the page loads is
// authorised by the ticket the owner's visit put in the address. A page itself always
// needs the owner, and the uuid alone never opens anything.
func (s Site) Decide(a Access) Verdict {
	switch {
	case s.Status != Running:
		return Missing
	case s.Visibility == Public:
		return Allow
	case a.UserID != "" && a.UserID == s.UserID:
		return Allow
	case a.Page:
		return AskSignIn
	case a.Ticket:
		return Allow
	}

	return Deny
}
