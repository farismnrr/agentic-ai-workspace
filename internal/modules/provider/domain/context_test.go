package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

func TestContextWindowPrefersTheChoiceThenTheModelNameThenTheDefault(t *testing.T) {
	t.Parallel()

	chosen := map[string]int64{"big[1m]": 400_000}
	cases := map[string]int64{
		"big[1m]":    400_000,
		"cx/gpt[1M]": domain.LongContextWindow,
		"plain":      domain.DefaultContextWindow,
	}

	for model, want := range cases {
		if got := domain.ContextWindow(model, chosen); got != want {
			t.Fatalf("%s: got %d, want %d", model, got, want)
		}
	}
}

func TestOnlySensibleContextWindowsAreAccepted(t *testing.T) {
	t.Parallel()

	tooSmall, tooBig := domain.ValidContextWindow(100), domain.ValidContextWindow(domain.MaxContextWindow+1)
	if tooSmall || tooBig || !domain.ValidContextWindow(32_000) {
		t.Fatal("context window bounds are wrong")
	}
}
