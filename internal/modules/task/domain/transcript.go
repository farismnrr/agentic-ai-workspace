package domain

import (
	"encoding/json"
	"errors"
)

// Role of a transcript message.
type Role string

// Roles.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// BlockKind is the type of a content block.
type BlockKind string

// Block kinds, shared by every model provider.
const (
	BlockText       BlockKind = "text"
	BlockToolUse    BlockKind = "tool_use"
	BlockToolResult BlockKind = "tool_result"
)

// Block is one piece of a message in the provider-neutral transcript.
type Block struct {
	Kind    BlockKind       `json:"kind"`
	Text    string          `json:"text,omitempty"`
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	IsError bool            `json:"isError,omitempty"`
}

// Message is a turn of the conversation with the model.
type Message struct {
	Role   Role
	Blocks []Block
}

// ToolDef describes a tool to the model.
type ToolDef struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

// Usage counts the tokens of one model call.
type Usage struct {
	Input  int64
	Output int64
}

// ModelError is returned by model clients; Retryable marks rate limits and outages.
type ModelError struct {
	Message   string
	Retryable bool
}

// Error implements error.
func (e *ModelError) Error() string {
	return e.Message
}

// AsModelError extracts a model error from err.
func AsModelError(err error) (*ModelError, bool) {
	var modelErr *ModelError

	ok := errors.As(err, &modelErr)

	return modelErr, ok
}
