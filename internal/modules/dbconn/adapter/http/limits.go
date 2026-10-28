package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const (
	testRequestsPerMinute   = 5
	sharedRequestsPerMinute = 10
)

// TestRateLimit limits database connection tests per signed-in person.
func TestRateLimit() fiber.Handler {
	return httpserver.Limit(testRequestsPerMinute, time.Minute, httpserver.UserID, false)
}

// SharedRateLimit limits shared database settings changes per signed-in person.
func SharedRateLimit() fiber.Handler {
	return httpserver.Limit(sharedRequestsPerMinute, time.Minute, httpserver.UserID, false)
}
