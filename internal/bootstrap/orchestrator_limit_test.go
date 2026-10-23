package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

type rateTokenVerifier map[string]spacehttp.Access

func (v rateTokenVerifier) Verify(_ context.Context, token string) (spacehttp.Access, error) {
	return v[token], nil
}

func TestOrchestratorRateLimitIsPerVerifiedUser(t *testing.T) {
	verifier := rateTokenVerifier{
		"token-a": {UserID: "user-a", Resource: "https://example.test/api/orchestrator"},
		"token-b": {UserID: "user-b", Resource: "https://example.test/api/orchestrator"},
	}
	app := fiber.New()
	app.All("/api/orchestrator",
		orchestratorRateLimit(verifier, "https://example.test/api/orchestrator"),
		httpserver.Limit(1, time.Minute, orchestratorUserKey, false),
		func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) },
	)

	request := func(token string) int {
		req, _ := http.NewRequestWithContext(
			context.Background(), http.MethodPost, "/api/orchestrator", strings.NewReader("{}"),
		)
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				t.Errorf("close response body: %v", err)
			}
		}()
		return response.StatusCode
	}

	if status := request("token-a"); status != http.StatusNoContent {
		t.Fatalf("first user request status = %d", status)
	}
	if status := request("token-a"); status != http.StatusTooManyRequests {
		t.Fatalf("second same-user request status = %d", status)
	}
	if status := request("token-b"); status != http.StatusNoContent {
		t.Fatalf("different user was affected by rate limit: status = %d", status)
	}
}
