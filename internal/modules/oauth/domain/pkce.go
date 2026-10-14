package domain

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

const (
	minVerifierLength = 43
	maxVerifierLength = 128
	challengeLength   = 43 // base64url of a SHA-256 digest, unpadded
)

// ValidChallenge reports whether challenge looks like an S256 code challenge.
func ValidChallenge(challenge string) bool {
	return len(challenge) == challengeLength && onlyChars(challenge, isBase64URL)
}

// VerifyS256 checks a PKCE code verifier against its S256 challenge (RFC 7636).
func VerifyS256(verifier, challenge string) bool {
	if len(verifier) < minVerifierLength || len(verifier) > maxVerifierLength {
		return false
	}

	if !onlyChars(verifier, isUnreserved) {
		return false
	}

	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])

	return subtle.ConstantTimeCompare([]byte(expected), []byte(challenge)) == 1
}

func onlyChars(value string, allowed func(rune) bool) bool {
	for _, r := range value {
		if !allowed(r) {
			return false
		}
	}

	return true
}

func isBase64URL(r rune) bool {
	return isAlnum(r) || r == '-' || r == '_'
}

func isUnreserved(r rune) bool {
	return isAlnum(r) || r == '-' || r == '.' || r == '_' || r == '~'
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
