package bootstrap

import "testing"

func login(t *testing.T, env testEnv, email, password string) int {
	t.Helper()

	body := `{"email":"` + email + `","password":"` + password + `"}`

	return do(t, env.app, "POST", "/api/auth/login", body, jsonHeaders()).Status
}

func TestRepeatedWrongPasswordsAreSlowedDownForThatAccountOnly(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	signedInClient(t, env.app)

	for range loginFailuresPerEmail {
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

	env := newEnv(t)
	signedInClient(t, env.app)

	for range loginFailuresPerEmail * 3 {
		if got := login(t, env, "sari@example.com", "correct horse"); got != 200 {
			t.Fatalf("successful sign-ins must not be limited, got %d", got)
		}
	}
}

func TestSignUpsAndClientRegistrationsAreCapped(t *testing.T) {
	t.Parallel()

	env := newEnv(t)

	for i := range registerPerHour {
		body := `{"name":"P","email":"p` + string(rune('a'+i)) + `@example.com","password":"correct horse"}`
		if got := do(t, env.app, "POST", "/api/auth/register", body, jsonHeaders()).Status; got != 201 {
			t.Fatalf("sign-up %d should work, got %d", i, got)
		}
	}

	if got := do(t, env.app, "POST", "/api/auth/register",
		`{"name":"P","email":"late@example.com","password":"correct horse"}`, jsonHeaders()).Status; got != 429 {
		t.Fatalf("too many sign-ups from one address must be refused, got %d", got)
	}

	registration := `{"redirect_uris":["` + redirect + `"]}`

	for range clientRegisterPerHour {
		do(t, env.app, "POST", "/oauth/register", registration, jsonHeaders())
	}

	if got := do(t, env.app, "POST", "/oauth/register", registration, jsonHeaders()).Status; got != 429 {
		t.Fatalf("too many client registrations must be refused, got %d", got)
	}
}
