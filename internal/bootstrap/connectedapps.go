package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	projectdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

// unnamedApp names the project of an app that did not give its name when it connected.
const unnamedApp = "Connected app"

// appWorkspaces gives every connected app a project of its own: ChatGPT works in a project called
// ChatGPT, Gemini in one called Gemini. They are ordinary projects the person can open, with the
// same tools, files and web app as the ones they start themselves.
type appWorkspaces struct {
	projects *projectapp.Service
	tools    *agentTools
}

// For returns the use cases bound to a project: the one named by its Project ID, or, when the call names none,
// the app's own project, made on first use. A person's other projects are open to their connected apps by ID.
func (w appWorkspaces) For(ctx context.Context, userID, client, project string) (codingmcp.Services, error) {
	if project != "" {
		found, err := w.byID(ctx, userID, project)
		if err != nil {
			return codingmcp.Services{}, err
		}

		return w.tools.services(found.ID)
	}

	own, err := w.projects.ForApp(ctx, userID, appName(client))
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("open the app's project: %w", err)
	}

	return w.tools.services(own.ID)
}

// Projects lists the person's projects and marks the app's own.
func (w appWorkspaces) Projects(ctx context.Context, userID, client string) ([]codingmcp.ProjectInfo, error) {
	own, err := w.projects.ForApp(ctx, userID, appName(client))
	if err != nil {
		return nil, fmt.Errorf("open the app's project: %w", err)
	}

	all, err := w.projects.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	infos := make([]codingmcp.ProjectInfo, 0, len(all))
	for _, project := range all {
		infos = append(infos, codingmcp.ProjectInfo{
			ID: project.Slug, Name: project.Name, Description: project.Description, Default: project.ID == own.ID,
		})
	}

	return infos, nil
}

// byID finds one of the person's active projects by its Project ID.
func (w appWorkspaces) byID(ctx context.Context, userID, id string) (projectdomain.Project, error) {
	all, err := w.projects.List(ctx, userID)
	if err != nil {
		return projectdomain.Project{}, fmt.Errorf("list projects: %w", err)
	}

	for _, project := range all {
		if strings.EqualFold(project.Slug, id) {
			return project, nil
		}
	}

	return projectdomain.Project{}, codingdomain.ErrUnknownProject
}

func appName(client string) string {
	if client == "" {
		return unnamedApp
	}

	return client
}

// toolGate is the one switchboard for both the agent and connected apps.
type toolGate struct {
	permissions *spaceapp.Permissions
}

func (g toolGate) Enabled(ctx context.Context, userID string) (map[string]bool, error) {
	states, err := g.permissions.For(ctx, userID, codingSpace())
	if err != nil {
		return nil, fmt.Errorf("load tool switches: %w", err)
	}

	for _, site := range []string{
		codingdomain.ToolStartSite.Name, codingdomain.ToolStopSite.Name, codingdomain.ToolScreenshot.Name,
		codingdomain.ToolWebSearch.Name, codingdomain.ToolFetchPage.Name, codingdomain.ToolSearchImages.Name,
		codingdomain.ToolDownloadAsset.Name, codingdomain.ToolViewImage.Name, codingdomain.ToolDeployExtension.Name,
		codingdomain.ToolRequestEgress.Name, codingdomain.ToolCILogs.Name,
		codingdomain.ToolBeginWork.Name, codingdomain.ToolSetRole.Name, codingdomain.ToolUpdatePlan.Name,
		codingdomain.ToolSaveResearch.Name, codingdomain.ToolSaveRequirements.Name,
		codingdomain.ToolReportProgress.Name, codingdomain.ToolCompleteWork.Name, codingdomain.ToolGetWorkState.Name,
		codingdomain.ToolSubmitPlan.Name,
		codingdomain.ToolFinishResearch.Name, codingdomain.ToolAskQuestion.Name,
	} {
		states[site] = states[codingdomain.SwitchOf(site)]
	}

	return states, nil
}

func (g toolGate) SkipApprovals(ctx context.Context, userID string) (bool, error) {
	skip, err := g.permissions.SkipApprovals(ctx, userID, codingSpace().Slug)
	if err != nil {
		return false, fmt.Errorf("load approval setting: %w", err)
	}

	return skip, nil
}

// appUndo is what the activity log keeps to reverse a change: the project it was made in, and the
// change itself, which only the coding tools understand.
type appUndo struct {
	Project string          `json:"project"`
	Change  json.RawMessage `json:"change"`
}

func wrapUndo(project string, change []byte) []byte {
	if len(change) == 0 || project == "" {
		return nil
	}

	blob, err := json.Marshal(appUndo{Project: project, Change: change})
	if err != nil {
		return nil
	}

	return blob
}

func unwrapUndo(blob []byte) (appUndo, error) {
	var undo appUndo
	if err := json.Unmarshal(blob, &undo); err != nil || undo.Project == "" {
		return appUndo{}, spacedomain.ErrNotUndoable
	}

	return undo, nil
}
