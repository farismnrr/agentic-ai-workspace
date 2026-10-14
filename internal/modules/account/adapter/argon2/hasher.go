package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters follow the OWASP argon2id minimum (19 MiB, 2 iterations, 1 lane).
const (
	memoryKiB  = 19456
	iterations = 2
	lanes      = 1
	saltBytes  = 16
	keyBytes   = 32
)

// Hasher implements password hashing with argon2id in PHC string format.
type Hasher struct{}

// Hash returns a PHC-encoded argon2id hash with a fresh salt.
func (Hasher) Hash(password string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, lanes, keyBytes)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memoryKiB, iterations, lanes,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// Verify reports whether password matches the PHC-encoded hash.
func (Hasher) Verify(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var memory, time uint32

	var threads uint8

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}

	salt, errSalt := base64.RawStdEncoding.DecodeString(parts[4])
	want, errKey := base64.RawStdEncoding.DecodeString(parts[5])

	if errSalt != nil || errKey != nil || len(want) == 0 || memory > 4*memoryKiB {
		return false
	}

	size := uint32(len(want)) //nolint:gosec // want is a decoded 32-byte hash, far below the uint32 range
	got := argon2.IDKey([]byte(password), salt, time, memory, threads, size)

	return subtle.ConstantTimeCompare(got, want) == 1
}
