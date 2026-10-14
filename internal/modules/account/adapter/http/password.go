package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func (h *Handler) changePassword(c fiber.Ctx) error {
	token := c.Cookies(httpserver.SessionCookieName)

	user, err := h.service.Authenticate(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("unauthenticated"))
	}

	var body struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := h.service.ChangePassword(c.Context(), user.ID, token, body.Current, body.Next); err != nil {
		return h.fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// deleteAccount removes the signed-in person's account after they confirm their password.
func (h *Handler) deleteAccount(c fiber.Ctx) error {
	user, err := h.service.Authenticate(c.Context(), c.Cookies(httpserver.SessionCookieName))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("unauthenticated"))
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := h.service.Delete(c.Context(), user.ID, body.Password); err != nil {
		return h.fail(c, err)
	}

	h.setSession(c, "", time.Unix(0, 0))

	return c.SendStatus(fiber.StatusNoContent)
}
