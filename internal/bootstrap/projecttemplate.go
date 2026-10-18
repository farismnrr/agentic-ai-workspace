package bootstrap

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"

	coding "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// templates are the official Nuxt UI templates baked into the toolbox image (sandbox/Dockerfile). Only these names
// ever reach the copy command.
var templates = map[string]bool{
	"landing": true, "saas": true, "dashboard": true, "docs": true, "chat": true,
	"portfolio": true, "changelog": true, "editor": true, "calendar": true,
}

const (
	templateRoot    = "/usr/local/share/templates/"
	templateSeconds = 60
)

// taken says whether the project already has an interface a template would overwrite. A new repository's README or
// git history does not count, so a template can follow "New repository".
func taken(tools *agentTools, project string) bool {
	dir, err := tools.folders.Dir(project)
	if err != nil {
		return true
	}

	for _, name := range []string{"app", "public", ".template"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}

	return false
}

// newTemplate mounts "start this project from a template": the template's look (its app/ and public/ folders,
// never its server or login) is copied into the project's folder, and its name is written to .template so
// the AI keeps to it instead of picking another.
func newTemplate(tools *agentTools, projects projectOwners, auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Post("/api/projects/:id/template", auth, sameOrigin, func(c fiber.Ctx) error {
			return applyTemplate(c, tools, projects)
		})
	}
}

func applyTemplate(c fiber.Ctx, tools *agentTools, projects projectOwners) error {
	project := c.Params("id")

	if owner, _, err := projects.Owner(c.Context(), project); err != nil || owner != httpserver.UserID(c) {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	var in struct {
		Template string `json:"template"`
	}

	if err := c.Bind().Body(&in); err != nil || !templates[in.Template] {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("bad_template"))
	}

	if taken(tools, project) {
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("not_empty"))
	}

	return copyTemplate(c, tools, project, in.Template)
}

func copyTemplate(c fiber.Ctx, tools *agentTools, project, template string) error {
	services, err := tools.services(project)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("server_error"))
	}

	command := "set -e; src=" + templateRoot + template + "; cp -a \"$src/app\" .; " +
		"if [ -d \"$src/public\" ]; then cp -a \"$src/public\" .; fi; printf '%s\\n' " + template + " > .template"

	result, err := services.Tasks.RunCommand(c.Context(), command, templateSeconds)
	if errors.Is(err, coding.ErrTaskBusy) {
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("busy"))
	}

	if err != nil || result.ExitCode != 0 || result.TimedOut {
		return c.Status(fiber.StatusBadGateway).JSON(httpserver.Failure("template_failed"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}
