package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestProviderIDDecodesAPlansColon(t *testing.T) {
	t.Parallel()

	app := fiber.New()

	var got string

	app.Get("/api/providers/:id", func(c fiber.Ctx) error {
		got = providerID(c)

		return c.SendStatus(fiber.StatusNoContent)
	})

	request := httptest.NewRequestWithContext(t.Context(), "GET", "/api/providers/plan%3Achatgpt", nil)
	if _, err := app.Test(request); err != nil {
		t.Fatal(err)
	}

	if got != "plan:chatgpt" {
		t.Fatalf("id = %q", got)
	}
}
