package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// TaskStore persists tasks.
type TaskStore interface {
	Create(ctx context.Context, task domain.Task) error
	CreateWithRequest(ctx context.Context, task domain.Task, event domain.Event, prompt string) error
	FollowUp(ctx context.Context, userID, taskID string, event domain.Event, prompt string,
		mode domain.ReviewMode, at time.Time) error
	// Find loads a task; an empty userID skips the ownership check (server use only).
	Find(ctx context.Context, userID, id string) (domain.Task, error)
	Pin(ctx context.Context, userID, id string, at time.Time) (domain.Task, error)
	Unpin(ctx context.Context, userID, id string) (domain.Task, error)
	List(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error)
	Archived(ctx context.Context, userID string, limit int) ([]domain.Task, error)
	All(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error)
	Archive(ctx context.Context, userID, id string, at time.Time) error
	Restore(ctx context.Context, userID, id string) error
	DeleteOwned(ctx context.Context, userID string, ids []string) ([]string, error)
	SetModel(ctx context.Context, id, providerID, model string, effort domain.Effort) error
	SetStatus(ctx context.Context, id string, status domain.Status, at time.Time) error
	// FailUnlessDone prevents an asynchronously cancelled runner from reopening a manually stopped task.
	FailUnlessDone(ctx context.Context, id string, at time.Time) (bool, error)
	SavePlan(ctx context.Context, id string, plan []domain.PlanStep, at time.Time) error
	SavePRD(ctx context.Context, id, prd string, at time.Time) error
	SaveResearch(ctx context.Context, id, research string, at time.Time) error
	Finish(ctx context.Context, id, summary string, proof *domain.Proof, at time.Time) error
	SetContext(ctx context.Context, id string, used, window int64) error
	SetContextState(ctx context.Context, id, state string) error
	Running(ctx context.Context) ([]domain.Task, error)
	DeleteProject(ctx context.Context, userID, projectID string) error
	DeleteAll(ctx context.Context, userID string) error
}

// LinkedTaskStore creates the internal chat and external-task link atomically.
type LinkedTaskStore interface {
	CreateLinkedTask(ctx context.Context, externalID string, task domain.Task, event domain.Event,
		prompt string) (domain.Task, bool, error)
}

// EventStore persists the thread.
type EventStore interface {
	AppendEvent(ctx context.Context, taskID string, event domain.Event) (domain.Event, error)
	EventsAfter(ctx context.Context, taskID string, seq int64) ([]domain.Event, error)
	EventsChangedAfter(ctx context.Context, taskID string, version int64) ([]domain.Event, error)
	FindEvent(ctx context.Context, taskID, eventID string) (domain.Event, error)
	EventByToolUse(ctx context.Context, taskID, toolUseID string) (domain.Event, bool, error)
	UpdateEvent(ctx context.Context, taskID string, event domain.Event) error
}

// TranscriptStore persists the model-facing conversation, one block at a time.
type TranscriptStore interface {
	AppendBlock(ctx context.Context, taskID string, role domain.Role, block domain.Block) error
	LoadTranscript(ctx context.Context, taskID string) ([]domain.Message, error)
}

type AgentSessionCleanup interface {
	RevokeTask(userID, taskID string)
}

type UsageRecord struct {
	At         time.Time
	TaskID     string
	UserID     string
	ProviderID string
	Model      string
	Usage      domain.Usage
}

type UsageRow struct {
	ProviderID string
	Model      string
	Tasks      int
	Input      int64
	Output     int64
}

type UsageStore interface {
	AddUsage(ctx context.Context, record UsageRecord) error
	UsageSummary(ctx context.Context, userID string, since time.Time) ([]UsageRow, error)
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}
