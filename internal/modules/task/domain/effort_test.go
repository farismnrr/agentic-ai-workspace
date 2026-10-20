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

func TestNearestGivesTheClosestLevelTheModelTakes(t *testing.T) {
	t.Parallel()

	levels := []string{"low", "medium", "high"}
	cases := map[domain.Effort]domain.Effort{
		"medium": "medium", "max": "high", "xhigh": "high", "minimal": "low", "": "", "weird": "",
	}

	for want, got := range cases {
		if res := domain.Nearest(levels, want); res != got {
			t.Errorf("Nearest(%q) = %q, want %q", want, res, got)
		}
	}

	if domain.Nearest(nil, domain.EffortHigh) != domain.EffortDefault {
		t.Error("a model with no levels takes none")
	}
}
