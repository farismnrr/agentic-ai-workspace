package connectors_test

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	cliagentcommand "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/command"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
)

type vcsPermissionSpy struct {
	readonly bool
	gitOnly bool
}

func (s *vcsPermissionSpy) Run(ctx context.Context, _ domain.Task) (codingapp.RunResult, error) {
	s.readonly = domain.ReadOnlyWorkspace(ctx)
	s.gitOnly = domain.GitOnly(ctx)
	return codingapp.RunResult{ExitCode: 0}, nil
}

func TestGitHubCliPreservesSourceIsolationByRole(t *testing.T) {
	for _, tc := range []struct {
		role string
		args []string
		readonly bool
	}{
		{"developer", []string{"issue", "create", "--title", "x"}, false},
		{"qa", []string{"pr", "review", "42", "--comment"}, true},
		{"devops", []string{"workflow", "run", "ci.yml"}, true},
	} {
		spy := &vcsPermissionSpy{}
		tasks := codingapp.NewTasks(nil, spy)
		if _, err := tasks.RunGitHub(context.Background(), tc.role, tc.args); err != nil {
			t.Errorf("%s gh %q failed: %v", tc.role, tc.args, err)
		}
		if spy.readonly != tc.readonly || !spy.gitOnly {
			t.Errorf("%s GH sandbox isolation: readonly=%v gitOnly=%v", tc.role, spy.readonly, spy.gitOnly)
		}
	}
}

func TestInternalVendorCannotTurnNamedChecksIntoRawBash(t *testing.T) {
	for _, name := range []string{"run_task", "run_build", "run_artifact", "git", "gh"} {
		for _, vendorTool := range cliagentcommand.ClaudeTools([]string{name}) {
			if vendorTool == "Bash" {
				t.Fatalf("%s granted unrestricted vendor Bash instead of server mediation", name)
			}
		}
	}
}

func TestDeveloperGitSupportsDailyWorkflowWithoutShell(t *testing.T) {
	supported := [][]string{
		{"status"}, {"diff", "--stat"}, {"add", "src/main.go"},
		{"branch", "feature/new-ui"}, {"switch", "dev"},
		{"cherry-pick", "abc123"}, {"cherry-pick", "--continue"},
		{"rebase", "--continue"}, {"rebase", "--abort"},
		{"merge", "feature/new-ui"}, {"stash", "push", "-m", "checkpoint"},
		{"stash", "pop"}, {"commit", "-m", "Improve the editor"},
		{"push", "origin", "HEAD:dev"}, {"worktree", "list"},
		{"sparse-checkout", "list"}, {"log", "--oneline"},
		{"show-ref"}, {"rev-parse", "HEAD"}, {"update-ref", "refs/heads/dev", "abc123"},
	}
	for _, args := range supported {
		task, err := domain.GitTask(args)
		if err != nil {
			t.Errorf("normal Git workflow %q was refused: %v", args, err)
			continue
		}
		if task.Command[0] != "git" {
			t.Fatalf("must execute Git directly, got %q", task.Command)
		}
	}
	for _, args := range [][]string{
		{}, {"unknown-external-helper"}, {"filter-branch"}, {"config", "core.hooksPath", "/tmp/x"},
		{"clean", "-fdx"}, {"reset", "--hard"}, {"push", "--force", "origin", "dev"},
		{"push", "origin", "+HEAD:dev"}, {"push", "origin", ":dev"},
		{"stash", "clear"}, {"branch", "-D", "dev"},
		{"submodule", "foreach", "sh"}, {"bisect", "run", "bash"},
		{"log", "--ext-diff"}, {"-c", "alias.foo=!sh"},
		{"commit", "--no-verify", "-m", "unsafe"},
	} {
		if _, err := domain.GitTask(args); err == nil {
			t.Errorf("unsafe Git operation accepted: %q", args)
		}
	}
}

func TestVCSReadCommandsSkipRoutineApproval(t *testing.T) {
	for _, test := range []struct {
		program string
		args []string
		readonly bool
	}{
		{"git", []string{"status", "--short"}, true},
		{"git", []string{"diff", "--stat"}, true},
		{"git", []string{"commit", "-m", "fix"}, false},
		{"git", []string{"branch", "feature"}, false},
		{"gh", []string{"issue", "view", "123"}, true},
		{"gh", []string{"run", "view", "444"}, true},
		{"gh", []string{"workflow", "run", "ci.yml"}, false},
		{"gh", []string{"issue", "close", "123"}, false},
	} {
		if got := domain.VCSReadOnly(test.program, test.args); got != test.readonly {
			t.Errorf("%s %q expected read-only=%v, got %v", test.program, test.args, test.readonly, got)
		}
	}
}

func TestGitHubCLIPermissionsByOperation(t *testing.T) {
	allowed := []struct{
		role string
		args []string
	}{
		{"developer", []string{"issue", "create", "--title", "Bug", "--body", "repro"}},
		{"developer", []string{"issue", "comment", "123", "--body", "fixed"}},
		{"developer", []string{"pr", "create", "--fill"}},
		{"developer", []string{"pr", "checks", "42"}},
		{"developer", []string{"run", "view", "1234"}},
		{"developer", []string{"run", "download", "1234"}},
		{"developer", []string{"workflow", "list"}},
		{"developer", []string{"release", "view", "v2"}},
		{"developer", []string{"repo", "view"}},
		{"developer", []string{"status"}},
		{"developer", []string{"api", "repos/owner/repo/actions/runs"}},
		{"qa", []string{"pr", "review", "123", "--comment", "--body", "needs fix"}},
		{"qa", []string{"run", "view", "1234"}},
		{"devops", []string{"run", "rerun", "1234", "--failed"}},
		{"devops", []string{"workflow", "run", "ci.yml"}},
		{"devops", []string{"release", "create", "v2", "release.tar.gz"}},
		{"researcher", []string{"issue", "view", "12"}},
		{"planner", []string{"pr", "view", "42"}},
		{"security", []string{"run", "view", "1234"}},
	}
	for _, test := range allowed {
		task, err := domain.GitHubTask(test.role, test.args)
		if err != nil || len(task.Command) < 2 || task.Command[0] != "gh" {
			t.Errorf("%s should be able to run gh %q: %v", test.role, test.args, err)
		}
	}
	blocked := []struct{
		role string
		args []string
	}{
		{"researcher", []string{"issue", "close", "12"}},
		{"planner", []string{"pr", "create", "--fill"}},
		{"security", []string{"workflow", "run", "ci.yml"}},
		{"developer", []string{"run", "rerun", "1234"}},
		{"developer", []string{"workflow", "run", "deploy.yml"}},
		{"developer", []string{"release", "create", "v2"}},
		{"qa", []string{"issue", "close", "123"}},
		{"devops", []string{"issue", "edit", "123"}},
		{"devops", []string{"pr", "merge", "123"}},
		{"developer", []string{"repo", "delete", "owner/repo"}},
		{"devops", []string{"workflow", "disable", "deploy.yml"}},
		{"developer", []string{"api", "-X", "DELETE", "repos/owner/repo"}},
		{"developer", []string{"api", "graphql", "-f", "query=mutation {}"}},
		{"developer", []string{"api", "repos/owner/repo", "-f", "name=x"}},
		{"developer", []string{"extension", "install", "owner/gh-evil"}},
		{"developer", []string{"alias", "set", "x", "echo shell"}},
		{"developer", []string{"auth", "token"}},
	}
	for _, test := range blocked {
		if _, err := domain.GitHubTask(test.role, test.args); err == nil {
			t.Errorf("%s unexpectedly allowed gh %q", test.role, test.args)
		}
	}
}
