package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Tasks lists and runs the named commands the person approved.
type Tasks struct {
	file   TaskFile
	runner Runner
}

// NewTasks wires the task use cases.
func NewTasks(file TaskFile, runner Runner) *Tasks {
	return &Tasks{file: file, runner: runner}
}

// List returns the approved tasks, re-reading the file so edits apply at once.
func (t *Tasks) List() ([]domain.Task, error) {
	data, err := t.file.ReadTasks()
	if err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}

	tasks, err := domain.ParseTasks(data)
	if err != nil {
		return nil, fmt.Errorf("parse tasks: %w", err)
	}

	return tasks, nil
}

// Run executes one approved task by name.
func (t *Tasks) Run(ctx context.Context, name string) (RunResult, error) {
	tasks, err := t.List()
	if err != nil {
		return RunResult{}, err
	}

	for _, task := range tasks {
		if task.Name == name && (task.Kind == "" || task.Kind == "check") {
			// User-defined checks are not trusted to obey a read-only instruction.
			// The Docker workspace itself must be mounted read-only.
			result, err := t.runner.Run(domain.WithReadOnlyWorkspace(ctx), task)
			if err != nil {
				return RunResult{}, fmt.Errorf("run %s: %w", name, err)
			}

			return result, nil
		}
	}

	return RunResult{}, domain.ErrUnknownTask
}

// Verify runs an ordinary QA test/lint/check without a named-task file.
// The role gate permits QA only; the OS mounts source read-only and the runner
// strips project secrets, GitHub credentials and granted egress.
func (t *Tasks) Verify(ctx context.Context, args []string) (RunResult, error) {
	task, err := domain.VerifyTask(args)
	if err != nil {
		return RunResult{}, err
	}
	result, err := t.runner.Run(domain.WithReadOnlyWorkspace(ctx), task)
	if err != nil {
		return RunResult{}, fmt.Errorf("verify: %w", err)
	}
	return result, nil
}

// RunBuild executes only a named owner-configured build, never a QA check.
// Build scripts see the project read-only; generated output is isolated
// under BUILD_OUT=/output and retained in a separate project artifact folder.
func (t *Tasks) RunBuild(ctx context.Context, name string) (RunResult, error) {
	tasks, err := t.List()
	if err != nil {
		return RunResult{}, err
	}
	for _, task := range tasks {
		if task.Name == name && task.Kind == "build" {
			result, err := t.runner.Run(domain.WithReadOnlyWorkspace(ctx), task)
			if err != nil {
				return RunResult{}, fmt.Errorf("run build %s: %w", name, err)
			}
			return result, nil
		}
	}
	return RunResult{}, domain.ErrUnknownTask
}

// RunArtifact executes only an owner-configured document/media generation task.
// Source and secrets are read-only; output is persisted in a separate folder.
func (t *Tasks) RunArtifact(ctx context.Context, name string) (RunResult, error) {
	tasks, err := t.List()
	if err != nil {
		return RunResult{}, err
	}
	for _, task := range tasks {
		if task.Name == name && task.Kind == "artifact" {
			result, err := t.runner.Run(domain.WithReadOnlyWorkspace(ctx), task)
			if err != nil {
				return RunResult{}, fmt.Errorf("run artifact %s: %w", name, err)
			}
			return result, nil
		}
	}
	return RunResult{}, domain.ErrUnknownTask
}

// RunGit supports regular Git subcommands through direct argv for Researcher,
// Developer and DevOps, with no general shell or test/deploy permission.
func (t *Tasks) RunGit(ctx context.Context, args []string) (RunResult, error) {
	task, err := domain.GitTask(args)
	if err != nil {
		return RunResult{}, err
	}
	result, err := t.runner.Run(domain.WithGitOnly(ctx), task)
	if err != nil {
		return RunResult{}, fmt.Errorf("run git: %w", err)
	}
	return result, nil
}

// RunGitHub supports GitHub Issues, PR, CI and release workflows. The
// operation policy is checked using the server-owned role (not model input).
func (t *Tasks) RunGitHub(ctx context.Context, role string, args []string) (RunResult, error) {
	task, err := domain.GitHubTask(strings.TrimSpace(role), args)
	if err != nil {
		return RunResult{}, err
	}
	if role != "developer" {
		ctx = domain.WithReadOnlyWorkspace(ctx)
	}
	result, err := t.runner.Run(domain.WithGitOnly(ctx), task)
	if err != nil {
		return RunResult{}, fmt.Errorf("run GitHub CLI: %w", err)
	}
	return result, nil
}

// Save replaces the approved tasks after checking them.
func (t *Tasks) Save(tasks []domain.Task) error {
	if err := domain.ValidateTasks(tasks); err != nil {
		return fmt.Errorf("check tasks: %w", err)
	}

	data, err := domain.EncodeTasks(tasks)
	if err != nil {
		return fmt.Errorf("encode tasks: %w", err)
	}

	if err := t.file.WriteTasks(data); err != nil {
		return fmt.Errorf("write tasks: %w", err)
	}

	return nil
}
