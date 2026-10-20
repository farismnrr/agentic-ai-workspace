package mcp

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type taskInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type tasksOutput struct {
	Tasks []taskInfo `json:"tasks"`
}

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
	sdk.AddTool(server, toolOf(domain.ToolRunCommand), guarded(t.guard, domain.ToolRunCommand,
		func(in commandInput) string { return "command: " + in.Command }, t.runCommand))
}

func (t *tools) runCommand(ctx context.Context, in commandInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return runOutput{}, report{}, err
	}

	result, err := svc.Tasks.RunCommand(ctx, in.Command, in.TimeoutSeconds)
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

func (t *tools) listTasks(ctx context.Context, _ Target) (tasksOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return tasksOutput{}, report{}, err
	}

	tasks, err := svc.Tasks.List()
	if err != nil {
		return tasksOutput{}, report{}, fmt.Errorf("list tasks: %w", err)
	}

	out := tasksOutput{Tasks: make([]taskInfo, 0, len(tasks))}
	for _, task := range tasks {
		out.Tasks = append(out.Tasks, taskInfo{Name: task.Name, Description: task.Description})
	}

	return out, report{}, nil
}

func (t *tools) runTask(ctx context.Context, in runInput) (runOutput, report, error) {
	svc, err := t.services(ctx)
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
