package bootstrap

import (
	"context"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	orchestratordomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"

	orchestrator "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// newOrchestratorHandler runs the completely separate external-only MCP server.
// It reuses the existing task worker runtime through its additive public facade.
func newOrchestratorHandler(s orchestrator.Services) http.Handler {
	server := orchestrator.NewServer(s)
	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, DisableLocalhostProtection: true,
	})
}

type orchestratorFiles struct {
	workspaces ownWorkspaces
	tasks      *taskapp.Service
}

func (f orchestratorFiles) List(ctx context.Context, userID, projectID, path string) (orchestratordomain.FileListing, error) {
	workspace, err := f.workspaces.Open(ctx, userID, projectID)
	if err != nil {
		return orchestratordomain.FileListing{}, err
	}
	if path == "" {
		path = "."
	}
	listing, err := workspace.Reader.List(path)
	if err != nil {
		return orchestratordomain.FileListing{}, err
	}
	out := orchestratordomain.FileListing{Truncated: listing.Truncated}
	for _, item := range listing.Entries {
		out.Entries = append(out.Entries, orchestratordomain.FileEntry{
			Path: item.Path, Size: item.Size, Dir: item.Dir,
		})
	}
	return out, nil
}

func (f orchestratorFiles) SaveText(ctx context.Context, userID, projectID, path, content, baseHash string) (string, error) {
	workspace, err := f.workspaces.Open(ctx, userID, projectID)
	if err != nil {
		return "", err
	}
	if _, err := workspace.Editor.WriteIfUnchanged(path, content, baseHash); err != nil {
		return "", err
	}
	f.tasks.NotifyExternalFileChanged(projectID)
	return path, nil
}
