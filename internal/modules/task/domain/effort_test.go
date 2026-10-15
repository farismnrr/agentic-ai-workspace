package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestOnlyKnownEffortLevelsAreValid(t *testing.T) {
	t.Parallel()

	for _, level := range []domain.Effort{"", "none", "minimal", "low", "medium", "high", "xhigh", "max"} {
		if !level.Valid() {
			t.Fatalf("%q must be a valid effort", level)
		}
	}

	for _, level := range []domain.Effort{"turbo", "HIGH", " low", "extra"} {
		if level.Valid() {
			t.Fatalf("%q must be refused", level)
		}
	}
}
