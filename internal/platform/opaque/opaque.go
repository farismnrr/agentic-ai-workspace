package opaque

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	tokenBytes = 32
	idBytes    = 16

	// Positions and masks that mark a UUID as random (version 4, RFC 4122 variant).
	versionByte    = 6
	variantByte    = 8
	lowNibble      = 0x0f
	lowSixBits     = 0x3f
	version4       = 0x40
	variantRFC4122 = 0x80
)

// Generator mints high-entropy secrets and identifiers. Secrets are only ever
// stored as their SHA-256 hash, so a database leak does not expose them.
type Generator struct{}

// New returns a random URL-safe secret and its storage hash.
func (Generator) New() (string, string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("read random bytes: %w", err)
	}

	plain := base64.RawURLEncoding.EncodeToString(buf)

	return plain, Generator{}.Hash(plain), nil
}

// Hash returns the storage hash of a secret.
func (Generator) Hash(plain string) string {
	sum := sha256.Sum256([]byte(plain))

	return hex.EncodeToString(sum[:])
}

// ID returns a random identifier.
func (Generator) ID() (string, error) {
	buf := make([]byte, idBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}

// UUID returns a random version 4 UUID, such as the secret part of a site address.
func (Generator) UUID() (string, error) {
	buf := make([]byte, idBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	buf[versionByte] = buf[versionByte]&lowNibble | version4
	buf[variantByte] = buf[variantByte]&lowSixBits | variantRFC4122

	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:]), nil
}
