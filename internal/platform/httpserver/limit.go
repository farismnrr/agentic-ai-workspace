package httpserver

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// Limit allows at most max requests per key in each window and answers the rest
// with 429. The key defaults to the client address. With failuresOnly, only
// requests that end in an error count, which suits sign-in: people who get it
// right are never slowed down, guessers are.
func Limit(max int, window time.Duration, key func(fiber.Ctx) string, failuresOnly bool) fiber.Handler {
	config := limiter.Config{
		Max:                    max,
		Expiration:             window,
		SkipSuccessfulRequests: failuresOnly,
		LimitReached: func(c fiber.Ctx) error {
			return JSONError(c, fiber.StatusTooManyRequests, "too_many_requests")
		},
	}
	if key != nil {
		config.KeyGenerator = key
	}

	return limiter.New(config)
}

// EmailKey keys a JSON request by client address and the email it names, so one
// address cannot hammer one account, while someone else's wrong guesses can
// never lock a person out of their own account.
func EmailKey(c fiber.Ctx) string {
	var body struct {
		Email string `json:"email"`
	}

	_ = json.Unmarshal(c.Body(), &body)

	return "email:" + c.IP() + ":" + strings.ToLower(strings.TrimSpace(body.Email))
}
