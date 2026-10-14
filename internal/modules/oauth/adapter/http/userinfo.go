package http

import "github.com/gofiber/fiber/v3"

// userinfo returns the OIDC claims the bearer token's scopes allow.
func (h *Handler) userinfo(c fiber.Ctx) error {
	noStore(c)

	token, err := bearer(c)
	if err != nil {
		c.Set(fiber.HeaderWWWAuthenticate, `Bearer`)

		return c.SendStatus(fiber.StatusUnauthorized)
	}

	claims, err := h.service.UserInfo(c.Context(), token)
	if err != nil {
		c.Set(fiber.HeaderWWWAuthenticate, `Bearer error="invalid_token"`)

		return c.SendStatus(fiber.StatusUnauthorized)
	}

	return c.JSON(claims)
}
