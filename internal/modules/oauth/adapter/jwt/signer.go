package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	golangjwt "github.com/golang-jwt/jwt/v5"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
)

const (
	keyFile   = "signing-key.pem"
	keyMode   = 0o600
	kidLength = 16
	coordSize = 32
)

var errBadKey = errors.New("signing key is not a PEM-encoded P-256 EC private key")

// Signer signs ID tokens with a persistent ES256 key and publishes its JWKS.
type Signer struct {
	key    *ecdsa.PrivateKey
	kid    string
	issuer string
}

// LoadOrCreate reads the signing key from dir, generating it on first start.
func LoadOrCreate(dir, issuer string) (*Signer, error) {
	path := filepath.Join(dir, keyFile)

	key, err := readKey(path)
	if errors.Is(err, os.ErrNotExist) {
		key, err = createKey(path)
	}

	if err != nil {
		return nil, err
	}

	return &Signer{key: key, kid: keyID(key), issuer: issuer}, nil
}

// Sign issues an ES256 ID token for the claims.
func (s *Signer) Sign(c application.IDClaims) (string, error) {
	claims := golangjwt.MapClaims{
		"iss": s.issuer, "sub": c.Subject, "aud": c.Audience, "iat": c.IssuedAt.Unix(), "exp": c.ExpiresAt.Unix(),
	}
	if c.Email != "" {
		claims["email"], claims["email_verified"] = c.Email, c.EmailVerified
	}

	if c.Name != "" {
		claims["name"] = c.Name
	}

	if c.Nonce != "" {
		claims["nonce"] = c.Nonce
	}

	token := golangjwt.NewWithClaims(golangjwt.SigningMethodES256, claims)
	token.Header["kid"] = s.kid

	signed, err := token.SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("sign id token: %w", err)
	}

	return signed, nil
}

// PublicKey returns the verification key.
func (s *Signer) PublicKey() *ecdsa.PublicKey {
	return &s.key.PublicKey
}

// JWKS returns the public key set (RFC 7517) for /oauth/jwks.json.
func (s *Signer) JWKS() map[string]any {
	point, _ := s.key.PublicKey.Bytes()
	encode := base64.RawURLEncoding.EncodeToString

	return map[string]any{"keys": []map[string]string{{
		"kty": "EC", "crv": "P-256", "use": "sig", "alg": "ES256", "kid": s.kid,
		"x": encode(point[1 : 1+coordSize]), "y": encode(point[1+coordSize:]),
	}}}
}

func readKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is built from the configured data dir
	if err != nil {
		return nil, fmt.Errorf("read signing key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errBadKey
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse signing key: %w", err)
	}

	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, errBadKey
	}

	return key, nil
}

func createKey(path string) (*ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("encode signing key: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, keyMode); err != nil {
		return nil, fmt.Errorf("write signing key: %w", err)
	}

	return key, nil
}

func keyID(key *ecdsa.PrivateKey) string {
	point, _ := key.PublicKey.Bytes()
	sum := sha256.Sum256(point)

	return base64.RawURLEncoding.EncodeToString(sum[:])[:kidLength]
}
