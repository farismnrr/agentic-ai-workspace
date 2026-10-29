package application

import (
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"strings"
	"testing"
)

func TestBoundedSummaryInputInitialFull(t *testing.T) {
	original := domain.Message{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Original instruction: preserve files " + strings.Repeat("A", 150000)}}}
	latest := domain.Message{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Latest instruction: do not delete files " + strings.Repeat("B", 180000)}}}
	messages := []domain.Message{original, latest}
	input := boundedSummaryInput("system", nil, messages, 16000)
	if len(input) < 2 {
		t.Fatalf("lost latest instruction: %d", len(input))
	}
	if got := estimateTokens("system", nil, withAsk(input)); got >= 8000 {
		t.Fatalf("summary request too large: %d tokens", got)
	}
	if !strings.Contains(input[0].Blocks[0].Text, "Original instruction") || !strings.Contains(input[1].Blocks[0].Text, "Latest instruction") {
		t.Fatal("lost user instructions")
	}
}
