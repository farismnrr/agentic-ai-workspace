package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type fixedAuth struct{}

func (fixedAuth) UserID(context.Context, string) (string, error) { return "u1", nil }

func TestSessionRoutesRefuseCrossSiteRequestsExceptNavigations(t *testing.T) {
	app := fiber.New()
	app.Use(httpserver.RequireSession(fixedAuth{}))
	app.All("/api/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	fetch := func(site, mode, dest string) map[string]string {
		return map[string]string{"Sec-Fetch-Site": site, "Sec-Fetch-Mode": mode, "Sec-Fetch-Dest": dest}
	}

	cases := map[string]struct {
		headers map[string]string
		method  string
		want    int
	}{
		"no metadata (older browser or curl)": {nil, "GET", 200},
		"same origin fetch":                   {fetch("same-origin", "cors", "empty"), "GET", 200},
		"sandboxed page fetch":                {fetch("cross-site", "cors", "empty"), "GET", 403},
		"sandboxed page form post":            {fetch("cross-site", "navigate", "document"), "POST", 403},
		"image tag from another site":         {fetch("cross-site", "no-cors", "image"), "GET", 403},
		"link followed from another site":     {fetch("cross-site", "navigate", "document"), "GET", 200},
	}

	for name, c := range cases {
		req := httptest.NewRequestWithContext(context.Background(), c.method, "/api/x", http.NoBody)
		for name, value := range c.headers {
			req.Header.Set(name, value)
		}

		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}

		_ = res.Body.Close()

		if res.StatusCode != c.want {
			t.Errorf("%s: status %d, want %d", name, res.StatusCode, c.want)
		}
	}
}
