package http

import "github.com/gofiber/fiber/v3"

func uploadConcurrency(max int) fiber.Handler {
	semaphore := make(chan struct{}, max)
	return func(c fiber.Ctx) error {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			return c.Next()
		default:
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "uploads_busy"})
		}
	}
}
