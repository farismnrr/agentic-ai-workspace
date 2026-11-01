package mcp

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)


type runInput struct {
	Target
	Name string `json:"name" jsonschema:"task name from list_tasks"`
}

type runOutput struct {
	Name       string `json:"name"`
	Output     string `json:"output" jsonschema:"combined stdout and stderr, cut to the last 64 KB"`
	ExitCode   int    `json:"exit_code" jsonschema:"0 means success; -1 means it timed out or could not start"`
	DurationMS int64  `json:"duration_ms"`
	TimedOut   bool   `json:"timed_out"`
	Truncated  bool   `json:"truncated" jsonschema:"true when earlier output was dropped"`
}


type commandInput struct {
	Target
	Command        string `json:"command" jsonschema:"shell command line, run with bash in the sandbox"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"stop after this many seconds (max 600)"`
}

type whoamiOutput struct {
	ID string `json:"id" jsonschema:"stable identifier of the signed-in account"`
}

func (t *tools) addExecTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolRunTask), guarded(t.guard, domain.ToolRunTask,
		func(in runInput) string { return "run " + in.Name }, t.runTask))
	sdk.AddTool(server, toolOf(domain.ToolVerify), guarded(t.guard, domain.ToolVerify,
		func(in gitInput) string { return vcsSummary("verify", in.Args) }, t.verify))
	sdk.AddTool(server, toolOf(domain.ToolRunBuild), guarded(t.guard, domain.ToolRunBuild,
		func(in runInput) string { return "build " + in.Name }, t.runBuild))
	sdk.AddTool(server, toolOf(domain.ToolRunArtifact), guarded(t.guard, domain.ToolRunArtifact,
		func(in runInput) string { return "artifact " + in.Name }, t.runArtifact))
	sdk.AddTool(server, toolOf(domain.ToolGit), guarded(t.guard, domain.ToolGit,
		func(in gitInput) string { return vcsSummary("git", in.Args) }, t.runGit))
	sdk.AddTool(server, toolOf(domain.ToolGh), guarded(t.guard, domain.ToolGh,
		func(in gitInput) string { return vcsSummary("gh", in.Args) }, t.runGh))
	sdk.AddTool(server, toolOf(domain.ToolRunCommand), guarded(t.guard, domain.ToolRunCommand,
		func(in commandInput) string { return "command: " + in.Command }, t.runCommand))
}


func (t *tools) runCommand(ctx context.Context, in commandInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	err = t.recheckWork(ctx, svc.ProjectID)
	if err != nil {
		return runOutput{}, report{}, err
	}

	// Connected role commands never get a writable project mount. This
	// includes DevOps and any user-configured role allowed to use shell.
	result, err := svc.Tasks.RunCommand(domain.WithReadOnlyWorkspace(ctx), in.Command, in.TimeoutSeconds)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("run command: %w", err)
	}

	return commandReport(result)
}

func commandReport(result application.RunResult) (runOutput, report, error) {
	out := runOutput{
		Name: "command", Output: result.Output, ExitCode: result.ExitCode, DurationMS: result.Duration.Milliseconds(),
		TimedOut: result.TimedOut, Truncated: result.Truncated,
	}
	detail := fmt.Sprintf("exit code %d in %s", result.ExitCode, result.Duration.Round(time.Millisecond))

	if result.TimedOut {
		detail = "timed out"
	}

	return out, report{detail: detail, failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}

func runNote(name string, result application.RunResult) *RunNote {
	return &RunNote{Name: name, ExitCode: result.ExitCode, TimedOut: result.TimedOut}
}


func (t *tools) runArtifact(ctx context.Context, in runInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	if err = t.recheckWork(ctx, svc.ProjectID); err != nil {
		return runOutput{}, report{}, err
	}
	result, err := svc.Tasks.RunArtifact(ctx, in.Name)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("run artifact: %w", err)
	}
	out := runOutput{
		Name: in.Name, Output: result.Output, ExitCode: result.ExitCode, DurationMS: result.Duration.Milliseconds(),
		TimedOut: result.TimedOut, Truncated: result.Truncated,
	}
	return out, report{detail: fmt.Sprintf("exit code %d", result.ExitCode),
		failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}

func (t *tools) runBuild(ctx context.Context, in runInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	if err = t.recheckWork(ctx, svc.ProjectID); err != nil {
		return runOutput{}, report{}, err
	}
	result, err := svc.Tasks.RunBuild(ctx, in.Name)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("run build: %w", err)
	}
	out := runOutput{
		Name: in.Name, Output: result.Output, ExitCode: result.ExitCode, DurationMS: result.Duration.Milliseconds(),
		TimedOut: result.TimedOut, Truncated: result.Truncated,
	}
	return out, report{detail: fmt.Sprintf("exit code %d", result.ExitCode),
		failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}

func (t *tools) verify(ctx context.Context, in gitInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	if err = t.recheckWork(ctx, svc.ProjectID); err != nil {
		return runOutput{}, report{}, err
	}
	result, err := svc.Tasks.Verify(ctx, in.Args)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("QA verification: %w", err)
	}
	out := runOutput{
		Name: "verify", Output: result.Output, ExitCode: result.ExitCode,
		DurationMS: result.Duration.Milliseconds(), TimedOut: result.TimedOut,
		Truncated: result.Truncated,
	}
	return out, report{detail: fmt.Sprintf("exit code %d", result.ExitCode),
		failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}

func (t *tools) runTask(ctx context.Context, in runInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}
	err = t.recheckWork(ctx, svc.ProjectID)
	if err != nil {
		return runOutput{}, report{}, err
	}

	result, err := svc.Tasks.Run(ctx, in.Name)
	if err != nil {
		return runOutput{}, report{}, fmt.Errorf("run task: %w", err)
	}

	out := runOutput{
		Name: in.Name, Output: result.Output, ExitCode: result.ExitCode, DurationMS: result.Duration.Milliseconds(),
		TimedOut: result.TimedOut, Truncated: result.Truncated,
	}
	detail := fmt.Sprintf("exit code %d in %s", result.ExitCode, result.Duration.Round(time.Millisecond))

	if result.TimedOut {
		detail = "timed out"
	}

	return out, report{detail: detail, failed: result.ExitCode != 0, run: runNote(out.Name, result)}, nil
}
