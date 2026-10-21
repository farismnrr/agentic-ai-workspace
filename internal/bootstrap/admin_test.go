package bootstrap

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func TestSharedDatabaseSettingsAreAdminOnly(t *testing.T) {
	env := newEnvWith(t, func(cfg *config.Config) { cfg.AdminEmail = "admin@example.com" })
	admin := cookieOf(t, do(t, env.app, "POST", "/api/auth/register",
		`{"name":"Admin","email":"admin@example.com","password":"long-enough-password"}`, jsonHeaders()))
	user := cookieOf(t, do(t, env.app, "POST", "/api/auth/register",
		`{"name":"User","email":"user@example.com","password":"long-enough-password"}`, jsonHeaders()))

	if role := do(t, env.app, "GET", "/api/auth/me", "", jsonCookie(user)).JSON(t)["role"]; role != "user" {
		t.Fatalf("new account role = %v, want user", role)
	}
	if role := do(t, env.app, "GET", "/api/auth/me", "", jsonCookie(admin)).JSON(t)["role"]; role != "admin" {
		t.Fatalf("configured account role = %v, want admin", role)
	}
	if got := do(t, env.app, "PUT", "/api/settings/database", sharedSettingsBody(), jsonCookie(user)).Status; got != 403 {
		t.Fatalf("user shared settings PUT status = %d, want 403", got)
	}
	if got := do(t, env.app, "PUT", "/api/settings/database", sharedSettingsBody(), jsonCookie(admin)).Status; got != 200 {
		t.Fatalf("admin shared settings PUT status = %d, want 200", got)
	}
	adminGet := do(t, env.app, "GET", "/api/settings/database", "", jsonCookie(admin))
	if adminGet.Status != 200 || adminGet.JSON(t)["host"] != "db.example" {
		t.Fatalf("admin GET did not return database settings: %s", adminGet.Body)
	}
	userGet := do(t, env.app, "GET", "/api/settings/database", "", jsonCookie(user))
	if userGet.Status != 200 || userGet.JSON(t)["configured"] != true || userGet.JSON(t)["host"] != nil {
		t.Fatalf("user GET exposed more than configured status: %s", userGet.Body)
	}
}

func sharedSettingsBody() string {
	return `{"host":"db.example","port":5432,"database":"workspace","user":"admin","password":"secret","sslMode":"prefer"}`
}
