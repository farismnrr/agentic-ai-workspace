package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// setApprovals turns "skip approvals" on or off for one space of the signed-in person.
func (a *API) setApprovals(c fiber.Ctx) error {
	userID, ok := a.signedIn(c)
	if !ok {
		return nil
	}

	var body struct {
		Skip bool `json:"skip"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := a.permissions.SetSkipApprovals(c.Context(), userID, c.Params("slug"), body.Skip); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
