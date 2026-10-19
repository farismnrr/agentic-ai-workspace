package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Sensio":           "sensio",
		"  My Cool App!! ": "my-cool-app",
		"Toko Kué 2024":    "toko-ku-2024",
		"日本語":              "site",
		"":                 "site",
		"a/b\\c..d":        "a-b-c-d",
		"x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x": "x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x",
	}

	for in, want := range cases {
		if got := domain.Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

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
