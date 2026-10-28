package apptest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

const requestTimeout = 30 * time.Second

// Reply is an HTTP response returned by Do.
type Reply struct {
	Header http.Header
	Body   []byte
	Status int
}

// JSON decodes a response body as a JSON object and fails the test on invalid JSON.
func (r Reply) JSON(t *testing.T) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatalf("not JSON (%d): %s", r.Status, r.Body)
	}

	return out
}

// Do sends a request to the in-process app and returns its response.
func Do(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) Reply {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: requestTimeout, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(resp.Body)
	return Reply{Status: resp.StatusCode, Header: resp.Header, Body: data}
}

// JSONHeaders returns headers for a JSON request body.
func JSONHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}

// Register creates a test account and returns its session cookie.
func Register(t *testing.T, app *fiber.App, name, email, password string) string {
	t.Helper()

	body := `{"name":"` + name + `","email":"` + email + `","password":"` + password + `"}`
	response := Do(t, app, http.MethodPost, "/api/auth/register", body, JSONHeaders())
	if response.Status != http.StatusCreated {
		t.Fatalf("registration failed: %d %s", response.Status, response.Body)
	}
	return CookieOf(t, response)
}

// RPC builds a JSON-RPC request body for an MCP method.
func RPC(method, params string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
}

// MCPHeaders returns the headers used by MCP requests.
func MCPHeaders(token string) map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json", "Accept": "application/json, text/event-stream",
		"MCP-Protocol-Version": "2025-11-25",
	}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	return headers
}

// MustContain fails when a response body omits any requested string.
func MustContain(t *testing.T, body []byte, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(string(body), want) {
			t.Errorf("response should contain %q, got %s", want, body)
		}
	}
}

// CookieOf extracts a session cookie from a response.
func CookieOf(t *testing.T, r Reply) string {
	t.Helper()

	cookie, _, _ := strings.Cut(r.Header.Get("Set-Cookie"), ";")
	if !strings.HasPrefix(cookie, "session=") {
		t.Fatalf("no session cookie in %q", r.Header.Get("Set-Cookie"))
	}
	return cookie
}
