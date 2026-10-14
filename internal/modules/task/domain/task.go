package domain

import "time"

// Status is where a task is in its life. The strings match the app's status labels.
type Status string

// Task statuses.
const (
	StatusQueued     Status = "queued"
	StatusPlanning   Status = "planning"
	StatusWorking    Status = "working"
	StatusNeedsInput Status = "needs-input"
	StatusReview     Status = "review"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// Running reports whether the agent is, or is about to be, working on the task.
func (s Status) Running() bool {
	return s == StatusQueued || s == StatusPlanning || s == StatusWorking
}

// Mode says how much the agent may do without asking.
type Mode string

// Approval modes: "risky" asks only before risky steps, "always" also before edits.
const (
	ModeRisky  Mode = "risky"
	ModeAlways Mode = "always"
)

// PlanState is the state of one plan step.
type PlanState string

// Plan step states.
const (
	PlanPending PlanState = "pending"
	PlanActive  PlanState = "active"
	PlanDone    PlanState = "done"
	PlanFailed  PlanState = "failed"
)

// PlanStep is one line of the agent's plan, in plain words.
type PlanStep struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Detail string    `json:"detail,omitempty"`
	State  PlanState `json:"state"`
}

// ProofRow is one labelled fact on the proof tab.
type ProofRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Check is something that was tested. Verified checks come from tasks the
// server ran itself; the rest are only claimed by the agent.
type Check struct {
	Text     string `json:"text"`
	Verified bool   `json:"verified"`
	Passed   bool   `json:"passed"`
}

// Proof is the evidence offered for a finished task.
type Proof struct {
	Rows   []ProofRow `json:"rows"`
	Checks []Check    `json:"checks"`
	// Proven is true only when a task the server ran passed after the last change.
	Proven bool `json:"proven"`
}

// Task is one request a person gave the agent, inside a project.
type Task struct {
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Proof      *Proof
	ID         string
	ProjectID  string
	UserID     string
	Title      string
	Summary    string
	ProviderID string
	Model      string
	Status     Status
	Mode       Mode
	Plan       []PlanStep
}
