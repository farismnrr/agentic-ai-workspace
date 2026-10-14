package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	codingbackup "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/backup"
	codingfs "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// agentTools lets the agent use the coding tools inside a project's own folder.
// Each project gets its own confined view of its files.
type agentTools struct {
	folders  *folder.Folders
	backups  *codingbackup.Store
	cache    map[string]codingmcp.Services
	handlers map[string]toolHandler
	mu       sync.Mutex
}

func newAgentTools(folders *folder.Folders, dataDir string) (*agentTools, error) {
	backups, err := codingbackup.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open backups: %w", err)
	}

	tools := &agentTools{folders: folders, backups: backups, cache: map[string]codingmcp.Services{}}
	tools.handlers = toolHandlers()

	return tools, nil
}

func (a *agentTools) services(projectID string) (codingmcp.Services, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if cached, ok := a.cache[projectID]; ok {
		return cached, nil
	}

	dir, err := a.folders.Dir(projectID)
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("project folder: %w", err)
	}

	files, err := codingfs.Open(dir)
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("open project folder: %w", err)
	}

	services := codingmcp.Services{
		Reader: codingapp.NewService(files),
		Editor: codingapp.NewEditor(files, files, a.backups, opaque.Generator{}),
		Tasks:  codingapp.NewTasks(files, codingrunner.New(dir)),
	}
	a.cache[projectID] = services

	return services, nil
}

// Definitions describes every project tool to the model.
func (a *agentTools) Definitions() []taskdomain.ToolDef {
	defs := make([]taskdomain.ToolDef, 0, len(toolSchemas))

	for _, info := range codingdomain.Tools() {
		if schema, ok := toolSchemas[info.Name]; ok {
			defs = append(defs, taskdomain.ToolDef{
				Name: info.Name, Description: info.Description, Schema: json.RawMessage(schema),
			})
		}
	}

	return defs
}

// Risk grades how careful the agent must be: edits are undoable (low), the rest ask by default.
func (a *agentTools) Risk(name string) taskdomain.Risk {
	switch name {
	case codingdomain.ToolEditFile.Name, codingdomain.ToolWriteFile.Name:
		return taskdomain.RiskLow
	case codingdomain.ToolDeleteFile.Name, codingdomain.ToolMoveFile.Name, codingdomain.ToolRunTask.Name:
		return taskdomain.RiskMedium
	}

	return taskdomain.RiskNone
}

// Undo reverses a change an earlier step made.
func (a *agentTools) Undo(_ context.Context, projectID string, blob []byte) error {
	services, err := a.services(projectID)
	if err != nil {
		return err
	}

	change, err := codingdomain.ParseChange(blob)
	if err != nil {
		return fmt.Errorf("read change: %w", err)
	}

	if err := services.Editor.Undo(change); err != nil {
		if errors.Is(err, codingdomain.ErrChanged) {
			return fmt.Errorf("%w: %w", taskdomain.ErrUndoRefused, err)
		}

		return fmt.Errorf("undo: %w", err)
	}

	return nil
}

// Describe says what a tool call does, in plain words and as a question.
func (a *agentTools) Describe(call taskapp.Call) taskapp.Described {
	var in toolArgs

	_ = json.Unmarshal(call.Input, &in)

	return describe(call.Name, in)
}
