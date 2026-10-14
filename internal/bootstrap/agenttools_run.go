package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type toolArgs struct {
	Path          string `json:"path"`
	Pattern       string `json:"pattern"`
	Query         string `json:"query"`
	OldText       string `json:"old_text"`
	NewText       string `json:"new_text"`
	Content       string `json:"content"`
	From          string `json:"from"`
	To            string `json:"to"`
	Name          string `json:"name"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	ReplaceAll    bool   `json:"replace_all"`
	CaseSensitive bool   `json:"case_sensitive"`
}

type toolHandler func(ctx context.Context, services codingmcp.Services, in toolArgs) taskapp.ToolResult

// Execute runs a project tool. Failures become results the model can read and react to.
func (a *agentTools) Execute(ctx context.Context, projectID string, call taskapp.Call) taskapp.ToolResult {
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

	return handler(ctx, services, in)
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
		Output: fmt.Sprintf("Task %q finished: %s.\n%s", in.Name, verdict, result.Output),
		Note:   verdict,
		Run:    &taskdomain.RunInfo{Name: in.Name, ExitCode: result.ExitCode, TimedOut: result.TimedOut},
	}
}
