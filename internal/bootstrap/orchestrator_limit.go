package bootstrap

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
)

const orchestratorUserLocal = "orchestrator-user-id"

func orchestratorRateLimit(verifier spacehttp.TokenVerifier, resource string) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := bearerToken(c.Get("Authorization"))
		if token != "" {
			access, err := verifier.Verify(c.Context(), token)
			if err == nil && access.Resource == resource && access.UserID != "" {
				c.Locals(orchestratorUserLocal, access.UserID)
			}
		}

		return c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

func orchestratorUserKey(c fiber.Ctx) string {
	if userID, ok := c.Locals(orchestratorUserLocal).(string); ok && userID != "" {
		return "orchestrator-user:" + userID
	}

	return "orchestrator-anonymous:" + c.IP()
}
