package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type editInput struct {
	Target
	Path       string `json:"path" jsonschema:"file to change, relative to the workspace root"`
	OldText    string `json:"old_text" jsonschema:"exact text to replace; must be unique unless replace_all is set"`
	NewText    string `json:"new_text" jsonschema:"the replacement text"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"replace every occurrence, not just a unique one"`
}

type writeInput struct {
	Target
	Path    string `json:"path" jsonschema:"file to create or replace, relative to the workspace root"`
	Content string `json:"content" jsonschema:"the complete new content of the file"`
}

type deleteInput struct {
	Target
	Path string `json:"path" jsonschema:"file to delete, relative to the workspace root"`
}

type moveInput struct {
	Target
	From string `json:"from" jsonschema:"current path of the file"`
	To   string `json:"to" jsonschema:"new path; it must not exist yet"`
}

type changeOutput struct {
	Path   string `json:"path"`
	Result string `json:"result"`
}

func (t *tools) addWriteTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolEditFile), guarded(t.guard, domain.ToolEditFile,
		func(in editInput) string { return "edit " + in.Path }, t.editFile))
	sdk.AddTool(server, toolOf(domain.ToolWriteFile), guarded(t.guard, domain.ToolWriteFile,
		func(in writeInput) string { return "write " + in.Path }, t.writeFile))
	sdk.AddTool(server, toolOf(domain.ToolDeleteFile), guarded(t.guard, domain.ToolDeleteFile,
		func(in deleteInput) string { return "delete " + in.Path }, t.deleteFile))
	sdk.AddTool(server, toolOf(domain.ToolMoveFile), guarded(t.guard, domain.ToolMoveFile,
		func(in moveInput) string { return "move " + in.From + " to " + in.To }, t.moveFile))
}

// changed turns a finished modification into the tool output and the log report.
func changed(project, path string, applied application.Applied, err error) (changeOutput, report, error) {
	if err != nil {
		return changeOutput{}, report{}, fmt.Errorf("change %s: %w", path, err)
	}

	undo, err := applied.Change.Encode()
	if err != nil {
		return changeOutput{}, report{}, fmt.Errorf("record change: %w", err)
	}

	done := report{detail: applied.Detail, undo: undo, project: project}

	return changeOutput{Path: path, Result: applied.Detail}, done, nil
}

func (t *tools) editFile(ctx context.Context, in editInput) (changeOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return changeOutput{}, report{}, err
	}

	applied, err := svc.Editor.Edit(in.Path, in.OldText, in.NewText, in.ReplaceAll)

	return changed(svc.ProjectID, in.Path, applied, err)
}

func (t *tools) writeFile(ctx context.Context, in writeInput) (changeOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return changeOutput{}, report{}, err
	}

	applied, err := svc.Editor.Write(in.Path, in.Content)

	return changed(svc.ProjectID, in.Path, applied, err)
}

func (t *tools) deleteFile(ctx context.Context, in deleteInput) (changeOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return changeOutput{}, report{}, err
	}

	applied, err := svc.Editor.Delete(in.Path)

	return changed(svc.ProjectID, in.Path, applied, err)
}

func (t *tools) moveFile(ctx context.Context, in moveInput) (changeOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return changeOutput{}, report{}, err
	}

	applied, err := svc.Editor.Move(in.From, in.To)

	return changed(svc.ProjectID, in.To, applied, err)
}
