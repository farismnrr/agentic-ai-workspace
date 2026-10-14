package bootstrap

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

func withCookie(cookie string) map[string]string {
	return map[string]string{"Cookie": cookie, "Content-Type": "application/json"}
}

func passwordLogin(t *testing.T, app *fiber.App, password string) int {
	t.Helper()

	body := `{"email":"sari@example.com","password":"` + password + `"}`

	return do(t, app, "POST", "/api/auth/login", body, jsonHeaders()).Status
}

func TestChangingPasswordKeepsOnlyThisDevice(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	here, _ := signedInClient(t, app)
	there := cookieOf(t, do(t, app, "POST", "/api/auth/login",
		`{"email":"sari@example.com","password":"correct horse"}`, jsonHeaders()))

	wrong := do(t, app, "POST", "/api/auth/password", `{"current":"nope nope","next":"battery staple"}`, withCookie(here))
	if wrong.Status != 400 || wrong.JSON(t)["error"] != "wrong_password" {
		t.Fatalf("a wrong current password must be refused: %d %s", wrong.Status, wrong.Body)
	}

	weak := do(t, app, "POST", "/api/auth/password", `{"current":"correct horse","next":"short"}`, withCookie(here))
	if weak.Status != 400 {
		t.Fatalf("a weak new password must be refused: %d", weak.Status)
	}

	good := `{"current":"correct horse","next":"battery staple"}`
	if done := do(t, app, "POST", "/api/auth/password", good, withCookie(here)); done.Status != 204 {
		t.Fatalf("change failed: %d %s", done.Status, done.Body)
	}

	if got := do(t, app, "GET", "/api/auth/me", "", withCookie(here)).Status; got != 200 {
		t.Fatalf("this device must stay signed in, got %d", got)
	}

	if got := do(t, app, "GET", "/api/auth/me", "", withCookie(there)).Status; got != 401 {
		t.Fatalf("other devices must be signed out, got %d", got)
	}

	old, fresh := passwordLogin(t, app, "correct horse"), passwordLogin(t, app, "battery staple")
	if old != 401 || fresh != 200 {
		t.Fatalf("only the new password may work: old %d, new %d", old, fresh)
	}
}
