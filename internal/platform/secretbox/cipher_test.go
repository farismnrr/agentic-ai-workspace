package secretbox_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

func open(t *testing.T, dir string) *secretbox.Cipher {
	t.Helper()

	cipher, err := secretbox.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	return cipher
}

func TestRoundTripAndKeyPersistsAcrossRestarts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sealed, err := open(t, dir).Encrypt([]byte("sk-secret-key"))
	if err != nil {
		t.Fatal(err)
	}

	plain, err := open(t, dir).Decrypt(sealed)
	if err != nil || !bytes.Equal(plain, []byte("sk-secret-key")) {
		t.Fatalf("a restarted cipher must open earlier values: %q %v", plain, err)
	}

	if info, _ := os.Stat(filepath.Join(dir, "secret.key")); info.Mode().Perm() != 0o600 {
		t.Fatalf("the key file must be private, got %v", info.Mode())
	}
}

func TestSealedValuesDifferAndTamperingFails(t *testing.T) {
	t.Parallel()

	cipher := open(t, t.TempDir())
	first, _ := cipher.Encrypt([]byte("same"))
	second, _ := cipher.Encrypt([]byte("same"))

	if first == second {
		t.Fatal("each sealing needs a fresh nonce")
	}

	tampered := first[:len(first)-2] + "AA"
	if _, err := cipher.Decrypt(tampered); err == nil {
		t.Fatal("a modified value must not open")
	}

	if _, err := open(t, t.TempDir()).Decrypt(first); err == nil {
		t.Fatal("a different key must not open the value")
	}
}
