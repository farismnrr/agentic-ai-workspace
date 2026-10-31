package mcp

import (
	"context"
	"encoding/json"
)

// WorkSessions is the narrow MCP-owned port for explicit connected-app work.
type WorkSessions interface {
	Validate(ctx context.Context, userID, client, projectID, workID string) error
	RequireRole(ctx context.Context, userID, client, projectID, workID, role string) error
	AvailableRoles(ctx context.Context, userID string) ([]WorkRole, error)
	Begin(ctx context.Context, userID, client, projectID string, in BeginWorkInput) (BeginWorkResult, error)
	SetRole(ctx context.Context, userID, client, projectID, workID, role, brief string) (RoleWorkResult, error)
	UpdatePlan(ctx context.Context, userID, client, projectID, workID string, in PlanWorkInput) (PlanWorkResult, error)
	SubmitPlan(ctx context.Context, userID, client, projectID, workID string, in SubmitPlanInput) (SubmitPlanResult, error)
	FinishResearch(ctx context.Context, userID, client, projectID, workID, summary string) (ResearchResult, error)
	AskQuestion(ctx context.Context, userID, client, projectID, workID, requestID string,
		questions json.RawMessage) (QuestionResult, error)
	ApprovePlan(ctx context.Context, userID, client, projectID, workID string,
		revision int64, idempotencyKey string) (PlanApprovalResult, error)
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
	Role           string
	IdempotencyKey string
	NewTask        bool
	PlannedRoles   []string
}

type PlanWorkStep struct {
	ID     string `json:"id,omitempty" jsonschema:"unique step ID (at most 80 chars); generated if omitted"`
	Title  string `json:"title" jsonschema:"step title; required, at most 200 chars"`
	Detail string `json:"detail,omitempty" jsonschema:"step details or instructions; at most 1000 chars"`
	State  string `json:"state" jsonschema:"step state: pending, active, done, or failed"`
}

type PlanWorkInput struct {
	RequestID    string         `json:"request_id,omitempty"`
	Steps        []PlanWorkStep `json:"steps"`
	BaseRevision int64          `json:"base_revision"`
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

type WorkState struct {
	Review       *PlanReviewInfo `json:"review,omitempty"`
	Research     string          `json:"research_preview,omitempty"`
	Role         string          `json:"role,omitempty"`
	State        string          `json:"state"`
	PlanState    string          `json:"plan_state"`
	NextAction   string          `json:"next_action"`
	WorkID       string          `json:"work_id"`
	PRD          string          `json:"prd_preview,omitempty"`
	TaskID       string          `json:"-"`
	Questions    QuestionGroups  `json:"questions"`
	Progress     []string        `json:"progress"`
	Plan         []PlanWorkStep  `json:"plan"`
	Warnings     []string        `json:"warnings"`
	Evidence     EvidenceCounts  `json:"evidence"`
	Revision     int64           `json:"revision"`
	ResearchSize int             `json:"research_size"`
	PRDSize      int             `json:"prd_size"`
	Stale        bool            `json:"stale"`
}

type SubmitPlanInput struct {
	Summary      string `json:"summary"`
	RequestID    string `json:"request_id,omitempty"`
	BaseRevision int64  `json:"base_revision"`
	SmallTask    bool   `json:"small_task,omitempty"`
}

type SubmitPlanResult struct {
	WorkID       string   `json:"work_id"`
	State        string   `json:"state"`
	PlanState    string   `json:"plan_state"`
	Summary      string   `json:"summary"`
	Status       string   `json:"status"`
	Warnings     []string `json:"warnings"`
	WorkRev      int64    `json:"revision"`
	PlanRevision int64    `json:"plan_revision"`
}

type PlanApprovalResult struct {
	Decision       string `json:"decision"`
	HandoffID      string `json:"handoff_id"`
	HandoffState   string `json:"handoff_state"`
	InternalTaskID string `json:"internal_task_id,omitempty"`
	Revision       int64  `json:"revision"`
}

type EvidenceCounts struct {
	Claimed  int `json:"claimed"`
	Observed int `json:"observed"`
	Ran      int `json:"ran"`
}

type WorkMutation struct {
	WorkID   string   `json:"work_id"`
	Path     string   `json:"-"`
	State    string   `json:"state"`
	Warnings []string `json:"warnings"`
	Revision int64    `json:"revision"`
}

type BeginWorkResult struct {
	Role        string     `json:"role"`
	TaskID      string     `json:"task_id"`
	WorkID      string     `json:"work_id"`
	Idempotency string     `json:"idempotency"`
	State       string     `json:"state"`
	Roles       []WorkRole `json:"roles"`
	Warnings    []string   `json:"warnings"`
	Revision    int64      `json:"revision"`
	Resumed     bool       `json:"resumed"`
}

type RoleWorkResult struct {
	Role     WorkRole `json:"role"`
	WorkID   string   `json:"work_id"`
	State    string   `json:"state"`
	Warnings []string `json:"warnings"`
	Revision int64    `json:"revision"`
	Changed  bool     `json:"changed"`
}

type PlanWorkResult struct {
	WorkID       string         `json:"work_id"`
	State        string         `json:"state"`
	Steps        []PlanWorkStep `json:"steps"`
	Warnings     []string       `json:"warnings"`
	Revision     int64          `json:"revision"`
	PlanRevision int64          `json:"plan_revision"`
}

type CompleteWorkResult struct {
	State      string       `json:"state"`
	TaskStatus string       `json:"task_status"`
	WorkID     string       `json:"work_id"`
	Warnings   []string     `json:"warnings"`
	Verified   VerifiedWork `json:"verified"`
	Revision   int64        `json:"revision"`
	Already    bool         `json:"already,omitempty"`
}

type VerifiedWork struct {
	FilesChanged  int  `json:"files_changed"`
	Runs          int  `json:"runs"`
	LastRunPassed bool `json:"last_run_passed"`
	Proven        bool `json:"proven"`
}
