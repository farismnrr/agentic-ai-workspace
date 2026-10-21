package http

import (
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func newTestJar() (*cookieJar, *time.Time) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	jar := newCookieJar()
	jar.now = func() time.Time { return clock }

	return jar, &clock
}

func TestJarReturnsWhatTheAppSetForTheSameVisitorOnly(t *testing.T) {
	jar, _ := newTestJar()
	mine := jar.session("tok", "browser-a")

	mine.record("/login", []string{"sid=abc; Path=/; HttpOnly", "theme=dark"})

	if got := mine.header("/me"); !strings.Contains(got, "sid=abc") || !strings.Contains(got, "theme=dark") {
		t.Errorf("the visitor gets their cookies back, got %q", got)
	}

	for name, other := range map[string]*jarSession{
		"another browser": jar.session("tok", "browser-b"),
		"another site":    jar.session("tok2", "browser-a"),
		"no jar":          nil,
	} {
		if got := other.header("/me"); got != "" {
			t.Errorf("%s must not see the cookies, got %q", name, got)
		}
	}
}

func TestJarForgetsEndedCookies(t *testing.T) {
	jar, clock := newTestJar()
	s := jar.session("tok", "a")

	s.record("/", []string{"sid=abc; Path=/", "short=1; Max-Age=60", "gone=1; Expires=Mon, 02 Jan 2006 15:04:05 GMT"})

	if got := s.header("/"); strings.Contains(got, "gone") || !strings.Contains(got, "short=1") {
		t.Errorf("an already expired cookie is never kept, got %q", got)
	}

	*clock = clock.Add(2 * time.Minute)

	if got := s.header("/"); got != "sid=abc" {
		t.Errorf("Max-Age ends the cookie, got %q", got)
	}

	s.record("/logout", []string{"sid=; Path=/; Max-Age=0"})

	if got := s.header("/"); got != "" {
		t.Errorf("logout clears the login, got %q", got)
	}
}

func TestJarHonoursCookiePaths(t *testing.T) {
	jar, _ := newTestJar()
	s := jar.session("tok", "a")

	s.record("/api/login", []string{"api=1; Path=/api", "folder=1"})

	// A cookie without a Path belongs to the folder of the request that set it (/api here).
	cases := map[string]string{"/api/me": "api=1; folder=1", "/api": "api=1; folder=1", "/apix": "", "/other": ""}
	for path, want := range cases {
		if got := s.header(path); !sameCookies(got, want) {
			t.Errorf("%s: got %q, want %q", path, got, want)
		}
	}
}

func sameCookies(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, part := range strings.Split(a, "; ") {
		if !strings.Contains(b, part) {
			return false
		}
	}

	return true
}

func TestJarIsBounded(t *testing.T) {
	jar, clock := newTestJar()
	s := jar.session("tok", "a")

	lines := []string{"big=" + strings.Repeat("x", jarMaxBytes)}
	for i := range jarMaxCookies + 10 {
		lines = append(lines, "c"+string(rune('a'+i%26))+strings.Repeat("z", i/26)+"=1")
	}

	s.record("/", lines)

	if got := strings.Count(s.header("/"), "="); got > jarMaxCookies || strings.Contains(s.header("/"), "big=") {
		t.Errorf("at most %d small cookies are kept, got %d", jarMaxCookies, got)
	}

	for i := range jarMaxJars + 5 {
		*clock = clock.Add(time.Millisecond)
		jar.session("tok", string(rune(i))).record("/", []string{"a=1"})
	}

	if len(jar.jars) > jarMaxJars {
		t.Errorf("at most %d jars, got %d", jarMaxJars, len(jar.jars))
	}
}

func TestAppLoginSurvivesRequestsWithoutTheBrowserEverSeeingTheCookie(t *testing.T) {
	app, _ := newApp(t, domain.Private, domain.Running)
	page := as("script", "")

	_, header, _ := (call{path: ticketRoot + "login", headers: page}).do(t, app)
	if header.Get("Set-Cookie") != "" {
		t.Errorf("the browser must not receive the app's cookie: %q", header.Get("Set-Cookie"))
	}

	if _, _, body := (call{path: ticketRoot + "me", headers: page}).do(t, app); !hasCookie(body, "sid=abc") {
		t.Errorf("the next request carries the login, got %q", body)
	}

	// The page cannot add cookies of its own, and the person's session never goes through.
	forged := as("script", "owner-session")
	if _, _, body := (call{path: ticketRoot + "me", headers: forged}).do(t, app); !hasCookie(body, "sid=abc") {
		t.Errorf("only the jar speaks for the visitor, got %q", body)
	}

	_, _, _ = (call{path: ticketRoot + "logout", headers: page}).do(t, app)

	if _, _, body := (call{path: ticketRoot + "me", headers: page}).do(t, app); !hasCookie(body, "") {
		t.Errorf("after logout there is no login, got %q", body)
	}
}

func TestAnotherBrowserWithItsOwnTicketIsNotLoggedIn(t *testing.T) {
	app, _ := newApp(t, domain.Private, domain.Running)

	_, _, _ = (call{path: ticketRoot + "login", headers: as("script", "")}).do(t, app)

	b := as("script", "")
	b["User-Agent"] = "browser-b"

	if _, _, body := (call{path: ticketFor("browser-b") + "me", headers: b}).do(t, app); !hasCookie(body, "") {
		t.Errorf("a different browser must not inherit the login, got %q", body)
	}
}

func hasCookie(body, want string) bool { return strings.Contains(body, `"cookie":"`+want+`"`) }
