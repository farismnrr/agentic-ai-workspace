package application

import (
	"encoding/json"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

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
