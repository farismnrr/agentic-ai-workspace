package jwt_test

import (
	"testing"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/jwt"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
)

const issuer = "https://example.test"

func load(t *testing.T, dir string) *jwt.Signer {
	t.Helper()

	signer, err := jwt.LoadOrCreate(dir, issuer)
	if err != nil {
		t.Fatal(err)
	}

	return signer
}

func TestKeyIsStableAcrossRestarts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if !load(t, dir).PublicKey().Equal(load(t, dir).PublicKey()) {
		t.Fatal("key must be stable across restarts")
	}
}

func TestSignedTokenVerifiesWithPublishedKey(t *testing.T) {
	t.Parallel()

	signer := load(t, t.TempDir())
	now := time.Now()

	signed, err := signer.Sign(application.IDClaims{
		Subject: "u1", Audience: "client", Email: "a@b.io", EmailVerified: true, Nonce: "n1",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := golangjwt.Parse(signed, func(*golangjwt.Token) (any, error) { return signer.PublicKey(), nil },
		golangjwt.WithValidMethods([]string{"ES256"}), golangjwt.WithIssuer(issuer), golangjwt.WithAudience("client"))
	if err != nil || !parsed.Valid {
		t.Fatalf("token must verify: %v", err)
	}
}
