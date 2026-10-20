package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// ProjectInfo is one of the person's projects, as a connected app sees it.
type ProjectInfo struct {
	ID          string
	Name        string
	Description string
	// Default is the project the app works in when a call names none.
	Default bool
}

type projectEntry struct {
	ID          string `json:"id" jsonschema:"the Project ID: give it as project to any other tool"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     bool   `json:"default" jsonschema:"true for the project you work in when you name none"`
}

type projectsOutput struct {
	Projects []projectEntry `json:"projects"`
}

func (t *tools) addProjectTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolListProjects), guarded(t.guard, domain.ToolListProjects,
		func(struct{}) string { return "list projects" }, t.listProjects))
}

func (t *tools) listProjects(ctx context.Context, _ struct{}) (projectsOutput, report, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	if who.User == "" {
		return projectsOutput{}, report{}, errNoIdentity
	}

	found, err := t.workspaces.Projects(ctx, who.User, who.Client)
	if err != nil {
		return projectsOutput{}, report{}, fmt.Errorf("list projects: %w", err)
	}

	out := projectsOutput{Projects: make([]projectEntry, 0, len(found))}
	for _, project := range found {
		out.Projects = append(out.Projects, projectEntry(project))
	}

	return out, report{}, nil
}
