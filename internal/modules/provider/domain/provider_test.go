package domain_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

func TestGeminiHasAHostedDefaultAddress(t *testing.T) {
	t.Parallel()

	if !domain.KindGemini.Valid() {
		t.Fatal("gemini must be a valid kind")
	}

	got, err := domain.NormalizeBaseURL(domain.KindGemini, "")
	want := "https://generativelanguage.googleapis.com/v1beta/openai"

	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

func TestOnlyCompatibleNeedsAnAddress(t *testing.T) {
	t.Parallel()

	if _, err := domain.NormalizeBaseURL(domain.KindCompatible, ""); err == nil {
		t.Fatal("a compatible gateway has no default address")
	}
}

func TestHostedProvidersAreLockedToTheirOwnAddress(t *testing.T) {
	t.Parallel()

	for _, kind := range []domain.Kind{domain.KindAnthropic, domain.KindOpenAI, domain.KindGemini} {
		fixed := domain.DefaultBaseURL(kind)

		if got, err := domain.NormalizeBaseURL(kind, " "+fixed+"/ "); err != nil || got != fixed {
			t.Errorf("%s: its own address must be accepted, got %q, %v", kind, got, err)
		}

		for _, other := range []string{"http://localhost:20128/v1", "https://evil.example.com", "http://169.254.169.254"} {
			if _, err := domain.NormalizeBaseURL(kind, other); !errors.Is(err, domain.ErrAddressFixed) {
				t.Errorf("%s must refuse %q, got %v", kind, other, err)
			}
		}
	}
}

func TestGatewaysMustBeOnThisComputer(t *testing.T) {
	t.Parallel()

	accepted := []string{
		"http://localhost:20128/v1", "http://127.0.0.1:20128/v1", "https://localhost/v1", "http://[::1]:1234",
	}

	for _, ok := range accepted {
		if _, err := domain.NormalizeBaseURL(domain.KindCompatible, ok); err != nil {
			t.Errorf("%q is on this computer and must be accepted: %v", ok, err)
		}
	}

	bad := []string{
		"http://192.168.1.5:20128/v1", "http://10.0.0.1/v1", "https://api.example.com/v1", "http://169.254.169.254/",
		"http://localhost.evil.com/v1", "http://127.0.0.1.nip.io/v1", "http://user:pass@localhost/v1",
		"ftp://localhost/v1", "http://localhost/v1?x=1", "localhost:20128", "",
	}

	for _, value := range bad {
		if _, err := domain.NormalizeBaseURL(domain.KindCompatible, value); !errors.Is(err, domain.ErrInvalidBaseURL) {
			t.Errorf("%q must be refused, got %v", value, err)
		}
	}
}
