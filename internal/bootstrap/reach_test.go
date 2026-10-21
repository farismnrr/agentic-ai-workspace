package bootstrap

import (
	"slices"
	"testing"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
)

func TestTwoTargetsNeverShareAPort(t *testing.T) {
	if !samePort([]string{"db.internal:5432"}, "localhost:5432") {
		t.Error("the database keeps a port the gateway also wants")
	}

	if samePort([]string{"db.internal:5432"}, "localhost:20128") || samePort(nil, "localhost:20128") {
		t.Error("different ports can both be opened")
	}
}

func TestProgramReachesWhatTheProjectWasGrantedBesidesItsOwnHosts(t *testing.T) {
	own := []string{"api.anthropic.com"}
	granted := codingrunner.Granted{Targets: []string{"db.internal:5432", "localhost:20128"}}

	want := []string{"api.anthropic.com", "db.internal:5432", "localhost:20128"}
	if got := programHosts(own, granted); !slices.Equal(got, want) {
		t.Errorf("hosts: %v", got)
	}

	if len(own) != 1 {
		t.Error("the program's own host list must not be changed")
	}

	if got := programHosts(own, codingrunner.Granted{}); !slices.Equal(got, own) {
		t.Errorf("nothing granted adds nothing: %v", got)
	}
}
