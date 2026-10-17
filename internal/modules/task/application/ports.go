package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Request is one conversation turn sent to a model.
type Request struct {
	// OnText, when set, asks the model to stream and is called with each piece of
	// text as it arrives. The full reply is still returned at the end.
	OnText func(delta string)
	Model  string
	System string
	// Effort is how hard the model should think; empty leaves it to the provider.
	Effort   domain.Effort
	Messages []domain.Message
	Tools    []domain.ToolDef
}

// Reply is what the model answered: text and tool calls, plus what it cost.
type Reply struct {
	Blocks []domain.Block
	Usage  domain.Usage
}

// ContextWindows says how much a model can hold at once, in tokens.
type ContextWindows interface {
	ContextWindow(ctx context.Context, userID, providerID, model string) int64
}

// Model sends a conversation to a model provider. Failures are *domain.ModelError.
type Model interface {
	Complete(ctx context.Context, request Request) (Reply, error)
}

// Models finds the model client for a person's provider and their default choice.
type Models interface {
	Client(ctx context.Context, userID, providerID string) (Model, error)
	Default(ctx context.Context, userID string) (providerID, model string, err error)
	// DefaultEffort is the effort the person prefers when they do not pick one; empty when none.
	DefaultEffort(ctx context.Context, userID string) domain.Effort
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

// ToolGate is the person's switchboard for tools, shared with the apps they connected: a tool
// switched off is neither offered to the agent nor run, and skipping approvals lets the
// agent run the enabled ones without asking.
type ToolGate interface {
	// Enabled says, for every tool, whether the person switched it on.
	Enabled(ctx context.Context, userID string) (map[string]bool, error)
	SkipApprovals(ctx context.Context, userID string) (bool, error)
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
	Note string
	// Link, when set, is an address the person can open, such as a running site; the thread shows it.
	Link   string
	Output string
	// Image is a PNG the tool produced, such as a screenshot; the model and the person both see it.
	Image   []byte
	Files   []string
	Undo    []byte
	IsError bool
}

// Step is one thing the agent did that changed or ran something, for the person's activity log.
type Step struct {
	UserID    string
	ProjectID string
	TaskID    string
	Tool      string
	Summary   string
	Detail    string
	Failed    bool
}

// ActivitySink receives every such step. A failure to record must never stop the agent.
type ActivitySink interface {
	Step(ctx context.Context, step Step)
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
	// SavePRD records the product requirements document the team builds from.
	SavePRD(ctx context.Context, id, prd string, at time.Time) error
	Finish(ctx context.Context, id, summary string, proof *domain.Proof, at time.Time) error
	// SetContext records how full the model's context window is, in tokens.
	SetContext(ctx context.Context, id string, used, window int64) error
	// Running lists tasks left queued, planning or working, for recovery after a restart.
	Running(ctx context.Context) ([]domain.Task, error)
	// DeleteAll removes every task a person has, with its thread.
	DeleteAll(ctx context.Context, userID string) error
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
	UsageSummary(ctx context.Context, userID string, since time.Time) ([]UsageRow, error)
}

// IDs mints identifiers.
type IDs interface {
	ID() (string, error)
}

// Computer describes the isolated computer where the agent's commands run.
type Computer interface {
	// Toolchains lists the languages and tools installed there, such as "Go 1.26"; empty when unknown.
	Toolchains(ctx context.Context) []string
	// GitHub is the login of the GitHub account this person connected, or "" when they have none.
	GitHub(ctx context.Context, userID string) string
}
