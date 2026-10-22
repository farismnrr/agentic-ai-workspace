package application_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func printed(ok bool, extra string) string {
	png := base64.StdEncoding.EncodeToString([]byte("PNGDATA"))

	return "chrome noise\nCAPTURE_JSON:{\"ok\":" + map[bool]string{true: "true", false: "false"}[ok] +
		",\"png\":\"" + png + "\"," + extra + "}\nmore noise\n"
}

func TestACaptureReadsThePictureAndTheFactsOfThePage(t *testing.T) {
	t.Parallel()

	extra := `"shot":{"y":0,"width":1280,"height":800,"scale":1},"console":["error: boom"],` +
		`"failed":["https://f.test/a.woff"],` +
		`"metrics":{"title":"Kopi","lang":"id","width":1280,"pageWidth":1500,"pageHeight":2400,"overflowX":true,` +
		`"viewportMeta":true,"headings":["h1 Kopi enak"],"images":{"total":3,"broken":1,"noAlt":2}}`

	got, err := application.ParseCapture(printed(true, extra))
	if err != nil || string(got.PNG) != "PNGDATA" || got.Page.Title != "Kopi" || got.Frame.Height != 800 {
		t.Fatalf("capture = %+v, %v", got, err)
	}

	text := got.Describe(domain.CaptureSpec{Path: "/"})
	wanted := []string{
		"scrolls sideways", "1 of 3 images did not load", "2 images have no alt", "error: boom", "h1 Kopi enak",
	}

	for _, want := range wanted {
		if !strings.Contains(text, want) {
			t.Errorf("the description should say %q:\n%s", want, text)
		}
	}
}

func TestACaptureThatFailedSaysWhy(t *testing.T) {
	t.Parallel()

	for _, output := range []string{"", "no marker here", printed(false, `"error":"the page did not open"`)} {
		if _, err := application.ParseCapture(output); !errors.Is(err, domain.ErrNoPicture) {
			t.Errorf("%q should be a failed picture: %v", output, err)
		}
	}
}

func TestCaptureDescriptionHidesQueryAndCredentialMarkers(t *testing.T) {
	shot := domain.Capture{Status: 404, Page: domain.Page{
		Title: "token=visible-token", Text: "Authorization: Bearer secret-token", Headings: []string{"csrf: token-value"},
	}}
	text := shot.Describe(domain.CaptureSpec{Path: "/private?token=query-secret"})
	for _, secret := range []string{"query-secret", "visible-token", "secret-token", "token-value"} {
		if strings.Contains(text, secret) {
			t.Errorf("capture description exposed %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "HTTP 404") {
		t.Fatalf("capture status missing from description: %s", text)
	}
}

func TestCaptureOnlyPhotographsARunningAppAtAKnownPage(t *testing.T) {
	t.Parallel()

	service, runtime, _, _ := newService(application.Limits{PerUser: 2, Total: 4})
	runtime.output = printed(true, `"shot":{"width":1280,"height":800,"scale":1},"metrics":{}`)

	if _, err := service.Capture(ctx, "p1", domain.CaptureSpec{}); !errors.Is(err, domain.ErrNotRunning) {
		t.Fatalf("no site yet: %v", err)
	}

	if _, err := service.Start(ctx, "p1", "python3 -m http.server 8000", 8000); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Capture(ctx, "p1", domain.CaptureSpec{Path: "/pricing"}); err != nil {
		t.Fatalf("a running app: %v", err)
	}

	for _, path := range []string{"http://evil.test/", "/../x", "pricing", "/a b", "//evil"} {
		if _, err := service.Capture(ctx, "p1", domain.CaptureSpec{Path: path}); !errors.Is(err, domain.ErrBadPath) {
			t.Errorf("%q must be refused: %v", path, err)
		}
	}
}
