package application

import (
	"context"

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

// EffortLevels says which effort levels a model takes, so the lead's choice can be fitted to a teammate's model.
type EffortLevels interface {
	EffortLevels(ctx context.Context, userID, providerID, model string) []string
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
	// Tier is the model and effort the person set for the named tier (TierThinking or TierFast); ok is false when
	// they set none, so the default model is used.
	Tier(ctx context.Context, userID, tier string) (providerID, model string, effort domain.Effort, ok bool)
}

// The model tiers a role can use. A person sets each once; a role picks one, so changing a model changes every role
// that uses it. The default tier is the person's default model.
const (
	TierDefault  = "default"
	TierThinking = "thinking"
	TierFast     = "fast"
)

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
	Execute(ctx context.Context, scope ToolScope, call Call) ToolResult
	Undo(ctx context.Context, projectID string, blob []byte) error
}

// Computer describes the isolated computer where the agent's commands run.
type Computer interface {
	// Toolchains lists the languages and tools installed there, such as "Go 1.26"; empty when unknown.
	Toolchains(ctx context.Context) []string
	// GitHub is the login of the GitHub account this person connected, or "" when they have none.
	GitHub(ctx context.Context, userID string) string
}
