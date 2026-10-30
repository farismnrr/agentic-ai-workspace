package bootstrap

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	molabapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/molab/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// MoLab credentials exist only inside this process. These authenticated routes
// never return a token, and CSRF/SameOrigin applies to mutations.
func newMoLabMount(sessions *molabapp.Sessions, tasks *taskapp.Service,
	auth, fence fiber.Handler, enabled bool) httpserver.Mount {
	allowed := func(c fiber.Ctx) (taskdomain.Task, bool) {
		task, err := tasks.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
		if err != nil {
			_ = c.Status(404).JSON(httpserver.Failure("not_found"))
			return taskdomain.Task{}, false
		}
		if task.Source != "" || task.WorkID != "" || task.ArchivedAt != nil || task.Status == taskdomain.StatusDone || task.Status == taskdomain.StatusFailed {
			sessions.Forget(task.ID)
			_ = c.Status(409).JSON(httpserver.Failure("task_closed"))
			return taskdomain.Task{}, false
		}
		return task, true
	}
	return func(app *fiber.App) {
		app.Get("/api/tasks/:id/molab", auth, func(c fiber.Ctx) error {
			c.Set("Cache-Control", "no-store")
			task, ok := allowed(c)
			if !ok {
				return nil
			}
			state := sessions.Status(task.ID, task.UserID, task.ProjectID)
			return c.JSON(molabPayload(state, enabled))
		})
		app.Post("/api/tasks/:id/molab", auth, fence, func(c fiber.Ctx) error {
			c.Set("Cache-Control", "no-store")
			task, ok := allowed(c)
			if !ok {
				return nil
			}
			if !enabled {
				return c.Status(503).JSON(httpserver.Failure("molab_disabled"))
			}
			var body struct {
				URL   string `json:"url"`
				Token string `json:"token"`
			}
			if c.Bind().JSON(&body) != nil {
				return c.Status(400).JSON(httpserver.Failure("invalid_body"))
			}
			state, err := sessions.Put(task.ID, task.UserID, task.ProjectID, body.URL, body.Token)
			if errors.Is(err, molabapp.ErrInvalid) {
				return c.Status(400).JSON(httpserver.Failure("invalid_molab_session"))
			}
			if err != nil {
				return c.Status(409).JSON(httpserver.Failure("pairing_unavailable"))
			}
			return c.JSON(molabPayload(state, true))
		})
		app.Delete("/api/tasks/:id/molab", auth, fence, func(c fiber.Ctx) error {
			c.Set("Cache-Control", "no-store")
			task, err := tasks.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
			if err != nil {
				return c.Status(404).JSON(httpserver.Failure("not_found"))
			}
			sessions.Forget(task.ID)
			return c.SendStatus(204)
		})
	}
}

// molabPayload only includes connection metadata, never the bearer token.
func molabPayload(state molabapp.Status, available bool) any {
	return struct {
		molabapp.Status
		Available bool `json:"available"`
	}{Status: state, Available: available}
}
