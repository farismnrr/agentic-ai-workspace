package mcp

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type gitInput struct {
	Target
	Args []string `json:"args" jsonschema:"Git subcommand and arguments as separate strings (without git prefix)"`
}

// vcsSummary logs only the command family, never message bodies or tokens.
func vcsSummary(program string, args []string) string {
	if len(args) == 0 {
		return program
	}
	if program == "gh" && len(args) > 1 {
		return program + " " + args[0] + " " + args[1]
	}
	return program + " " + args[0]
}

func (t *tools) runGit(ctx context.Context, in gitInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	if err = t.recheckWork(ctx, svc.ProjectID); err != nil {
		return runOutput{}, report{}, err
	}
	gitCtx := ctx
	if t.works != nil {
		who, _ := ctx.Value(identityKey{}).(identity)
		if who.WorkID != "" {
			state, stateErr := t.works.State(ctx, who.User, who.Client, svc.ProjectID, who.WorkID)
			if stateErr == nil {
				if state.Role != "developer" {
					gitCtx = domain.WithReadOnlyWorkspace(gitCtx)
				}
				if state.Role == "devops" && !domain.VCSReadOnly("git", in.Args) {
					gitCtx = domain.WithWritableGitWorkspace(gitCtx)
				}
			}
		}
	}
	result, err := svc.Tasks.RunGit(gitCtx, in.Args)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("run git: %w", err)
	}
	out := runOutput{
		Name: "git", Output: result.Output, ExitCode: result.ExitCode,
		DurationMS: result.Duration.Milliseconds(), TimedOut: result.TimedOut, Truncated: result.Truncated,
	}
	return out, report{detail: fmt.Sprintf("exit code %d", result.ExitCode),
		failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}

func (t *tools) runGh(ctx context.Context, in gitInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	if err = t.recheckWork(ctx, svc.ProjectID); err != nil {
		return runOutput{}, report{}, err
	}
	if t.works == nil {
		return runOutput{}, report{}, fmt.Errorf("tracked role required")
	}
	who, _ := ctx.Value(identityKey{}).(identity)
	state, err := t.works.State(ctx, who.User, who.Client, svc.ProjectID, who.WorkID)
	if err != nil {
		return runOutput{}, report{}, err
	}
	result, err := svc.Tasks.RunGitHub(ctx, state.Role, in.Args)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("run GitHub CLI: %w", err)
	}
	out := runOutput{
		Name: "gh", Output: result.Output, ExitCode: result.ExitCode,
		DurationMS: result.Duration.Milliseconds(), TimedOut: result.TimedOut,
		Truncated: result.Truncated,
	}
	return out, report{detail: fmt.Sprintf("exit code %d", result.ExitCode),
		failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}
