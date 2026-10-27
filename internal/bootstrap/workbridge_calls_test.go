package bootstrap

import (
	"testing"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestWorkEventPreservesConversationBubbleKinds(t *testing.T) {
	for _, test := range []struct {
		kind codingmcp.NoteKind
		want taskdomain.EventKind
	}{
		{kind: codingmcp.NoteUser, want: taskdomain.EventRequest},
		{kind: codingmcp.NoteAssistant, want: taskdomain.EventMessage},
	} {
		event := workEvent(codingmcp.WorkCall{Note: codingmcp.Note{Kind: test.kind}})
		if event.Kind != test.want {
			t.Errorf("note kind %q became %q, want %q", test.kind, event.Kind, test.want)
		}
	}
}
