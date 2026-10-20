package application

import (
	"strings"
	"testing"
)

func TestDelegateBriefUsesMarkdownSections(t *testing.T) {
	args := delegateArgs{Brief: "Fix the login page", Context: "Keep existing routes"}
	got := args.briefText()
	for _, want := range []string{"## Assignment\n\nFix the login page", "## Context from the orchestrator:\n\nKeep existing routes"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in brief: %s", want, got)
		}
	}
}

func TestDelegationInstructionsRequireStructuredBrief(t *testing.T) {
	tool := delegateTool([]AgentRole{{Key: "developer", Name: "Developer"}})
	if !strings.Contains(string(tool.Schema), "Markdown assignment") {
		t.Fatalf("delegate tool should request structured Markdown: %s", tool.Schema)
	}
}
