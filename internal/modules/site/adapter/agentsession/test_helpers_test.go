package agentsession

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

func sessionURL(t *testing.T) *url.URL {
	t.Helper()
	parsed, err := url.Parse("https://app.example.test")
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func testRegistry(t *testing.T, capacity int, idle, absolute, operation time.Duration, now func() time.Time) *Registry {
	t.Helper()
	registry := NewWithLimits(capacity, idle, absolute, operation, now)
	t.Cleanup(registry.Close)
	return registry
}

func testRegistryWithUserLimit(t *testing.T, capacity, userCap int) *Registry {
	t.Helper()
	registry := NewWithUserLimit(capacity, userCap, time.Hour, 2*time.Hour, time.Second, time.Now)
	t.Cleanup(registry.Close)
	return registry
}

func newTestBoundedJar(t *testing.T) *boundedJar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &boundedJar{jar: jar, keys: make(map[string]cookieRecord)}
}

func sessionScope(userID, taskID string) application.AgentSessionScope {
	return application.AgentSessionScope{UserID: userID, ProjectID: "p", TaskID: taskID, ContainerID: "c"}
}

func setCookie(t *testing.T, registry *Registry, scope application.AgentSessionScope, target *url.URL, value string) {
	t.Helper()
	if err := registry.Do(t.Context(), scope, func(_ context.Context, jar http.CookieJar) error {
		jar.SetCookies(target, []*http.Cookie{{
			Name: "session", Value: value, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func assertCookie(
	t *testing.T, registry *Registry, scope application.AgentSessionScope, target *url.URL, expected string,
) {
	t.Helper()
	if err := registry.Do(t.Context(), scope, func(_ context.Context, jar http.CookieJar) error {
		cookies := jar.Cookies(target)
		if len(cookies) != 1 || cookies[0].Value != expected {
			t.Errorf("cookies = %v, want %q", cookies, expected)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func assertNoCookie(t *testing.T, registry *Registry, scope application.AgentSessionScope, target *url.URL) {
	t.Helper()
	assertCookieCount(t, registry, scope, target, 0)
}

func assertCookieCount(
	t *testing.T, registry *Registry, scope application.AgentSessionScope, target *url.URL, expected int,
) {
	t.Helper()
	if err := registry.Do(t.Context(), scope, func(_ context.Context, jar http.CookieJar) error {
		if got := len(jar.Cookies(target)); got != expected {
			t.Errorf("cookie count = %d, want %d", got, expected)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
