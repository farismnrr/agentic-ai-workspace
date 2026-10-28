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
type WorkRef struct {
	ID           string       `json:"id"`
	Actor        string       `json:"actor,omitempty"`
	Role         string       `json:"role,omitempty"`
	RoleName     string       `json:"roleName,omitempty"`
	From         string       `json:"from,omitempty"`
	Evidence     EvidenceKind `json:"evidence,omitempty"`
	Outcome      string       `json:"outcome,omitempty"`
	CallID       string       `json:"callId,omitempty"`
	Phase        string       `json:"phase,omitempty"`
	PlannedRoles []string     `json:"plannedRoles,omitempty"`
}

// WorkSummary is the safe, read-only work detail attached to a task response.
type WorkSummary struct {
	UpdatedAt       time.Time  `json:"updatedAt"`
	ReportedAt      *time.Time `json:"reportedAt,omitempty"`
	ID              string     `json:"id"`
	State           WorkState  `json:"state"`
	PlanState       PlanPhase  `json:"planState"`
	Role            string     `json:"role,omitempty"`
	RoleName        string     `json:"roleName,omitempty"`
	Actor           string     `json:"actor"`
	Decision        string     `json:"decision,omitempty"`
	HandoffState    string     `json:"handoffState,omitempty"`
	LinkedTaskID    string     `json:"linkedTaskId,omitempty"`
	Revision        int64      `json:"revision"`
	CurrentRevision int64      `json:"currentRevision"`
	PlanRev         int64      `json:"planRev"`
	Stale           bool       `json:"stale"`
}

// ExternalWork is the persisted identity and lifecycle record for a work session.
type ExternalWork struct {
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ReportedAt *time.Time
	ID         string
	TaskID     string
	UserID     string
	ProjectID  string
	Client     string
	State      WorkState
	PlanState  PlanPhase
	RoleKey    string
	RoleName   string
	IdemKey    string
	IdemHash   string
	EndedAt    int64
	Rev        int64
	PlanRev    int64
	NextStep   int64
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
	MaxWorkEvents       = 2000
	MaxOpenWorksUser    = 20
	MaxOpenWorksProject = 10
	BeginWorkPerMinute  = 10
	OtherWorkPerMinute  = 120
)
