package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

func html(body string) application.Page {
	return application.Page{Status: 200, Type: "text/html", Body: body}
}

func TestAuditFindsLinksThatLeaveTheAddressAndFilesThatAreMissing(t *testing.T) {
	svc, runtime, _, _ := newService(application.Limits{PerUser: 2, Total: 6})
	runtime.pages = map[string]application.Page{
		"/": html(`<link href="/style.css"><link href="./app.css"><a href="admin/">a</a><img src="./logo.png">
			<a href="https://example.com/x">x</a><a href="#top">t</a>`),
		"/app.css":    {Status: 200, Type: "text/css"},
		"/admin/":     html(`<link href="./app.css"><script src="../app.js"></script>`),
		"/app.js":     {Status: 200, Type: "text/javascript"},
		"/logo.png":   {Status: 200, Type: "text/html"},
		"/style.css":  {Status: 200, Type: "text/html"},
		"/other.html": html(""),
	}

	site, err := svc.Start(ctx, "p1", "serve", 3000)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.Join(svc.Audit(ctx, site), "\n")

	wanted := []string{
		"`/style.css` on `/` points outside", "`./logo.png` on `/` answers with a web page",
		"`./app.css` on `/admin/` answers 404",
	}

	for _, want := range wanted {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}

	for _, unwanted := range []string{"`./app.css` on `/`", "example.com", "#top"} {
		if !strings.Contains(got, unwanted) {
			continue
		}

		t.Errorf("working or external links must not be reported:\n%s", got)
	}
}

func TestAuditPassesAnAppWhoseLinksAreRelative(t *testing.T) {
	svc, runtime, _, _ := newService(application.Limits{PerUser: 2, Total: 6})
	runtime.pages = map[string]application.Page{
		"/":          html(`<link href="./style.css"><a href="./docs/">d</a>`),
		"/docs/":     html(`<link href="../style.css">`),
		"/style.css": {Status: 200, Type: "text/css"},
	}

	site, _ := svc.Start(ctx, "p1", "serve", 3000)
	if got := svc.Audit(ctx, site); len(got) != 0 {
		t.Fatalf("a correct app has no problems: %v", got)
	}
}
