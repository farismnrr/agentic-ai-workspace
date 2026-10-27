package mcp

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestChangedMarksOnlyAppliedWritesAsObserved(t *testing.T) {
	_, unchanged, err := changed("project", "page.ts", application.Applied{Detail: "unchanged"}, nil)
	if err != nil || len(unchanged.files) != 0 {
		t.Fatalf("unchanged report = %+v, err=%v", unchanged, err)
	}

	_, moved, err := changed("project", "new.ts", application.Applied{
		Detail: "moved", Change: domain.Change{Op: domain.OpMove},
	}, nil, "old.ts", "new.ts")
	if err != nil || len(moved.files) != 2 || moved.files[0] != "old.ts" || moved.files[1] != "new.ts" {
		t.Fatalf("move report = %+v, err=%v", moved, err)
	}
}
