package domain

import "time"

// PlanRevisionStatus records the person-facing decision state of a snapshot.
type PlanRevisionStatus string

const (
	PlanRevisionSubmitted        PlanRevisionStatus = "submitted"
	PlanRevisionSuperseded       PlanRevisionStatus = "superseded"
	PlanRevisionChangesRequested PlanRevisionStatus = "changes_requested"
	PlanRevisionApproved         PlanRevisionStatus = "approved"
)

// PlanRevision is an immutable submitted plan snapshot.
type PlanRevision struct {
	CreatedAt time.Time          `json:"createdAt"`
	QA        string             `json:"qa,omitempty"`
	Summary   string             `json:"summary"`
	TaskID    string             `json:"taskId"`
	Status    PlanRevisionStatus `json:"status"`
	EventID   string             `json:"eventId"`
	Digest    string             `json:"digest"`
	PRD       string             `json:"prd,omitempty"`
	Research  string             `json:"research,omitempty"`
	ID        string             `json:"id"`
	WorkID    string             `json:"workId"`
	Steps     []PlanStep         `json:"steps"`
	RevNo     int64              `json:"revision"`
	PlanRev   int64              `json:"planRevision"`
	SmallTask bool               `json:"smallTask"`
}

// PlanReview is the read-only data shown on a plan review card.
type PlanReview struct {
	DecidedAt    time.Time          `json:"decidedAt,omitempty"`
	Summary      string             `json:"summary"`
	Source       string             `json:"source,omitempty"`
	Status       PlanRevisionStatus `json:"status"`
	Decision     string             `json:"decision,omitempty"`
	HandoffID    string             `json:"handoffId,omitempty"`
	HandoffState string             `json:"handoffState,omitempty"`
	TaskID       string             `json:"taskId,omitempty"`
	LinkedTaskID string             `json:"linkedTaskId,omitempty"`
	Steps        []PlanStep         `json:"steps,omitempty"`
	Revision     int64              `json:"revision"`
	SmallTask    bool               `json:"smallTask,omitempty"`
}
