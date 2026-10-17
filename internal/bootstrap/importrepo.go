package bootstrap

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"

	coding "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	githubapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// repoName is "owner/name" as GitHub writes it. Only these characters ever reach the clone command.
var repoName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}/[A-Za-z0-9._-]{1,100}$`)

const (
	cloneSeconds = 300
	maxCloneNote = 400
)

// newImport mounts "import a repository into this project": the person picks one of the repositories they let
// the GitHub App reach, and it is cloned into the project's empty folder, with its history, so the AI can keep
// working on it and push back.
func newImport(
	github *githubapp.Service, tools *agentTools, projects projectOwners, auth, sameOrigin fiber.Handler,
) httpserver.Mount {
	return func(app *fiber.App) {
		app.Post("/api/projects/:id/import", auth, sameOrigin, func(c fiber.Ctx) error {
			return importRepo(c, github, tools, projects)
		})
	}
}

func importRepo(c fiber.Ctx, github *githubapp.Service, tools *agentTools, projects projectOwners) error {
	user, project := httpserver.UserID(c), c.Params("id")

	if owner, _, err := projects.Owner(c.Context(), project); err != nil || owner != user {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	var in struct {
		Repo string `json:"repo"`
	}

	if err := c.Bind().Body(&in); err != nil || !repoName.MatchString(in.Repo) || strings.Contains(in.Repo, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("bad_repo"))
	}

	if failure := checkImport(c.Context(), github, tools, user, project, in.Repo); failure != nil {
		return c.Status(failure.status).JSON(httpserver.Failure(failure.code))
	}

	return runImport(c, tools, project, in.Repo)
}

type importFailure struct {
	code   string
	status int
}

// checkImport refuses an import that should not start: GitHub not connected, a repository the person did not let
// the app reach, or a project that already has files.
func checkImport(
	ctx context.Context, github *githubapp.Service, tools *agentTools, user, project, repo string,
) *importFailure {
	listed, err := github.Reaches(ctx, user, repo)
	if err != nil {
		return &importFailure{"not_connected", fiber.StatusConflict}
	}

	if !listed {
		return &importFailure{"not_listed", fiber.StatusNotFound}
	}

	dir, err := tools.folders.Dir(project)
	if err != nil {
		return &importFailure{"server_error", fiber.StatusInternalServerError}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return &importFailure{"server_error", fiber.StatusInternalServerError}
	}

	for _, entry := range entries {
		if entry.Name() != ".workspace" {
			return &importFailure{"not_empty", fiber.StatusConflict}
		}
	}

	return nil
}

func runImport(c fiber.Ctx, tools *agentTools, project, repo string) error {
	services, err := tools.services(project)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("server_error"))
	}

	command := "set -e; git clone --depth 1 -- https://github.com/" + repo + ".git .gh-import && " +
		"cp -a .gh-import/. . && rm -rf .gh-import"

	result, err := services.Tasks.RunCommand(c.Context(), command, cloneSeconds)
	if errors.Is(err, coding.ErrTaskBusy) {
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("busy"))
	}

	if err != nil || result.ExitCode != 0 || result.TimedOut {
		note := result.Output
		if len(note) > maxCloneNote {
			note = note[len(note)-maxCloneNote:]
		}

		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "clone_failed", "detail": note})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
