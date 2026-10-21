package bootstrap

import (
	"context"
	"slices"
	"testing"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

type fixedDatabase struct{ ok bool }

func (f fixedDatabase) Target(context.Context, string) (string, int, bool) {
	return "db.internal", 5432, f.ok
}

func TestProgramReachesTheProjectDatabaseBesidesItsOwnHosts(t *testing.T) {
	own := []string{"api.anthropic.com"}
	req := taskapp.CLIRequest{Program: "claude", ProjectID: "p"}

	granted := &plans{hosts: map[string][]string{"claude": own}, tunnel: fixedDatabase{ok: true}}
	hosts, database := granted.withDatabase(context.Background(), req)

	if database != "db.internal:5432" || !slices.Equal(hosts, []string{"api.anthropic.com", "db.internal:5432"}) {
		t.Errorf("hosts %v database %q", hosts, database)
	}

	if len(own) != 1 {
		t.Error("the program's own host list must not be changed")
	}

	for name, p := range map[string]*plans{
		"no database for this project": {hosts: map[string][]string{"claude": own}, tunnel: fixedDatabase{}},
		"tunnel not wired":             {hosts: map[string][]string{"claude": own}},
	} {
		if hosts, database = p.withDatabase(context.Background(), req); database != "" || !slices.Equal(hosts, own) {
			t.Errorf("%s: hosts %v database %q", name, hosts, database)
		}
	}
}
