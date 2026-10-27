package bootstrap

import (
	"errors"
	"slices"
	"strings"
	"testing"

	teamdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
	githubdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

var errBoom = errors.New("boom")

func TestOnlyTheRoleThatRunsAppsGetsCILogs(t *testing.T) {
	t.Parallel()

	if !slices.Contains(toolsFor(teamdomain.Role{RunsApps: true}), toolCILogs) {
		t.Error("the role that runs apps (DevOps) should read CI logs")
	}

	if slices.Contains(toolsFor(teamdomain.Role{}), toolCILogs) {
		t.Error("a role that runs nothing should not get ci_logs")
	}
}

func TestCILogsTellTheAgentWhyItFailedWithoutBlamingTheToken(t *testing.T) {
	t.Parallel()

	denied := ciLogsFailure(githubdomain.ErrLogsDenied)
	if !strings.Contains(denied, "do not ask the person to change a token scope") {
		t.Errorf("a refusal must not send the person to change a token scope: %q", denied)
	}

	for err, want := range map[error]string{
		githubdomain.ErrNotConnected: "connect",
		githubdomain.ErrLogsGone:     "90 days",
		githubdomain.ErrBadLogRef:    "owner/name",
		errBoom:                      "could not be reached",
	} {
		if got := ciLogsFailure(err); !strings.Contains(got, want) {
			t.Errorf("ciLogsFailure(%v) = %q, want it to mention %q", err, got, want)
		}
	}
}

func TestCILogsAreClampedAndTitledPerJob(t *testing.T) {
	t.Parallel()

	if clampTail(0) != defaultTail || clampTail(9999) != maxTail || clampTail(50) != 50 {
		t.Error("tail_lines should default to 120 and never pass 400")
	}

	out := formatCILogs([]githubdomain.JobLog{{Name: "Deploy", Conclusion: "failure", Text: "a\nb\nboom"}}, 2)
	if !strings.Contains(out, "== Deploy failure (last 2 lines) ==") || !strings.Contains(out, "b\nboom") ||
		strings.Contains(out, "a\n") {
		t.Errorf("unexpected log text: %q", out)
	}

	if !strings.Contains(formatCILogs(nil, 5), "No job of that run failed") {
		t.Error("a run with no failed job should say so")
	}
}
