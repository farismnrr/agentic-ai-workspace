package application

import (
	"context"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type capturedResult struct {
	text   string
	failed bool
}

// capturedResults keeps what the loop's code answers a call with, in place of the model's conversation: the
// program has its own.
type capturedResults struct {
	TranscriptStore
	answers map[string]capturedResult
	mu      sync.Mutex
}

func (c *capturedResults) AppendBlock(_ context.Context, _ string, _ domain.Role, block domain.Block) error {
	if block.Kind == domain.BlockToolResult {
		c.mu.Lock()
		c.answers[block.ID] = capturedResult{text: block.Text, failed: block.IsError}
		c.mu.Unlock()
	}

	return nil
}

func (c *capturedResults) LoadTranscript(context.Context, string) ([]domain.Message, error) {
	return nil, nil
}

func (c *capturedResults) take(id string) capturedResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.answers[id]
}
