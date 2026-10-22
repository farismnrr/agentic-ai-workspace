package runtime_test

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sitehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	"golang.org/x/net/publicsuffix"
)

//go:embed testdata/session_fixture.py
var sessionFixture string

type fixtureRuntime interface {
	Session(context.Context, application.CaptureRequest) (string, error)
	Capture(context.Context, application.CaptureRequest) (string, error)
	Socket(string) string
}

func TestAgentSessionFormAndJSONFlowAcrossProtectedScreens(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "session_fixture.py"), []byte(sessionFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	rt := newRuntimeWithProject(t, project)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	id, err := rt.Start(ctx, application.StartSpec{
		ProjectID: "session-fixture", BasePath: "/run/fixture", Port: 8000,
		Command: "exec python3 session_fixture.py",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx, id)
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal(err)
	}
	invalid := fixtureLogin("/login-form")
	invalid.Password = "wrong-password"
	assertFixtureSessionStatus(t, ctx, rt, jar, invalid, "login_failed")
	if shot := captureFixturePage(t, ctx, rt, jar, "/missing", false); shot.Status != 404 {
		t.Fatalf("404 response status = %d, want 404", shot.Status)
	}
	external := application.SessionAction{
		Action: "logout", Path: "/external", SubmitSelector: "button[type=submit]",
		ReadySelector: `[data-ready="logged-out"]`,
	}
	assertFixtureSessionStatus(t, ctx, rt, jar, external, "external_redirect_blocked")
	login := fixtureLogin("/login-form")
	assertFixtureSessionStatus(t, ctx, rt, jar, login, "indicator_found")
	assertProtectedMatrix(t, ctx, rt, jar)
	logout := application.SessionAction{
		Action: "logout", Path: "/logout", SubmitSelector: "button[type=submit]", ReadySelector: `[data-ready="logged-out"]`,
	}
	assertFixtureSessionStatus(t, ctx, rt, jar, logout, "indicator_found")
	assertProtectedCapture(t, ctx, rt, jar, "/private-one", false, "Sign in required", 1280)
	login = fixtureLogin("/login-json")
	assertFixtureSessionStatus(t, ctx, rt, jar, login, "indicator_found")
	assertProtectedMatrix(t, ctx, rt, jar)
	assertOffscreenSubmitAndNonclickableFailure(t, ctx, rt)
	if shot := captureFixturePage(t, ctx, rt, jar, "/expire", false); shot.Status != 200 {
		t.Fatalf("expiry response status = %d, want 200", shot.Status)
	}
	assertProtectedCapture(t, ctx, rt, jar, "/private-one", false, "Sign in required", 1280)
}

func assertProtectedMatrix(t *testing.T, ctx context.Context, rt fixtureRuntime, jar http.CookieJar) {
	t.Helper()
	for _, screen := range []struct{ path, text string }{
		{path: "/private-one", text: "Protected one"},
		{path: "/private-two", text: "Protected two"},
	} {
		assertProtectedCapture(t, ctx, rt, jar, screen.path, false, screen.text, 1280)
		assertProtectedCapture(t, ctx, rt, jar, screen.path, true, screen.text, 390)
	}
}

func assertOffscreenSubmitAndNonclickableFailure(t *testing.T, ctx context.Context, rt fixtureRuntime) {
	t.Helper()
	jar := newFixtureJar(t)
	action := fixtureLogin("/login-form-offscreen")
	assertFixtureSessionStatus(t, ctx, rt, jar, action, "indicator_found")
	assertProtectedCapture(t, ctx, rt, jar, "/private-one", false, "Protected one", 1280)
	hiddenJar := newFixtureJar(t)
	hidden := fixtureLogin("/login-form-hidden")
	output := runFixtureSession(t, ctx, rt, hiddenJar, hidden)
	if strings.Contains(output, `"status":"indicator_found"`) || strings.Contains(output, `"status":"login_failed"`) {
		t.Fatalf("hidden submit was reported successful: %s", output)
	}
}

func newFixtureJar(t *testing.T) http.CookieJar {
	t.Helper()
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal(err)
	}
	return jar
}

func assertFixtureSessionStatus(
	t *testing.T, ctx context.Context, rt fixtureRuntime, jar http.CookieJar,
	action application.SessionAction, expected string,
) {
	t.Helper()
	output := runFixtureSession(t, ctx, rt, jar, action)
	want := `"status":"` + expected + `"`
	if !strings.Contains(output, want) {
		t.Fatalf("session status = %s, want %s", output, expected)
	}
}

func fixtureLogin(path string) application.SessionAction {
	return application.SessionAction{
		Action: "login", Path: path, EmailSelector: "#email", PasswordSelector: "#password",
		SubmitSelector: "button[type=submit]", ReadySelector: `[data-ready="one"]`,
		ErrorSelector: "#login-error", Email: "agent@example.test", Password: "fixture-password",
	}
}

func runFixtureSession(
	t *testing.T, ctx context.Context, rt fixtureRuntime, jar http.CookieJar, action application.SessionAction,
) string {
	t.Helper()
	viewer, closeView, err := sitehttp.NewViewer().Open(
		ctx, rt.Socket("session-fixture"), "/run/fixture", "/run/fixture", jar,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer closeView()
	output, err := rt.Session(ctx, application.CaptureRequest{
		ProjectID: "session-fixture", Port: 8000, Prefix: "/run/fixture", ViewSocket: viewer,
		Spec: domain.CaptureSpec{Path: action.Path}, Session: &action,
	})
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func assertProtectedCapture(
	t *testing.T, ctx context.Context, rt fixtureRuntime, jar http.CookieJar,
	path string, mobile bool, expected string, width int,
) {
	t.Helper()
	shot := captureFixturePage(t, ctx, rt, jar, path, mobile)
	if shot.Frame.Width != width || !strings.Contains(shot.Page.Text, expected) {
		t.Fatalf("capture %s: width=%d text=%q", path, shot.Frame.Width, shot.Page.Text)
	}
}

func captureFixturePage(
	t *testing.T, ctx context.Context, rt fixtureRuntime, jar http.CookieJar, path string, mobile bool,
) domain.Capture {
	t.Helper()
	viewer, closeView, err := sitehttp.NewViewer().Open(
		ctx, rt.Socket("session-fixture"), "/run/fixture", "/run/fixture", jar,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer closeView()
	output, err := rt.Capture(ctx, application.CaptureRequest{
		ProjectID: "session-fixture", Port: 8000, Prefix: "/run/fixture", ViewSocket: viewer,
		Spec: domain.CaptureSpec{Path: path, Mobile: mobile},
	})
	if err != nil {
		t.Fatal(err)
	}
	shot, err := application.ParseCapture(output)
	if err != nil {
		t.Fatalf("capture %s failed: %v", path, err)
	}
	return shot
}
