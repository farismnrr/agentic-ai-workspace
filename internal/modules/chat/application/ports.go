package application

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
)

// ChatStore persists project conversations and messages.
type ChatStore interface {
	FindOrCreate(ctx context.Context, userID, projectID string) (domain.Chat, error)
	Get(ctx context.Context, userID, chatID string) (domain.Chat, error)
	Messages(ctx context.Context, chatID string) ([]domain.Message, error)
	Append(ctx context.Context, msg domain.Message) error
	Clear(ctx context.Context, chatID string) error
	Touch(ctx context.Context, chatID string, at time.Time) error
}

// ModelMessage is one message sent to a language model.
type ModelMessage struct {
	Role    domain.Role
	Content string
}

// ModelRequest is a completion request for chat.
type ModelRequest struct {
	OnText   func(delta string)
	Model    string
	System   string
	Messages []ModelMessage
}

// Model communicates with an AI provider.
type Model interface {
	Complete(ctx context.Context, req ModelRequest) (string, error)
}

// Models resolves model clients and default settings.
type Models interface {
	Client(ctx context.Context, userID, providerID string) (Model, error)
	Default(ctx context.Context, userID string) (providerID, model string, err error)
}

// Projects resolves project names.
type Projects interface {
	Name(ctx context.Context, userID, projectID string) (string, error)
}

// Persona is the custom persona configuration.
type Persona struct {
	Name      string
	TalkStyle string
	Language  string
	About     string
	Rules     []string
}

// PersonaSource provides the user's active persona and rules.
type PersonaSource interface {
	Persona(ctx context.Context, userID string) (Persona, error)
}

// TaskInput is the payload to create a new task from chat.
type TaskInput struct {
	ProjectID string
	Prompt    string
	Mode      string
}

// TaskCreator allows chat to escalate/handover into a task.
type TaskCreator interface {
	CreateTask(ctx context.Context, userID string, in TaskInput) (string, error)
}

// IDs generates unique identifiers.
type IDs interface {
	ID() (string, error)
}
