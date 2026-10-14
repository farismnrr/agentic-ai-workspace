package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type checkPayload struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Command        []string `json:"command"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

func (h *Handler) checks(c fiber.Ctx) error {
	checks, err := h.service.Checks(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	out := make([]checkPayload, 0, len(checks))
	for _, check := range checks {
		out = append(out, checkPayload{
			Name: check.Name, Description: check.Description, Command: check.Command, TimeoutSeconds: check.TimeoutSeconds,
		})
	}

	return c.JSON(out)
}

func (h *Handler) saveChecks(c fiber.Ctx) error {
	var body []checkPayload
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	checks := make([]application.Check, 0, len(body))
	for _, item := range body {
		checks = append(checks, application.Check{
			Name: item.Name, Description: item.Description, Command: item.Command, TimeoutSeconds: item.TimeoutSeconds,
		})
	}

	if err := h.service.SaveChecks(c.Context(), httpserver.UserID(c), c.Params("id"), checks); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
