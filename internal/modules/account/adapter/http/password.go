package http

import (
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

// forgot always answers 202, so it never reveals which addresses have an account.
func (h *Handler) forgot(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := h.recovery.Request(c.Context(), body.Email); err != nil {
		return h.fail(c, err)
	}

	return c.SendStatus(fiber.StatusAccepted)
}

func (h *Handler) reset(c fiber.Ctx) error {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := h.recovery.Reset(c.Context(), body.Token, body.Password); err != nil {
		return h.fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
