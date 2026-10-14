package bootstrap

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func do(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) reply {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(resp.Body)

	return reply{Status: resp.StatusCode, Header: resp.Header, Body: data}
}

func jsonHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}

func formHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
}

func cookieOf(t *testing.T, r reply) string {
	t.Helper()

	cookie := strings.SplitN(r.Header.Get("Set-Cookie"), ";", 2)[0]
	if !strings.HasPrefix(cookie, "session=") {
		t.Fatalf("no session cookie in %q", r.Header.Get("Set-Cookie"))
	}

	return cookie
}
