package domain

import (
	"slices"
	"testing"
)

func TestTargetKeepsTheNameAndPortOfAGatewayOnThisComputer(t *testing.T) {
	good := map[string]int{
		"http://localhost:20128/v1": 20128, "http://127.0.0.1:8080": 8080,
		"http://localhost/v1": 80, "https://localhost/v1": 443,
	}
	for in, port := range good {
		if _, got, ok := Target(in); !ok || got != port {
			t.Errorf("%s: port %d ok %v, want %d", in, got, ok, port)
		}
	}
}

func TestTargetRefusesAnythingElse(t *testing.T) {
	for _, in := range []string{
		"", "ftp://localhost:1", "http://example.com:20128", "http://10.0.0.5:20128", "http://[::1]:20128",
		"http://localhost:0", "http://localhost:99999", "localhost:20128",
	} {
		if host, _, ok := Target(in); ok {
			t.Errorf("%q must not be a target (host %q)", in, host)
		}
	}
}

func TestEnvironmentCarriesTheKeyOnlyWhenThereIsOne(t *testing.T) {
	base := "http://localhost:20128/v1"

	if got := Environment(base, ""); !slices.Equal(got, []string{"OPENAI_BASE_URL=" + base}) {
		t.Errorf("no key: %v", got)
	}

	if got := Environment(base, "k"); len(got) != 2 || got[1] != "OPENAI_API_KEY=k" {
		t.Errorf("with key: %v", got)
	}
}
