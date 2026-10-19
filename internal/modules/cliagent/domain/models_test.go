package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

func TestChoice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		rt            domain.Runtime
		model, effort string
		wantModel     string
		wantEffort    string
	}{
		{domain.Claude, "opus", "high", "opus", "high"},
		{domain.Claude, "default", "max", "", "max"},
		{domain.Claude, "opus", "", "opus", ""},
		{domain.Claude, "opus", "turbo", "opus", ""},
		{domain.Claude, "--dangerously-skip-permissions", "high", "", ""},
		{domain.ChatGPT, "default", "xhigh", "", ""},
		{domain.ChatGPT, "default", "medium", "", "medium"},
		{domain.ChatGPT, "opus", "low", "", ""},
	}

	for _, c := range cases {
		model, effort := domain.Choice(c.rt, c.model, c.effort)
		if model != c.wantModel || effort != c.wantEffort {
			t.Errorf("Choice(%s, %q, %q) = %q, %q", c.rt, c.model, c.effort, model, effort)
		}
	}
}
