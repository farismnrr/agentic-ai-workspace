package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// RFC 7636 appendix B test vector.
const (
	rfcVerifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	rfcChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
)

func TestVerifyS256(t *testing.T) {
	t.Parallel()

	if !domain.VerifyS256(rfcVerifier, rfcChallenge) {
		t.Fatal("RFC 7636 vector should verify")
	}

	for name, verifier := range map[string]string{
		"wrong":     "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXl",
		"too short": "short",
		"bad chars": rfcVerifier[:42] + "!",
	} {
		if domain.VerifyS256(verifier, rfcChallenge) {
			t.Errorf("%s verifier must not verify", name)
		}
	}
}

func TestValidateRedirectURI(t *testing.T) {
	t.Parallel()

	good := []string{
		"https://chatgpt.com/connector_platform_oauth_redirect",
		"http://localhost:6274/oauth/callback",
		"http://127.0.0.1:3000/cb",
	}
	bad := []string{"http://example.com/cb", "https://example.com/cb#frag", "javascript:alert(1)", "/relative", ""}

	for _, uri := range good {
		if err := domain.ValidateRedirectURI(uri); err != nil {
			t.Errorf("%q should be accepted: %v", uri, err)
		}
	}

	for _, uri := range bad {
		if err := domain.ValidateRedirectURI(uri); err == nil {
			t.Errorf("%q should be rejected", uri)
		}
	}
}

func TestIsMetadataClientID(t *testing.T) {
	t.Parallel()

	if !domain.IsMetadataClientID("https://chatgpt.com/oauth/client.json") {
		t.Error("CIMD url should be recognised")
	}

	notClientIDs := []string{"abc123", "http://chatgpt.com/client.json", "https://chatgpt.com", "https://chatgpt.com/#x"}

	for _, id := range notClientIDs {
		if domain.IsMetadataClientID(id) {
			t.Errorf("%q is not a metadata client id", id)
		}
	}
}

func TestClientAllowsRedirectIsExact(t *testing.T) {
	t.Parallel()

	client := domain.Client{RedirectURIs: []string{"https://app.example/cb"}}
	if !client.AllowsRedirect("https://app.example/cb") || client.AllowsRedirect("https://app.example/cb/") {
		t.Error("redirect matching must be exact")
	}
}

func TestClientVerifySecret(t *testing.T) {
	t.Parallel()

	hasher := func(s string) string { return "hashed:" + s }

	publicClient := domain.Client{ID: "pub"}
	if publicClient.IsConfidential() {
		t.Error("client without secret hash should not be confidential")
	}
	if publicClient.VerifySecret("any", hasher) {
		t.Error("public client should never verify a secret")
	}

	confClient := domain.Client{
		ID:         "conf",
		AuthMethod: domain.AuthMethodClientSecretBasic,
		SecretHash: "hashed:my-secret",
	}
	if !confClient.IsConfidential() {
		t.Error("client with secret hash should be confidential")
	}
	if !confClient.VerifySecret("my-secret", hasher) {
		t.Error("correct secret should verify")
	}
	if confClient.VerifySecret("wrong-secret", hasher) {
		t.Error("wrong secret must not verify")
	}
	if confClient.VerifySecret("", hasher) {
		t.Error("empty secret must not verify")
	}
}
