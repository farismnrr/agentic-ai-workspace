package account_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"

	"github.com/gofiber/fiber/v3"
)

func withCookie(cookie string) map[string]string {
	return map[string]string{"Cookie": cookie, "Content-Type": "application/json"}
}

func passwordLogin(t *testing.T, app *fiber.App, password string) int {
	t.Helper()

	body := `{"email":"sari@example.com","password":"` + password + `"}`

	return apptest.Do(t, app, "POST", "/api/auth/login", body, apptest.JSONHeaders()).Status
}

func TestChangingPasswordKeepsOnlyThisDevice(t *testing.T) {
	t.Parallel()

	app := apptest.NewEnv(t).App
	here, _ := apptest.SignedInClient(t, app)
	there := apptest.CookieOf(t, apptest.Do(t, app, "POST", "/api/auth/login",
		`{"email":"sari@example.com","password":"correct horse"}`, apptest.JSONHeaders()))

	wrong := apptest.Do(t, app, "POST", "/api/auth/password",
		`{"current":"nope nope","next":"battery staple"}`, withCookie(here))
	if wrong.Status != 400 || wrong.JSON(t)["error"] != "wrong_password" {
		t.Fatalf("a wrong current password must be refused: %d %s", wrong.Status, wrong.Body)
	}

	weak := apptest.Do(t, app, "POST", "/api/auth/password", `{"current":"correct horse","next":"short"}`,
		withCookie(here))
	if weak.Status != 400 {
		t.Fatalf("a weak new password must be refused: %d", weak.Status)
	}

	good := `{"current":"correct horse","next":"battery staple"}`
	if done := apptest.Do(t, app, "POST", "/api/auth/password", good, withCookie(here)); done.Status != 204 {
		t.Fatalf("change failed: %d %s", done.Status, done.Body)
	}

	if got := apptest.Do(t, app, "GET", "/api/auth/me", "", withCookie(here)).Status; got != 200 {
		t.Fatalf("this device must stay signed in, got %d", got)
	}

	if got := apptest.Do(t, app, "GET", "/api/auth/me", "", withCookie(there)).Status; got != 401 {
		t.Fatalf("other devices must be signed out, got %d", got)
	}

	old, fresh := passwordLogin(t, app, "correct horse"), passwordLogin(t, app, "battery staple")
	if old != 401 || fresh != 200 {
		t.Fatalf("only the new password may work: old %d, new %d", old, fresh)
	}
}
