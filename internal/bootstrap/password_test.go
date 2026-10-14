package bootstrap

import (
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

var resetLink = regexp.MustCompile(`/reset\?token=([A-Za-z0-9_-]+)`)

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

func TestForgottenPasswordCanBeResetOnceByEmailLink(t *testing.T) {
	t.Parallel()

	addr, mails := fakeSMTP(t)
	env := newEnvWith(t, func(c *config.Config) { c.SMTPAddr, c.SMTPFrom = addr, "Workspace <no-reply@example.test>" })
	app := env.app
	cookie, _ := signedInClient(t, app)

	asked := do(t, app, "POST", "/api/auth/forgot", `{"email":"Sari@Example.com"}`, jsonHeaders())
	if asked.Status != 202 {
		t.Fatalf("forgot must answer 202, got %d", asked.Status)
	}

	match := resetLink.FindStringSubmatch(waitMail(t, mails))
	if match == nil {
		t.Fatal("the email must contain a reset link")
	}

	body := `{"token":"` + match[1] + `","password":"battery staple"}`
	if got := do(t, app, "POST", "/api/auth/reset", body, jsonHeaders()).Status; got != 204 {
		t.Fatalf("reset failed: %d", got)
	}

	if got := do(t, app, "POST", "/api/auth/reset", body, jsonHeaders()).Status; got != 400 {
		t.Fatalf("a link must work only once, got %d", got)
	}

	if got := do(t, app, "GET", "/api/auth/me", "", withCookie(cookie)).Status; got != 401 {
		t.Fatalf("old sessions must end after a reset, got %d", got)
	}

	if got := passwordLogin(t, app, "battery staple"); got != 200 {
		t.Fatalf("the new password must work, got %d", got)
	}
}

func TestForgotNeverRevealsWhetherAnAccountExists(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	signedInClient(t, app)

	known := do(t, app, "POST", "/api/auth/forgot", `{"email":"sari@example.com"}`, jsonHeaders())
	unknown := do(t, app, "POST", "/api/auth/forgot", `{"email":"nobody@example.com"}`, jsonHeaders())
	junk := do(t, app, "POST", "/api/auth/forgot", `{"email":"not an email"}`, jsonHeaders())

	if known.Status != 202 || unknown.Status != 202 || junk.Status != 202 {
		t.Fatalf("all must look alike: %d %d %d", known.Status, unknown.Status, junk.Status)
	}

	bad := do(t, app, "POST", "/api/auth/reset", `{"token":"made-up","password":"battery staple"}`, jsonHeaders())
	if bad.Status != 400 || bad.JSON(t)["error"] != "invalid_token" {
		t.Fatalf("an unknown link must be refused: %d %s", bad.Status, bad.Body)
	}
}

func TestForgotIsRateLimitedPerAddress(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	body := `{"email":"sari@example.com"}`

	if got := do(t, app, "POST", "/api/auth/forgot", body, jsonHeaders()).Status; got != 202 {
		t.Fatalf("first request: %d", got)
	}

	if got := do(t, app, "POST", "/api/auth/forgot", body, jsonHeaders()).Status; got != 429 {
		t.Fatalf("asking again for the same address at once must be limited, got %d", got)
	}
}
