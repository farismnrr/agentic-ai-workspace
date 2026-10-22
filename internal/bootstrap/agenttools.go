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
	envapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	extensionapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/application"
	githubapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	siteapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	webapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// agentTools lets the agent use the coding tools inside a project's own folder.
// Each project gets its own confined view of its files.
type agentTools struct {
	folders *folder.Folders
	engine  *codingrunner.Engine
	env     codingrunner.ProjectEnv
	secrets interface {
		Secret(context.Context, string, string, string) (string, error)
	}
	sites *siteapp.Service
	// extensions packages the browser extensions the AI builds.
	extensions *extensionapp.Service
	web        *webapp.Service
	backups    *codingbackup.Store
	cache      map[string]codingmcp.Services
	handlers   map[string]toolHandler
	github     *githubapp.Service
	plans      *plans
	git        func(projectID string) codingrunner.GitSource
	reach      codingrunner.Reach
	public     string
	mu         sync.Mutex
}

func newAgentTools(
	folders *folder.Folders, dataDir, publicURL string, engine *codingrunner.Engine,
) (*agentTools, error) {
	backups, err := codingbackup.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open backups: %w", err)
	}

	tools := &agentTools{
		folders: folders, public: publicURL, engine: engine, backups: backups, cache: map[string]codingmcp.Services{},
	}
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

	run := codingrunner.New(dir, a.engine)
	if a.env != nil {
		run.WithProjectEnv(projectID, a.env)
	}
	if a.git != nil {
		run.WithGit(a.git(projectID))
	}
	if a.reach != nil {
		run.WithReach(a.reach)
	}

	services := codingmcp.Services{
		Reader: codingapp.NewService(files),
		Editor: codingapp.NewEditor(files, files, a.backups, opaque.Generator{}),
		Tasks:  codingapp.NewTasks(files, run),
		Sites:  projectSites{tools: a, projectID: projectID}, ProjectID: projectID,
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

	defs = append(defs, siteDefinitions()...)
	defs = append(defs, extensionDefinition())
	defs = append(defs, screenshotDefinition(), screenshotSessionDefinition(), viewImageDefinition())

	return append(defs, a.webDefinitions()...)
}

// Risk grades how careful the agent must be: edits are undoable (low), the rest ask by default.
func (a *agentTools) Risk(name string) taskdomain.Risk {
	switch name {
	case codingdomain.ToolEditFile.Name, codingdomain.ToolWriteFile.Name, toolStopSite,
		codingdomain.ToolDownloadAsset.Name:
		return taskdomain.RiskLow
	case codingdomain.ToolDeleteFile.Name, codingdomain.ToolMoveFile.Name,
		codingdomain.ToolRunTask.Name, codingdomain.ToolRunCommand.Name, toolStartSite, toolScreenshotSession:
		return taskdomain.RiskMedium
	}

	return taskdomain.RiskNone
}

var _ interface {
	Secret(context.Context, string, string, string) (string, error)
} = (*envapp.Service)(nil)

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

// forgetting removes a project's folder and then what its commands left outside it (package cache, proxy socket).
type forgetting struct {
	*folder.Folders
	tools *agentTools
}

func (f forgetting) Remove(id string) error {
	f.tools.forget(id)

	return f.Folders.Remove(id) //nolint:wrapcheck // the project service words the failure
}

func (a *agentTools) forget(projectID string) {
	a.mu.Lock()
	delete(a.cache, projectID)
	a.mu.Unlock()

	if dir, err := a.folders.Dir(projectID); err == nil {
		a.engine.Forget(dir)
	}
}
