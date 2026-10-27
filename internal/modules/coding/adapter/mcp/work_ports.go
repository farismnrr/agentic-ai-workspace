package mcp

import "context"

// WorkSessions is the narrow MCP-owned port for explicit connected-app work.
type WorkSessions interface {
	Validate(ctx context.Context, userID, client, projectID, workID string) error
	Begin(ctx context.Context, userID, client, projectID string, in BeginWorkInput) (BeginWorkResult, error)
	SetRole(ctx context.Context, userID, client, projectID, workID, role, brief string) (RoleWorkResult, error)
	UpdatePlan(ctx context.Context, userID, client, projectID, workID string, in PlanWorkInput) (PlanWorkResult, error)
	SaveResearch(ctx context.Context, userID, client, projectID, workID string, in ResearchWorkInput) (WorkMutation, error)
	SaveRequirements(ctx context.Context, userID, client, projectID, workID, markdown string) (WorkMutation, error)
	Progress(ctx context.Context, userID, client, projectID, workID string, in ProgressWorkInput) (WorkMutation, error)
	Complete(
		ctx context.Context, userID, client, projectID, workID string, in CompleteWorkInput,
	) (CompleteWorkResult, error)
	State(ctx context.Context, userID, client, projectID, workID string) (WorkState, error)
}

type BeginWorkInput struct {
	Summary        string
	IdempotencyKey string
	PlannedRoles   []string
}

type PlanWorkStep struct {
	ID     string `json:"id,omitempty"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	State  string `json:"state"`
}

//nolint:govet // fieldalignment: preserve the port's input field order.
type PlanWorkInput struct {
	BaseRevision int64          `json:"base_revision"`
	RequestID    string         `json:"request_id,omitempty"`
	Steps        []PlanWorkStep `json:"steps"`
}

type ResearchWorkInput struct {
	Markdown string   `json:"markdown"`
	Sources  []string `json:"sources,omitempty"`
}

type ProgressWorkInput struct {
	Message string   `json:"message"`
	Claims  []string `json:"claims,omitempty"`
}

type CompleteWorkInput struct {
	Outcome string   `json:"outcome"`
	Summary string   `json:"summary"`
	Claims  []string `json:"claims,omitempty"`
}

type WorkRole struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
}

//nolint:govet // fieldalignment: preserve the compact state output field order.
type WorkState struct {
	TaskID       string         `json:"-"`
	WorkID       string         `json:"work_id"`
	Revision     int64          `json:"revision"`
	State        string         `json:"state"`
	Stale        bool           `json:"stale"`
	Plan         []PlanWorkStep `json:"plan"`
	Progress     []string       `json:"progress"`
	Role         string         `json:"role,omitempty"`
	Research     string         `json:"research_preview,omitempty"`
	PRD          string         `json:"prd_preview,omitempty"`
	Evidence     EvidenceCounts `json:"evidence"`
	ResearchSize int            `json:"research_size"`
	PRDSize      int            `json:"prd_size"`
	Warnings     []string       `json:"warnings"`
}

type EvidenceCounts struct {
	Claimed  int `json:"claimed"`
	Observed int `json:"observed"`
	Ran      int `json:"ran"`
}

//nolint:govet // fieldalignment: preserve the work mutation output field order.
type WorkMutation struct {
	WorkID   string   `json:"work_id"`
	Path     string   `json:"-"`
	Revision int64    `json:"revision"`
	State    string   `json:"state"`
	Warnings []string `json:"warnings"`
}

//nolint:govet // fieldalignment: preserve the public begin response field order.
type BeginWorkResult struct {
	TaskID      string     `json:"task_id"`
	WorkID      string     `json:"work_id"`
	Roles       []WorkRole `json:"roles"`
	Resumed     bool       `json:"resumed"`
	Idempotency string     `json:"idempotency"`
	Revision    int64      `json:"revision"`
	State       string     `json:"state"`
	Warnings    []string   `json:"warnings"`
}

//nolint:govet // fieldalignment: preserve the public role response field order.
type RoleWorkResult struct {
	WorkID   string   `json:"work_id"`
	Revision int64    `json:"revision"`
	State    string   `json:"state"`
	Role     WorkRole `json:"role"`
	Changed  bool     `json:"changed"`
	Warnings []string `json:"warnings"`
}

//nolint:govet // fieldalignment: preserve the public plan response field order.
type PlanWorkResult struct {
	WorkID       string         `json:"work_id"`
	Revision     int64          `json:"revision"`
	State        string         `json:"state"`
	PlanRevision int64          `json:"plan_revision"`
	Steps        []PlanWorkStep `json:"steps"`
	Warnings     []string       `json:"warnings"`
}

//nolint:govet // fieldalignment: preserve the public completion response field order.
type CompleteWorkResult struct {
	WorkID     string       `json:"work_id"`
	Revision   int64        `json:"revision"`
	State      string       `json:"state"`
	TaskStatus string       `json:"task_status"`
	Verified   VerifiedWork `json:"verified"`
	Warnings   []string     `json:"warnings"`
}

type VerifiedWork struct {
	FilesChanged  int  `json:"files_changed"`
	Runs          int  `json:"runs"`
	LastRunPassed bool `json:"last_run_passed"`
	Proven        bool `json:"proven"`
}
