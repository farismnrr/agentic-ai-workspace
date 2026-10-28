package account_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

func TestSharedDatabaseSettingsAreAdminOnly(t *testing.T) {
	env := apptest.NewEnvWith(t, func(cfg *config.Config) { cfg.AdminEmail = "admin@example.com" })
	admin := apptest.CookieOf(t, apptest.Do(t, env.App, "POST", "/api/auth/register",
		`{"name":"Admin","email":"admin@example.com","password":"long-enough-password"}`, apptest.JSONHeaders()))
	user := apptest.CookieOf(t, apptest.Do(t, env.App, "POST", "/api/auth/register",
		`{"name":"User","email":"user@example.com","password":"long-enough-password"}`, apptest.JSONHeaders()))

	if role := apptest.Do(t, env.App, "GET", "/api/auth/me", "",
		apptest.JSONCookie(user)).JSON(t)["role"]; role != "user" {
		t.Fatalf("new account role = %v, want user", role)
	}
	if role := apptest.Do(t, env.App, "GET", "/api/auth/me", "",
		apptest.JSONCookie(admin)).JSON(t)["role"]; role != "admin" {
		t.Fatalf("configured account role = %v, want admin", role)
	}
	if got := apptest.Do(t, env.App, "PUT", "/api/settings/database", sharedSettingsBody(),
		apptest.JSONCookie(user)).Status; got != 403 {
		t.Fatalf("user shared settings PUT status = %d, want 403", got)
	}
	if got := apptest.Do(t, env.App, "PUT", "/api/settings/database", sharedSettingsBody(),
		apptest.JSONCookie(admin)).Status; got != 200 {
		t.Fatalf("admin shared settings PUT status = %d, want 200", got)
	}
	adminGet := apptest.Do(t, env.App, "GET", "/api/settings/database", "", apptest.JSONCookie(admin))
	if adminGet.Status != 200 || adminGet.JSON(t)["host"] != "db.example" {
		t.Fatalf("admin GET did not return database settings: %s", adminGet.Body)
	}
	userGet := apptest.Do(t, env.App, "GET", "/api/settings/database", "", apptest.JSONCookie(user))
	if userGet.Status != 200 || userGet.JSON(t)["configured"] != true || userGet.JSON(t)["host"] != nil {
		t.Fatalf("user GET exposed more than configured status: %s", userGet.Body)
	}
}

func sharedSettingsBody() string {
	return `{"host":"db.example","port":5432,"database":"workspace","user":"admin","password":"secret","sslMode":"prefer"}`
}
