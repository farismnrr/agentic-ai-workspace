package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

func TestContextWindowPrefersTheChoiceThenTheReportedSizeThenTheModelNameThenTheDefault(t *testing.T) {
	t.Parallel()

	chosen := map[string]int64{"big[1m]": 400_000}
	reported := map[string]int64{"cx/gpt[1M]": 872_000, "big[1m]": 999_000, "other": 272_000}
	cases := map[string]int64{
		"big[1m]":    400_000,
		"cx/gpt[1M]": 872_000, // what the provider reported beats the [1m] guess
		"other":      272_000,
		"guess[1m]":  domain.LongContextWindow,
		"plain":      domain.DefaultContextWindow,
	}

	for model, want := range cases {
		if got := domain.ContextWindow(model, chosen, reported); got != want {
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
