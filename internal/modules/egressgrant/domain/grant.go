package domain

import (
	"errors"
	"strings"
	"time"
)

// Errors of the egress grant rules.
var (
	ErrInvalidRequest = errors.New("the request is missing something or is not valid")
	ErrNotAuthorized  = errors.New("only a project owner or an admin may decide on a request")
	ErrNotPending     = errors.New("that request has already been decided or has expired")
	ErrNotApproved    = errors.New("that grant is not active")
	ErrTTLTooLong     = errors.New("that lifetime is longer than a grant may last")
)

// Lifetimes. A pending request that nobody decides on also stops counting after PendingTTL.
const (
	DefaultTTL      = 24 * time.Hour
	MaxTTL          = 7 * 24 * time.Hour
	PendingTTL      = 24 * time.Hour
	maxReasonLength = 500
)

// Status is where a grant is in its life. StatusExpired is never stored: Effective derives it.
type Status string

// The states of a grant.
const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusDenied   Status = "denied"
	StatusRevoked  Status = "revoked"
	StatusExpired  Status = "expired"
)

// ActorKind says who asked. Only a person can decide; an agent can ask.
type ActorKind string

// The kinds of requester.
const (
	ActorAgent  ActorKind = "agent"
	ActorPerson ActorKind = "person"
)

// Actor is who made a request.
type Actor struct {
	Kind ActorKind
	ID   string
}

// Decider is a person allowed to approve, deny or revoke. The HTTP adapter builds one from the signed-in session
// and nothing else does, so a request made by an agent never carries the right to decide. Owner means the owner of
// ProjectID only: the right never reaches another project's grants. Admin is the one right that spans projects.
type Decider struct {
	ID        string
	ProjectID string
	Owner     bool
	Admin     bool
}

func (d Decider) allowed(g Grant) bool {
	return d.ID != "" && (d.Admin || (d.Owner && d.ProjectID != "" && d.ProjectID == g.ProjectID))
}

// Grant is one task's access to one extra HTTPS host (port 443, public addresses only).
type Grant struct {
	CreatedAt   time.Time
	DecidedAt   time.Time
	ExpiresAt   time.Time
	ID          string
	ProjectID   string
	TaskID      string
	Host        string
	Reason      string
	RequestedBy Actor
	DecidedBy   string
	Status      Status
}

// Request starts a grant in the pending state. It is the only thing an agent can do.
func Request(id, projectID, taskID, host, reason string, by Actor, now time.Time) (Grant, error) {
	reason = strings.TrimSpace(reason)
	known := by.Kind == ActorAgent || by.Kind == ActorPerson

	if id == "" || projectID == "" || taskID == "" || by.ID == "" || !known ||
		reason == "" || len(reason) > maxReasonLength {
		return Grant{}, ErrInvalidRequest
	}

	normal, err := NormalizeHost(host)
	if err != nil {
		return Grant{}, err
	}

	return Grant{
		ID: id, ProjectID: projectID, TaskID: taskID, Host: normal, Reason: reason, RequestedBy: by,
		Status: StatusPending, CreatedAt: now, ExpiresAt: now.Add(PendingTTL),
	}, nil
}

// Effective is the status at a moment, with an approval or request past its end shown as expired.
func (g Grant) Effective(now time.Time) Status {
	if (g.Status == StatusApproved || g.Status == StatusPending) && !now.Before(g.ExpiresAt) {
		return StatusExpired
	}

	return g.Status
}

// Active reports whether the grant may open new connections now.
func (g Grant) Active(now time.Time) bool { return g.Effective(now) == StatusApproved }

// Covers reports whether this grant lets that task of that project reach that host now. The host is normalized
// again, so the check never depends on how the caller spelled it.
func (g Grant) Covers(projectID, taskID, host string, now time.Time) bool {
	normal, err := NormalizeHost(host)

	return err == nil && g.Active(now) && g.ProjectID == projectID && g.TaskID == taskID && g.Host == normal
}

// Approve activates a pending request for ttl (DefaultTTL when zero), counted from now.
func (g Grant) Approve(by Decider, ttl time.Duration, now time.Time) (Grant, error) {
	if !by.allowed(g) {
		return Grant{}, ErrNotAuthorized
	}

	if g.Effective(now) != StatusPending {
		return Grant{}, ErrNotPending
	}

	if ttl == 0 {
		ttl = DefaultTTL
	}

	if ttl < 0 || ttl > MaxTTL {
		return Grant{}, ErrTTLTooLong
	}

	return g.decided(StatusApproved, by, now, now.Add(ttl)), nil
}

// Deny closes a pending request.
func (g Grant) Deny(by Decider, now time.Time) (Grant, error) {
	if !by.allowed(g) {
		return Grant{}, ErrNotAuthorized
	}

	if g.Effective(now) != StatusPending {
		return Grant{}, ErrNotPending
	}

	return g.decided(StatusDenied, by, now, g.ExpiresAt), nil
}

// Revoke ends an active grant at once.
func (g Grant) Revoke(by Decider, now time.Time) (Grant, error) {
	if !by.allowed(g) {
		return Grant{}, ErrNotAuthorized
	}

	if !g.Active(now) {
		return Grant{}, ErrNotApproved
	}

	return g.decided(StatusRevoked, by, now, now), nil
}

func (g Grant) decided(status Status, by Decider, now, expires time.Time) Grant {
	g.Status, g.DecidedBy, g.DecidedAt, g.ExpiresAt = status, by.ID, now, expires

	return g
}
