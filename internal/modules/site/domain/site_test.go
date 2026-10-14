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

func TestCanView(t *testing.T) {
	site := func(v domain.Visibility, s domain.Status) domain.Site {
		return domain.Site{UserID: "owner", Visibility: v, Status: s}
	}

	cases := map[string]struct {
		user     string
		site     domain.Site
		document bool
		want     bool
	}{
		"private page, owner":         {"owner", site(domain.Private, domain.Running), true, true},
		"private page, anonymous":     {"", site(domain.Private, domain.Running), true, false},
		"private page, someone else":  {"other", site(domain.Private, domain.Running), true, false},
		"private asset, token holder": {"", site(domain.Private, domain.Running), false, true},
		"public page, anonymous":      {"", site(domain.Public, domain.Running), true, true},
		"stopped public":              {"owner", site(domain.Public, domain.Stopped), true, false},
		"stopped private asset":       {"owner", site(domain.Private, domain.Stopped), false, false},
	}

	for name, c := range cases {
		if got := c.site.CanView(c.user, c.document); got != c.want {
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
	if site.BasePath() != "/run/sensio/abc" {
		t.Fatal(site.BasePath())
	}
}
