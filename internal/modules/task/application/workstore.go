package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// WorkStore persists external work sessions and atomically commits task changes and events.
type WorkStore interface {
	CreateWork(ctx context.Context, create WorkCreate) (domain.ExternalWork, error)
	Commit(ctx context.Context, change WorkChange) ([]domain.Event, int64, error)
	FindWork(ctx context.Context, userID, client, projectID, workID string) (domain.ExternalWork, error)
	FindIdempotent(ctx context.Context, userID, client, projectID, key string) (domain.ExternalWork, bool, error)
	FindOp(ctx context.Context, workID, opKey string) (WorkOp, bool, error)
	StaleOpen(ctx context.Context, userID string, before time.Time) ([]domain.ExternalWork, error)
	StaleOpenAll(ctx context.Context, before time.Time) ([]domain.ExternalWork, error)
	InterruptRunningCalls(ctx context.Context, before *time.Time) ([]domain.ExternalWork, error)
	CountOpen(ctx context.Context, userID, projectID string) (int, error)
	ByTask(ctx context.Context, taskID string) (domain.ExternalWork, error)
}

// WorkCallStore records MCP calls with the work role read in the same transaction as the event.
type WorkCallStore interface {
	AppendWorkCall(
		ctx context.Context, userID, client, projectID, workID string, event domain.Event, reported bool,
	) (string, domain.Event, error)
	UpdateWorkCall(ctx context.Context, taskID, workID, eventID string, event domain.Event) (domain.Event, error)
}

// WorkCreate is one atomic task, work, and ordered initial-event creation request.
type WorkCreate struct {
	Task   domain.Task
	Events []domain.Event
	Work   domain.ExternalWork
}

// WorkOp is the saved response for an idempotent operation on one work session.
type WorkOp struct {
	Response string
	WorkID   string
	OpKey    string
	Tool     string
	ReqHash  string
	At       int64
}

// WorkPatch contains optional changes to the external work record.
type WorkPatch struct {
	State           *domain.WorkState
	PlanState       *domain.PlanPhase
	RoleKey         *string
	RoleName        *string
	PlanRev         *int64
	NextStep        *int64
	EndedAt         *time.Time
	ClearReportedAt bool
	SupersedePlan   bool
}

// TaskPatch contains optional changes to the task fields owned by work operations.
type TaskPatch struct {
	Status   *domain.Status
	Plan     *[]domain.PlanStep
	PRD      *string
	Research *string
	Summary  *string
	Proof    *domain.Proof
}

// WorkChange commits a work revision, optional idempotency response, task update, and events atomically.
type WorkChange struct {
	WorkID    string
	UserID    string
	Op        *WorkOp
	Task      *TaskPatch
	Patch     *WorkPatch
	Events    []domain.Event
	ExpectRev int64
}
