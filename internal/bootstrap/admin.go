package bootstrap

import (
	"github.com/gofiber/fiber/v3"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	accountdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func accountRole(account *accountapp.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, err := account.Profile(c.Context(), httpserver.UserID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
		}
		httpserver.SetRole(c, user.Role)
		return c.Next()
	}
}

func adminOnly(account *accountapp.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, err := account.Profile(c.Context(), httpserver.UserID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
		}
		httpserver.SetRole(c, user.Role)
		if user.Role != accountdomain.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(httpserver.Failure("forbidden"))
		}
		return c.Next()
	}
}
