package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Request is one conversation turn sent to a model.
type Request struct {
	Model    string
	System   string
	Messages []domain.Message
	Tools    []domain.ToolDef
}

// Reply is what the model answered: text and tool calls, plus what it cost.
type Reply struct {
	Blocks []domain.Block
	Usage  domain.Usage
}

// Model sends a conversation to a model provider. Failures are *domain.ModelError.
type Model interface {
	Complete(ctx context.Context, request Request) (Reply, error)
}

// Models finds the model client for a person's provider and their default choice.
type Models interface {
	Client(ctx context.Context, userID, providerID string) (Model, error)
	Default(ctx context.Context, userID string) (providerID, model string, err error)
}

// Persona is how the person wants the agent to behave.
type Persona struct {
	Name, TalkStyle, Language, About string
	Rules                            []string
}

// PersonaSource loads a person's persona and enabled rules.
type PersonaSource interface {
	Persona(ctx context.Context, userID string) (Persona, error)
}

// Projects resolves a project the person owns.
type Projects interface {
	Name(ctx context.Context, userID, projectID string) (string, error)
}

// Call is a tool call the model made.
type Call struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// Described is a tool call in words: Text says what was done, Ask is the same
// as a question for approvals, and Technical is the detail for the curious.
type Described struct {
	Text      string
	Ask       string
	Technical string
}

// ToolResult is what running a tool produced. Tool failures are results, not Go errors,
// so the model can read them and adapt.
type ToolResult struct {
	Run *domain.RunInfo
	// Note is a short extra for the technical line, such as an exit code.
	Note    string
	Output  string
	Files   []string
	Undo    []byte
	IsError bool
}

// ToolSet runs the project tools for the agent.
type ToolSet interface {
	Definitions() []domain.ToolDef
	Risk(name string) domain.Risk
	Describe(call Call) Described
	Execute(ctx context.Context, projectID string, call Call) ToolResult
	Undo(ctx context.Context, projectID string, blob []byte) error
}

// TaskStore persists tasks.
type TaskStore interface {
	Create(ctx context.Context, task domain.Task) error
	// Find loads a task; an empty userID skips the ownership check (server use only).
	Find(ctx context.Context, userID, id string) (domain.Task, error)
	List(ctx context.Context, userID, projectID string, limit int) ([]domain.Task, error)
	SetStatus(ctx context.Context, id string, status domain.Status, at time.Time) error
	SavePlan(ctx context.Context, id string, plan []domain.PlanStep, at time.Time) error
	Finish(ctx context.Context, id, summary string, proof *domain.Proof, at time.Time) error
	// Running lists tasks left queued, planning or working, for recovery after a restart.
	Running(ctx context.Context) ([]domain.Task, error)
}

// EventStore persists the thread.
type EventStore interface {
	AppendEvent(ctx context.Context, taskID string, event domain.Event) (domain.Event, error)
	EventsAfter(ctx context.Context, taskID string, seq int64) ([]domain.Event, error)
	FindEvent(ctx context.Context, taskID, eventID string) (domain.Event, error)
	EventByToolUse(ctx context.Context, taskID, toolUseID string) (domain.Event, bool, error)
	UpdateEvent(ctx context.Context, taskID string, event domain.Event) error
}

// TranscriptStore persists the model-facing conversation, one block at a time.
type TranscriptStore interface {
	AppendBlock(ctx context.Context, taskID string, role domain.Role, block domain.Block) error
	LoadTranscript(ctx context.Context, taskID string) ([]domain.Message, error)
}

// UsageRecord is the cost of one model call.
type UsageRecord struct {
	At         time.Time
	TaskID     string
	UserID     string
	ProviderID string
	Model      string
	Usage      domain.Usage
}

// UsageRow is usage summed per model.
type UsageRow struct {
	ProviderID string
	Model      string
	Tasks      int
	Input      int64
	Output     int64
}

// UsageStore persists and summarises token usage.
type UsageStore interface {
	AddUsage(ctx context.Context, record UsageRecord) error
	TotalUsage(ctx context.Context, taskID string) (domain.Usage, error)
	UsageSummary(ctx context.Context, userID string, since time.Time) ([]UsageRow, error)
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}
