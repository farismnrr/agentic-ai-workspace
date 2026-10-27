package domain

import "time"

// EventKind is what a thread entry is. These are what the person reads.
type EventKind string

// Event kinds.
const (
	EventRequest  EventKind = "request"
	EventMessage  EventKind = "message"
	EventAction   EventKind = "action"
	EventQuestion EventKind = "question"
	EventApproval EventKind = "approval"
	EventError    EventKind = "error"
	EventReceipt  EventKind = "receipt"
	EventSite     EventKind = "site"
	// EventHandoff marks work passing to a teammate, or coming back from one.
	EventHandoff EventKind = "handoff"
	// EventShot is a screenshot of the running app; Image names the saved PNG and Text says what it shows.
	EventShot EventKind = "shot"
	// EventResearch is research notes a teammate wrote; its text is the document.
	EventResearch EventKind = "research"
	// EventExtension marks a browser extension packaged for the person; URL is where they install it.
	EventExtension EventKind = "extension"
	// EventPlan marks the plan (the team's todo list) being set or changed; Text says how far it is and Technical
	// lists the steps.
	EventPlan EventKind = "plan"
	// EventPRD is the product requirements document a teammate wrote; its text is the document.
	EventPRD EventKind = "prd"
	// EventRole records an external assistant changing its working role.
	EventRole EventKind = "role"
	// EventPlanReview is the person's review card for an external plan revision.
	EventPlanReview EventKind = "plan_review"
)

// Risk grades how careful the agent must be before a step.
type Risk string

// Risk levels; RiskNone steps never need approval.
const (
	RiskNone   Risk = ""
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// Resolution of an approval.
const (
	Allowed = "allowed"
	Denied  = "denied"
)

// RunInfo records a task the server ran for the agent; proof is built from these.
type RunInfo struct {
	Name     string `json:"name"`
	ExitCode int    `json:"exitCode"`
	TimedOut bool   `json:"timedOut,omitempty"`
}

// Event is one entry of a task's thread. Fields beyond Kind and Text only
// apply to some kinds, matching what the thread renders.
type Event struct {
	Time       time.Time   `json:"time"`
	Run        *RunInfo    `json:"run,omitempty"`
	Work       *WorkRef    `json:"work,omitempty"`
	PlanReview *PlanReview `json:"planReview,omitempty"`
	ID         string      `json:"id"`
	Kind       EventKind   `json:"kind"`
	Text       string      `json:"text"`
	Technical  string      `json:"technical,omitempty"`
	Action     string      `json:"action,omitempty"`
	Risk       Risk        `json:"risk,omitempty"`
	Resolved   string      `json:"resolved,omitempty"`
	Answered   string      `json:"answered,omitempty"`
	Hint       string      `json:"hint,omitempty"`
	// Model is the AI model a teammate works with, shown on its handoffs (EventHandoff only).
	Model string `json:"model,omitempty"`
	// Effort is how hard that model was asked to think, shown beside Model: the level it works at, or the one the
	// orchestrator asked for when the model has no level of its own (EventHandoff only).
	Effort string `json:"effort,omitempty"`
	// Agent names the teammate who did the step; empty for the orchestrator or a task without a team.
	Agent string `json:"agent,omitempty"`
	// URL is where a running site can be opened (EventSite only).
	URL string `json:"url,omitempty"`
	// Image names the saved screenshot (EventShot only).
	Image string `json:"image,omitempty"`
	// ToolUseID links an approval or question to the model's pending tool call.
	ToolUseID string   `json:"-"`
	Options   []string `json:"options,omitempty"`
	// Questions is the form of a question event; Options only fills in for questions saved before forms.
	Questions []Question `json:"questions,omitempty"`
	Documents []string   `json:"documents,omitempty"`
	Files     []string   `json:"files,omitempty"`
	// Undo describes how to reverse the step; only the owning tools understand it.
	Undo   []byte `json:"-"`
	Undone bool   `json:"undone,omitempty"`
	// Seq keeps the display order; Version advances when an event is inserted or updated.
	Seq     int64 `json:"seq"`
	Version int64 `json:"version"`
}

// CanUndo reports whether the step made a reversible change that still stands.
func (e Event) CanUndo() bool {
	return len(e.Undo) > 0 && !e.Undone
}
