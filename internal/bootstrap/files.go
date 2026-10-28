package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/gitstatus"
	codinghttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/http"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	projectdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// newFiles mounts the person's own view of a project's files: the tree, a file, saving an edit and the stream
// that says when they changed.
func newFiles(
	projects *projectapp.Service, tools *agentTools, hub *taskapp.Hub, sess sessions, auth, sameOrigin fiber.Handler,
) httpserver.Mount {
	return codinghttp.NewHandler(ownWorkspaces{projects: projects, tools: tools}, hubChanges{hub: hub}).
		Mount(auth, sameOrigin, sess)
}

// ownWorkspaces opens the files of a project the person owns, with the same confined view the agent has.
type ownWorkspaces struct {
	projects *projectapp.Service
	tools    *agentTools
}

func (w ownWorkspaces) Open(ctx context.Context, userID, projectID string) (codinghttp.Workspace, error) {
	if _, err := w.projects.Get(ctx, userID, projectID); err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return codinghttp.Workspace{}, codingdomain.ErrNotFound
		}

		return codinghttp.Workspace{}, fmt.Errorf("find project: %w", err)
	}

	services, err := w.tools.services(projectID)
	if err != nil {
		return codinghttp.Workspace{}, err
	}

	dir, err := w.tools.folders.Dir(projectID)
	if err != nil {
		return codinghttp.Workspace{}, fmt.Errorf("resolve project folder: %w", err)
	}

	git := codingapp.NewGitStatus(gitstatus.New(dir))
	return codinghttp.Workspace{Reader: services.Reader, Editor: services.Editor, Git: git}, nil
}

// hubChanges carries "the files changed" between the agent, the file screens and every open page.
type hubChanges struct{ hub *taskapp.Hub }

func (h hubChanges) Announce(projectID string) { h.hub.FilesChanged(projectID) }

func (h hubChanges) Follow(projectID string) (<-chan struct{}, func()) {
	sub := h.hub.Subscribe(taskapp.ProjectTopic(projectID))
	signals := make(chan struct{}, 1)

	go func() {
		defer close(signals)

		for range sub.C {
			select {
			case signals <- struct{}{}:
			default:
			}
		}
	}()

	return signals, sub.Close
}
