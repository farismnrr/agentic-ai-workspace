package domain

import "time"

// PlanDecisionKind is the durable choice made for one submitted plan revision.
type PlanDecisionKind string

const (
	// PlanDecisionApproved approves the immutable snapshot and starts a handoff.
	PlanDecisionApproved PlanDecisionKind = "approved"
	// PlanDecisionChangesRequested sends the plan back to its planner.
	PlanDecisionChangesRequested PlanDecisionKind = "changes_requested"
)

// PlanDecision records who decided what for one immutable plan revision.
type PlanDecision struct {
	DecidedAt  time.Time
	ID         string
	RevisionID string
	WorkID     string
	TaskID     string
	UserID     string
	ProjectID  string
	Kind       PlanDecisionKind
	Source     string
	Client     string
	IdemKey    string
	ReqHash    string
	Note       string
}
