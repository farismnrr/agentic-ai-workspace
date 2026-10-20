package domain_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestDecide(t *testing.T) {
	site := func(v domain.Visibility, s domain.Status) domain.Site {
		return domain.Site{UserID: "owner", Visibility: v, Status: s}
	}
	private, public := site(domain.Private, domain.Running), site(domain.Public, domain.Running)
	owner, other := domain.Access{UserID: "owner"}, domain.Access{UserID: "other"}
	page := func(a domain.Access) domain.Access { a.Page = true; return a }
	ticket := func(a domain.Access) domain.Access { a.Ticket = true; return a }

	cases := map[string]struct {
		site   domain.Site
		access domain.Access
		want   domain.Verdict
	}{
		"private page, owner":                    {private, page(owner), domain.Allow},
		"private asset, owner":                   {private, owner, domain.Allow},
		"private page, anonymous":                {private, page(domain.Access{}), domain.AskSignIn},
		"private page, someone else":             {private, page(other), domain.AskSignIn},
		"private page, anonymous with ticket":    {private, ticket(page(domain.Access{})), domain.AskSignIn},
		"private asset, ticket":                  {private, ticket(domain.Access{}), domain.Allow},
		"private asset, nothing":                 {private, domain.Access{}, domain.Deny},
		"private asset, someone else, no ticket": {private, other, domain.Deny},
		"public page, anonymous":                 {public, page(domain.Access{}), domain.Allow},
		"stopped public":                         {site(domain.Public, domain.Stopped), owner, domain.Missing},
		"stopped private":                        {site(domain.Private, domain.Stopped), ticket(owner), domain.Missing},
	}

	for name, c := range cases {
		if got := c.site.Decide(c.access); got != c.want {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
	}
}

func TestPortAndVisibilityValidation(t *testing.T) {
	for _, port := range []int{0, 80, 1023, 65536} {
		if domain.ValidPort(port) == nil {
			t.Errorf("port %d must be refused", port)
		}
	}

	if domain.ValidPort(3000) != nil || domain.ValidPort(65535) != nil {
		t.Error("ordinary ports must be accepted")
	}

	if v, err := domain.ParseVisibility(" Public "); err != nil || v != domain.Public {
		t.Errorf("got %q %v", v, err)
	}

	if _, err := domain.ParseVisibility("everyone"); err == nil {
		t.Error("unknown visibility must be refused")
	}
}

func TestBasePath(t *testing.T) {
	site := domain.Site{Slug: "sensio", Token: "abc"}
	if site.BasePath() != "/run/sensio" {
		t.Fatal(site.BasePath())
	}
}

func TestACaptureNamesWhatMakesThePageUnusable(t *testing.T) {
	var c domain.Capture

	c.Failed = []string{"stylesheet /_next/a.css answered 200 text/html", "https://fonts.example/x.woff2"}
	c.Page.Images.Total, c.Page.Images.Broken = 3, 1

	got := strings.Join(c.Problems(), "|")
	for _, want := range []string{"no styling at all", "/_next/a.css answered 200 text/html", "1 of 3 images"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}

	if strings.Contains(got, "fonts.example") {
		t.Errorf("a file from another website is not the app's fault: %s", got)
	}

	c.Page.Stylesheets.Rules = 40
	if len((domain.Capture{Page: c.Page}).Problems()) != 1 {
		t.Error("only the broken image is left once the page has styling")
	}
}
