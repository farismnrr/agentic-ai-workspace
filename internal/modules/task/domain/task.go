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
	Text string `json:"text"`
	// By names the teammate who ran the check; empty for the lead and for tasks without a team.
	By       string `json:"by,omitempty"`
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
	ProviderID string
	// Source names the connected app the thread was recorded from; empty for tasks the agent ran.
	Source  string
	UserID  string
	Title   string
	Summary string
	// PRD is the product requirements document the team builds from; empty until a teammate writes one.
	PRD   string
	ID    string
	Model string
	// ProjectID is the project the task works in.
	ProjectID string
	// Effort is how hard the model was asked to think; empty is the provider's default.
	Effort Effort
	Status Status
	Plan   []PlanStep
	// ContextUsed and ContextWindow are tokens: what the last model call carried and what the model can hold.
	// Zero until the first call.
	ContextUsed   int64
	ContextWindow int64
	// Team is true when the agent hands work to teammates instead of doing it all itself.
	Team bool
}
