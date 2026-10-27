package domain

import "time"

// WorkState is the durable lifecycle state of one external work session.
type WorkState string

// External work states.
const (
	WorkOpen      WorkState = "open"
	WorkBlocked   WorkState = "blocked"
	WorkCompleted WorkState = "completed"
	WorkFailed    WorkState = "failed"
	WorkStopped   WorkState = "stopped"
	WorkAbandoned WorkState = "abandoned"
)

// EvidenceKind describes how a work outcome was established.
type EvidenceKind string

// Evidence sources distinguish a report from observations made by the server.
const (
	EvidenceClaimed  EvidenceKind = "claimed"
	EvidenceObserved EvidenceKind = "observed"
	EvidenceRan      EvidenceKind = "ran"
)

// WorkRef attributes an event to an external assistant's current work session.
//
//nolint:govet // fieldalignment: planned role keys remain a JSON list in the event contract.
type WorkRef struct {
	PlannedRoles []string     `json:"plannedRoles,omitempty"`
	ID           string       `json:"id"`
	Actor        string       `json:"actor,omitempty"`
	Role         string       `json:"role,omitempty"`
	RoleName     string       `json:"roleName,omitempty"`
	From         string       `json:"from,omitempty"`
	Evidence     EvidenceKind `json:"evidence,omitempty"`
	Outcome      string       `json:"outcome,omitempty"`
	CallID       string       `json:"callId,omitempty"`
	Phase        string       `json:"phase,omitempty"`
}

// WorkSummary is the safe, read-only work detail attached to a task response.
type WorkSummary struct {
	UpdatedAt time.Time `json:"updatedAt"`
	ID        string    `json:"id"`
	State     WorkState `json:"state"`
	Role      string    `json:"role,omitempty"`
	RoleName  string    `json:"roleName,omitempty"`
	Actor     string    `json:"actor"`
	Revision  int64     `json:"revision"`
	PlanRev   int64     `json:"planRev"`
	Stale     bool      `json:"stale"`
}

// ExternalWork is the persisted identity and lifecycle record for a work session.
type ExternalWork struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        string
	TaskID    string
	UserID    string
	ProjectID string
	Client    string
	State     WorkState
	RoleKey   string
	RoleName  string
	IdemKey   string
	IdemHash  string
	EndedAt   int64
	Rev       int64
	PlanRev   int64
	NextStep  int64
}

// External work inputs are bounded before they are persisted or exported.
const (
	MaxWorkSummary      = 4000
	MaxWorkBrief        = 1000
	MaxWorkSteps        = 40
	MaxWorkMarkdown     = 12000
	MaxWorkSources      = 30
	MaxWorkProgress     = 2000
	MaxWorkClaims       = 10
	MaxWorkClaim        = 300
	MinRequestID        = 8
	MaxRequestID        = 64
	MaxPlannedRoles     = 10
	MaxOpenWorksUser    = 20
	MaxOpenWorksProject = 10
	BeginWorkPerMinute  = 10
	OtherWorkPerMinute  = 120
)
