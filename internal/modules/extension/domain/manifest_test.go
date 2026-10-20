package domain_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
)

func manifestOf(t *testing.T, text string) domain.Manifest {
	t.Helper()

	m, err := domain.ParseManifest([]byte(text))
	if err != nil {
		t.Fatal(err)
	}

	return m
}

func TestAManifestThatTheBrowsersWouldRefuseIsSentBackWithWhatToFix(t *testing.T) {
	t.Parallel()

	m := manifestOf(t, `{"manifest_version":2,"name":"`+strings.Repeat("x", 50)+`","version":"one",
	  "content_security_policy":{"extension_pages":"script-src 'self' 'unsafe-eval'"}}`)

	got := strings.Join(m.Problems(), "|")
	for _, want := range []string{"manifest_version must be 3", "name is required", "version must be numbers", "eval"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}

	good := manifestOf(t, `{"manifest_version":3,"name":"Notes","version":"1.2.3"}`)
	if len(good.Problems()) != 0 {
		t.Fatalf("a good manifest has no problems: %v", good.Problems())
	}
}

func TestReferencedFilesAreEverythingTheManifestPointsAt(t *testing.T) {
	t.Parallel()

	m := manifestOf(t, `{"manifest_version":3,"name":"n","version":"1","background":{"service_worker":"sw.js"},
	  "action":{"default_popup":"popup.html"},"icons":{"16":"/i16.png"},
	  "content_scripts":[{"matches":["https://a.test/*"],"js":["c.js"],"css":["c.css"]}]}`)

	got := strings.Join(m.Referenced(), ",")
	for _, want := range []string{"sw.js", "popup.html", "i16.png", "c.js", "c.css"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
}

func TestWhatAnExtensionMayDoIsToldInPlainWordsWithTheRiskiestFirst(t *testing.T) {
	t.Parallel()

	m := manifestOf(t, `{"manifest_version":3,"name":"n","version":"1","permissions":["storage","cookies","unknownthing"],
	  "host_permissions":["<all_urls>","https://*.example.com/*"],
	  "content_scripts":[{"matches":["https://*.example.com/*"],"js":["c.js"]}]}`)

	caps := m.Capabilities()
	if len(caps) < 3 || caps[0].Risk != domain.RiskHigh || caps[len(caps)-1].Risk != domain.RiskLow {
		t.Fatalf("the risky ones come first: %+v", caps)
	}

	text := ""
	seen := map[string]int{}

	for _, c := range caps {
		text += c.Text + "|"
		seen[c.Text]++
	}

	for _, want := range []string{"everything on every website", "cookies", "example.com", "Remember its settings"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %s", want, text)
		}
	}

	for words, count := range seen {
		if count > 1 {
			t.Errorf("%q is listed %d times", words, count)
		}
	}
}
