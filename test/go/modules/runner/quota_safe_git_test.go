package runner_test

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
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
		{"log patch is not quota exempt", []string{"log", "--patch"}, false},
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
			if got := runner.QuotaSafeGit(authorized, task); got != tt.safe {
				t.Errorf("quotaSafeGit(%v) = %t, want %t", tt.args, got, tt.safe)
			}
			if got := runner.ReadOnlyCheck(authorized, task); got != tt.safe {
				t.Errorf("readOnlyCheck(%v) = %t, want %t", tt.args, got, tt.safe)
			}
		})
	}
}
