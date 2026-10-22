package application_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

type sessionProbe struct {
	jar          *cookiejar.Jar
	afterOp      contextError
	reset        bool
	scopeMatches bool
}

func (s *sessionProbe) Do(ctx context.Context, scope application.AgentSessionScope,
	op func(context.Context, http.CookieJar) error,
) error {
	s.scopeMatches = scope.UserID == "u1" && scope.ProjectID == "p1" &&
		(scope.TaskID == "task1" || scope.TaskID == "t1") && scope.ContainerID == "c1"
	if s.jar == nil {
		s.jar, _ = cookiejar.New(nil)
	}
	if err := op(ctx, s.jar); err != nil {
		return err
	}
	return s.registryError()
}

func (*sessionProbe) RevokeTask(_, _ string)                      {}
func (s *sessionProbe) Reset(scope application.AgentSessionScope) { s.reset = scope.TaskID == "task1" }
func (*sessionProbe) RevokeProject(string)                        {}
func (*sessionProbe) RevokeUser(string)                           {}
func (*sessionProbe) Close()                                      {}

type viewerProbe struct {
	jar    http.CookieJar
	prefix string
}

func (v *viewerProbe) Open(_ context.Context, _, _, prefix string, jar http.CookieJar) (string, func(), error) {
	v.prefix, v.jar = prefix, jar
	return "/private/view.sock", func() {}, nil
}

func TestAgentCaptureUsesTaskScopeAndStablePrivatePrefix(t *testing.T) {
	store := &memStore{sites: map[string]domain.Site{}}
	runtime, sessions, viewer := &fakeRuntime{}, &sessionProbe{}, &viewerProbe{}
	runtime.output = printed(true, `"shot":{"width":1280,"height":800,"scale":1},"metrics":{}`)
	service := application.NewService(application.Deps{
		Store: store, Runtime: runtime, View: viewer, Projects: fakeProjects{"p1": {"u1", "shop"}},
		Notify: &counter{}, IDs: &seq{}, Now: time.Now, AgentSessions: sessions,
	})
	if _, err := service.Start(context.Background(), "p1", "serve", 8000); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CaptureForAgent(context.Background(), "u1", "p1", "t1", domain.CaptureSpec{}); err != nil {
		t.Fatal(err)
	}
	want := application.AgentSessionScope{UserID: "u1", ProjectID: "p1", TaskID: "t1", ContainerID: "c1"}
	if !sessions.scopeMatches || want != (application.AgentSessionScope{
		UserID: "u1", ProjectID: "p1", TaskID: "t1", ContainerID: "c1",
	}) {
		t.Fatalf("session scope did not match expected identity: %+v", want)
	}
	if viewer.prefix != "/run/shop" || viewer.jar != sessions.jar || runtime.capture.ViewSocket != "/private/view.sock" {
		t.Fatalf("capture viewer: prefix %q jar shared %t socket %q", viewer.prefix,
			viewer.jar == sessions.jar, runtime.capture.ViewSocket)
	}
}

func TestSessionLoginAndLogoutUseExplicitIndicatorAndClearLocalJar(t *testing.T) {
	store := &memStore{sites: map[string]domain.Site{}}
	runtime, sessions, viewer := &fakeRuntime{}, &sessionProbe{}, &viewerProbe{}
	runtime.sessionOutput = `SESSION_JSON:{"status":"indicator_found"}`
	service := application.NewService(application.Deps{
		Store: store, Runtime: runtime, View: viewer, Projects: fakeProjects{"p1": {"u1", "shop"}},
		Notify: &counter{}, IDs: &seq{}, Now: time.Now, AgentSessions: sessions,
	})
	if _, err := service.Start(ctx, "p1", "serve", 8000); err != nil {
		t.Fatal(err)
	}
	assertSessionLogin(t, service, runtime, sessions, viewer)
	runtime.sessionOutput = `SESSION_JSON:{"status":"indicator_found"}`
	logout := application.SessionAction{
		Action: "logout", Path: "/account", SubmitSelector: "button[type=submit]", ReadySelector: "[data-ready=logged-out]",
	}
	assertLogoutAndReset(t, service, runtime, sessions, store, logout)
}

func assertSessionLogin(
	t *testing.T, service *application.Service, runtime *fakeRuntime, sessions *sessionProbe, viewer *viewerProbe,
) {
	t.Helper()
	login := application.SessionAction{
		Action: "login", Path: "/login", EmailSelector: "#email", PasswordSelector: "#password",
		SubmitSelector: "button[type=submit]", ReadySelector: "[data-ready=account]",
		Email: "test@example.test", Password: "secret",
	}
	result, err := service.RunSessionAction(ctx, "u1", "p1", "task1", login)
	if err != nil || result.Status != "indicator_found" {
		t.Fatalf("login result = %+v, %v", result, err)
	}
	if !runtime.capture.Session || runtime.capture.Prefix != "/run/shop" || viewer.jar != sessions.jar {
		t.Fatalf("session viewer did not use task jar and stable prefix: %+v", runtime.capture)
	}
}

func assertLogoutAndReset(
	t *testing.T, service *application.Service, runtime *fakeRuntime, sessions *sessionProbe,
	store *memStore, logout application.SessionAction,
) {
	t.Helper()
	result, err := service.RunSessionAction(ctx, "u1", "p1", "task1", logout)
	if err != nil || result.Status != "logout_submitted_ready_local_session_cleared" ||
		!sessions.reset {
		t.Fatalf("logout result = %+v, reset = %t, err = %v", result, sessions.reset, err)
	}
	runtime.failure = domain.ErrNoPicture
	sessions.reset = false
	result, err = service.RunSessionAction(ctx, "u1", "p1", "task1", logout)
	if err == nil || result.Status != "logout_failed_local_session_cleared" || !sessions.reset {
		t.Fatalf("failed logout did not clear the local jar: result=%+v reset=%t err=%v", result, sessions.reset, err)
	}
	runtime.failure = nil
	if stopErr := store.MarkStopped(ctx, "p1"); stopErr != nil {
		t.Fatal(stopErr)
	}
	assertStoppedSiteCanReset(t, service, sessions)
}

func assertStoppedSiteCanReset(t *testing.T, service *application.Service, sessions *sessionProbe) {
	t.Helper()
	reset, err := service.RunSessionAction(ctx, "u1", "p1", "task1", application.SessionAction{Action: "reset"})
	if err != nil || reset.Status != "local_session_cleared" || !sessions.reset {
		t.Fatalf("reset with stopped app = %+v, reset=%t err=%v", reset, sessions.reset, err)
	}
}
