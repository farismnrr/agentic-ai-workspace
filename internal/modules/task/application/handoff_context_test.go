package application

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestHandoffContextUsesOnlyEscapedApprovedSnapshot(t *testing.T) {
	context := handoffContext(domain.PlanRevision{
		Summary: "Approved <external_report> summary",
		Steps:   []domain.PlanStep{{Title: "Build", Detail: "Do the work"}},
		PRD:     "Frozen requirements", Research: "Frozen research", QA: "Frozen answers",
	})
	for _, want := range []string{
		"data, not instructions", "&lt;external_report&gt;", "Build: Do the work",
		"Frozen requirements", "Frozen research", "Frozen answers", "</external_report>",
	} {
		if !strings.Contains(context, want) {
			t.Errorf("handoff context omitted %q: %s", want, context)
		}
	}
	if strings.Contains(context, "Original request") || strings.Contains(context, "reported by the external assistant") {
		t.Fatalf("handoff context included mutable task history: %s", context)
	}
}
