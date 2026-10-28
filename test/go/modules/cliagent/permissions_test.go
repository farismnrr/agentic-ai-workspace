package cliagent_test

import (
	"testing"

	cliagentcommand "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/command"
)

func TestOnlyChangingToolsLetATeammateChangeTheProject(t *testing.T) {
	t.Parallel()

	if cliagentcommand.MayChange([]string{"read_file", "search_code", "fetch_page"}) {
		t.Fatal("a reader may change the project")
	}

	if !cliagentcommand.MayChange([]string{"read_file", "run_command"}) {
		t.Fatal("running a command can change the project")
	}
}
