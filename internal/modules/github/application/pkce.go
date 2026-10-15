package application

import (
	"crypto/sha256"
	"encoding/base64"
)

// challenge is the PKCE S256 challenge for a verifier.
func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}
