package account_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func login(t *testing.T, env apptest.Environment, email, password string) int {
	t.Helper()

	body := `{"email":"` + email + `","password":"` + password + `"}`

	return apptest.Do(t, env.App, "POST", "/api/auth/login", body, apptest.JSONHeaders()).Status
}

func TestRepeatedWrongPasswordsAreSlowedDownForThatAccountOnly(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	apptest.SignedInClient(t, env.App)

	for range 8 {
		if got := login(t, env, "sari@example.com", "wrong password"); got != 401 {
			t.Fatalf("a wrong password is a plain 401 until the limit, got %d", got)
		}
	}

	if got := login(t, env, "sari@example.com", "correct horse"); got != 429 {
		t.Fatalf("after too many failures even the right password must wait, got %d", got)
	}

	if got := login(t, env, "other@example.com", "wrong password"); got != 401 {
		t.Fatalf("another account from the same address is not blocked yet, got %d", got)
	}
}

func TestCorrectSignInsAreNeverCounted(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)
	apptest.SignedInClient(t, env.App)

	for range 24 {
		if got := login(t, env, "sari@example.com", "correct horse"); got != 200 {
			t.Fatalf("successful sign-ins must not be limited, got %d", got)
		}
	}
}

func TestSignUpsAndClientRegistrationsAreCapped(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnv(t)

	for i := range 5 {
		body := `{"name":"P","email":"p` + string(rune('a'+i)) + `@example.com","password":"correct horse"}`
		if got := apptest.Do(t, env.App, "POST", "/api/auth/register", body, apptest.JSONHeaders()).Status; got != 201 {
			t.Fatalf("sign-up %d should work, got %d", i, got)
		}
	}

	if got := apptest.Do(t, env.App, "POST", "/api/auth/register",
		`{"name":"P","email":"late@example.com","password":"correct horse"}`, apptest.JSONHeaders()).Status; got != 429 {
		t.Fatalf("too many sign-ups from one address must be refused, got %d", got)
	}

	registration := `{"redirect_uris":["` + apptest.Redirect + `"]}`

	for range 20 {
		apptest.Do(t, env.App, "POST", "/oauth/register", registration, apptest.JSONHeaders())
	}

	if got := apptest.Do(t, env.App, "POST", "/oauth/register", registration, apptest.JSONHeaders()).Status; got != 429 {
		t.Fatalf("too many client registrations must be refused, got %d", got)
	}
}
