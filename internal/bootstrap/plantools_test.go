package bootstrap

import (
	"slices"
	"testing"
)

func TestClaudeToolsAreOnlyWhatTheRoleMayUse(t *testing.T) {
	t.Parallel()

	got := claudeTools([]string{"read_file", "search_code", "edit_file", "start_site"})
	if want := []string{"Edit", "Grep", "MultiEdit", "Read"}; !slices.Equal(got, want) {
		t.Fatalf("tools = %v, want %v", got, want)
	}

	if len(claudeTools(nil)) != 0 {
		t.Fatal("a role with no tools got some")
	}
}

func TestOnlyChangingToolsLetATeammateChangeTheProject(t *testing.T) {
	t.Parallel()

	if mayChange([]string{"read_file", "search_code", "fetch_page"}) {
		t.Fatal("a reader may change the project")
	}

	if !mayChange([]string{"read_file", "run_command"}) {
		t.Fatal("running a command can change the project")
	}
}
