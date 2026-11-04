package runner

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestQuotaSafeGitOnlyPermitsServerAuthorizedReadOnlyWorkspaceOperations(t *testing.T) {
	authorized := domain.WithGitOnly(context.Background())
	tests := []struct {
		name string
		args []string
		safe bool
	}{
		{"status", []string{"status", "--short", "--branch"}, true},
		{"diff check", []string{"diff", "--check"}, true},
		{"diff stat", []string{"diff", "--stat"}, true},
		{"branch verbose", []string{"branch", "-vv"}, true},
		{"branch current", []string{"branch", "--show-current"}, true},
		{"log last commit", []string{"log", "-1", "--oneline"}, true},
		{"ls-files", []string{"ls-files"}, true},
		{"show-ref", []string{"show-ref"}, true},
		{"rev-parse", []string{"rev-parse", "HEAD"}, true},
		{"ls-remote", []string{"ls-remote", "origin", "HEAD"}, true},
		{"push", []string{"push", "origin", "dev"}, true},
		{"push dry-run", []string{"push", "--dry-run", "origin", "dev"}, true},
		{"remote list", []string{"remote"}, true},
		{"remote -v", []string{"remote", "-v"}, true},
		{"remote get-url", []string{"remote", "get-url", "origin"}, true},
		{"remote set-url", []string{"remote", "set-url", "origin", "https://example.org/repo"}, false},
		{"branch creates ref", []string{"branch", "feature"}, false},
		{"branch renames ref", []string{"branch", "--move", "dev", "feature"}, false},
		{"log patch is not quota exempt", []string{"log", "-p"}, false},
		{"fetch writes Git metadata", []string{"fetch", "origin"}, false},
		{"git add mutates index", []string{"add", "README.md"}, false},
		{"git commit writes objects", []string{"commit", "-m", "test"}, false},
		{"git checkout", []string{"checkout", "dev"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := domain.GitTask(tt.args)
			if err != nil {
				t.Fatalf("build bounded Git task: %v", err)
			}
			if got := quotaSafeGit(authorized, task); got != tt.safe {
				t.Errorf("quotaSafeGit(%v) = %t, want %t", tt.args, got, tt.safe)
			}
			if got := readOnlyCheck(authorized, task); got != tt.safe {
				t.Errorf("readOnlyCheck(%v) = %t, want %t", tt.args, got, tt.safe)
			}
			if quotaSafeGit(context.Background(), task) {
				t.Fatal("ordinary shell context must never get Git-only quota exemption")
			}
			if quotaSafeGit(domain.WithWritableGitWorkspace(authorized), task) {
				t.Fatal("writable Git context must never get immutable quota exemption")
			}
		})
	}
}

func TestQuotaSafeGitRejectsSpoofedCommandAndGhMutations(t *testing.T) {
	ctx := domain.WithGitOnly(context.Background())
	for _, task := range []domain.Task{
		{Kind: "git", Command: []string{"bash", "-c", "git push origin dev"}},
		{Kind: "git", Command: []string{"git", "-c", "core.hooksPath=/tmp", "--no-pager", "push", "origin", "dev"}},
		{Kind: "git", Command: []string{"git", "push", "origin", "dev"}},
		{Name: "gh", Command: []string{"gh", "run", "rerun", "123"}},
		{Name: "gh", Command: []string{"gh", "api", "-X", "POST", "repos/owner/repo"}},
	} {
		if quotaSafeGit(ctx, task) {
			t.Fatalf("unsafe/spoofed task was quota-exempt: %+v", task)
		}
	}
	status, err := domain.GitHubTask("devops", []string{"auth", "status"})
	if err != nil {
		t.Fatalf("build gh auth status task: %v", err)
	}
	if !quotaSafeGit(ctx, status) || !readOnlyCheck(ctx, status) {
		t.Fatal("server-authorized gh auth status should be available over quota")
	}
}

func TestOversizedWorkspaceImmutableChecksAndIsolatedOutputs(t *testing.T) {
	readOnly := domain.WithReadOnlyWorkspace(context.Background())
	check := domain.Task{Kind: "check", Name: "verify", Command: []string{"go", "test", "./..."}}
	if !readOnlyCheck(readOnly, check) || isolatedOutput(readOnly, check) {
		t.Fatal("QA check must be immutable, not treated as writable build output")
	}
	for _, kind := range []string{"build", "artifact"} {
		job := domain.Task{Kind: kind, Name: "isolated-output", Command: []string{"bash", "-c", "true"}}
		if readOnlyCheck(readOnly, job) || !isolatedOutput(readOnly, job) {
			t.Fatalf("%s must use a separate bounded writable output, not skip its output quota", kind)
		}
		if isolatedOutput(context.Background(), job) {
			t.Fatalf("%s without server read-only context must never bypass source quota", kind)
		}
	}
	command := domain.Task{Name: "command", Command: []string{"bash", "-c", "true"}}
	if readOnlyCheck(context.Background(), command) || isolatedOutput(context.Background(), command) {
		t.Fatal("arbitrary shell must remain subject to the full workspace quota")
	}
}
