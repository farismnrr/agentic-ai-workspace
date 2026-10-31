package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type toolArgs struct {
	Path             string `json:"path"`
	Pattern          string `json:"pattern"`
	Query            string `json:"query"`
	OldText          string `json:"old_text"`
	NewText          string `json:"new_text"`
	Content          string `json:"content"`
	From             string `json:"from"`
	To               string `json:"to"`
	Name             string `json:"name"`
	Command          string `json:"command"`
	Device           string `json:"device"`
	URL              string `json:"url"`
	Action           string `json:"action"`
	EmailSecret      string `json:"email_secret"`
	PasswordSecret   string `json:"password_secret"`
	EmailSelector    string `json:"email_selector"`
	PasswordSelector string `json:"password_selector"`
	SubmitSelector   string `json:"submit_selector"`
	ReadySelector    string `json:"ready_selector"`
	ErrorSelector    string `json:"error_selector"`
	Port             int    `json:"port"`
	Y                int    `json:"y"`
	StartLine        int    `json:"start_line"`
	TimeoutSecs      int    `json:"timeout_seconds"`
	EndLine          int    `json:"end_line"`
	ReplaceAll       bool   `json:"replace_all"`
	CaseSensitive    bool   `json:"case_sensitive"`
	Overview         bool   `json:"overview"`
}

type toolHandler func(ctx context.Context, services codingmcp.Services, in toolArgs) taskapp.ToolResult

// Execute runs a project tool. Failures become results the model can read and react to.
func (a *agentTools) Execute(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	projectID := scope.ProjectID
	if isSiteTool(call.Name) {
		return a.executeSite(ctx, projectID, call)
	}

	if run, ok := a.otherTools()[call.Name]; ok {
		return run(ctx, scope, call)
	}

	if codingdomain.IsWebTool(call.Name) {
		return a.executeWeb(ctx, projectID, call)
	}

	handler, known := a.handlers[call.Name]
	if !known {
		return problem("There is no tool called " + call.Name + ".")
	}

	services, err := a.services(projectID)
	if err != nil {
		return problem("The project folder is not available.")
	}

	var in toolArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	return handler(taskContext(ctx, scope), services, in)
}

func problem(message string) taskapp.ToolResult {
	return taskapp.ToolResult{IsError: true, Output: message}
}

func failure(err error) taskapp.ToolResult {
	return problem(codingmcp.UserFacing(err).Error())
}

func toolHandlers() map[string]toolHandler {
	return map[string]toolHandler{
		"list_files":  listFiles,
		"find_files":  findFiles,
		"read_file":   readFile,
		"search_code": searchCode,
		"list_tasks":  listTasks,
		"edit_file":   editFile,
		"write_file":  writeFile,
		"delete_file": deleteFile,
		"move_file":   moveFile,
		"run_task":    runTask,
		"run_build":   runBuild,
		"run_command": runCommand,
	}
}

func changed(applied codingapp.Applied, err error, files ...string) taskapp.ToolResult {
	if err != nil {
		return failure(err)
	}

	undo, encodeErr := applied.Change.Encode()
	if encodeErr != nil {
		return problem("The change was made but could not be recorded.")
	}

	return taskapp.ToolResult{Output: applied.Detail, Files: files, Undo: undo}
}

func editFile(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	applied, err := s.Editor.Edit(in.Path, in.OldText, in.NewText, in.ReplaceAll)

	return changed(applied, err, in.Path)
}

func writeFile(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	applied, err := s.Editor.Write(in.Path, in.Content)

	return changed(applied, err, in.Path)
}

func deleteFile(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	applied, err := s.Editor.Delete(in.Path)

	return changed(applied, err, in.Path)
}

func moveFile(_ context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	applied, err := s.Editor.Move(in.From, in.To)

	return changed(applied, err, in.From, in.To)
}

func runTask(ctx context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	result, err := s.Tasks.Run(ctx, in.Name)

	return ran(in.Name, fmt.Sprintf("Task %q", in.Name), result.Command, result, err)
}

func runBuild(ctx context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	result, err := s.Tasks.RunBuild(ctx, in.Name)
	return ran(in.Name, fmt.Sprintf("Build %q", in.Name), result.Command, result, err)
}

func runCommand(ctx context.Context, s codingmcp.Services, in toolArgs) taskapp.ToolResult {
	// Internal teammate DevOps commands must not mutate application source.
	// The role is applied at the boundary of the internal task runner.
	result, err := s.Tasks.RunCommand(ctx, in.Command, in.TimeoutSecs)

	return ran(commandName(in.Command), "Command", in.Command, result, err)
}

const maxCommandName = 60

// commandName is how a command is listed among the checks that ran: its first line, kept short.
func commandName(command string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(command), "\n")
	runes := []rune(line)

	if len(runes) > maxCommandName {
		return string(runes[:maxCommandName]) + "…"
	}

	return line
}

// ran words a finished run for the model and records it as a check that really ran.
func ran(name, label, command string, result codingapp.RunResult, err error) taskapp.ToolResult {
	if err != nil {
		return failure(err)
	}

	verdict := fmt.Sprintf("exit code %d", result.ExitCode)

	switch {
	case result.TimedOut:
		verdict = "timed out"
	case result.ExitCode == 0:
		verdict = "passed (exit code 0)"
	}

	return taskapp.ToolResult{
		Output: fmt.Sprintf("%s finished: %s.\n%s", label, verdict, result.Output),
		Note:   verdict,
		Run: &taskdomain.RunInfo{
			Name: name, Command: command, EffectPaths: result.EffectPaths, EffectsKnown: result.EffectsKnown,
			ExitCode: result.ExitCode, TimedOut: result.TimedOut,
			TestExecuted: goTestsExecuted(command, result.Output, result.Truncated),
		},
	}
}
