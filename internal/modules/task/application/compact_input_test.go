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

func TestBoundedSummaryNeverReplaysOrphanToolCalls(t *testing.T) {
	messages := []domain.Message{
		{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Original request: no deletes"}}},
		{Role: domain.RoleAssistant, Blocks: []domain.Block{{Kind: domain.BlockToolUse, ID: "u1", Name: "run_command", Input: []byte(`{"command":"echo safe"}`)}}},
		{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockToolResult, ID: "r1", Text: strings.Repeat("tool output ", 150000)}}},
		{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Latest instruction: continue safely " + strings.Repeat("猫🙂", 70000)}}},
	}
	input := boundedSummaryInput("system", nil, messages, 16000)
	if got := estimateTokens("system", nil, withAsk(input)); got >= 8000 {
		t.Fatalf("summary exceeded half-window: %d", got)
	}
	for _, message := range input {
		for _, block := range message.Blocks {
			if block.Kind != domain.BlockText || block.ID != "" {
				t.Fatalf("raw executable tool block leaked: %+v", block)
			}
		}
	}
	var text string
	for _, message := range input {
		for _, block := range message.Blocks {
			text += block.Text
		}
	}
	if !strings.Contains(text, "Original request") || !strings.Contains(text, "Latest instruction") {
		t.Fatal("original or latest instructions lost")
	}
}
