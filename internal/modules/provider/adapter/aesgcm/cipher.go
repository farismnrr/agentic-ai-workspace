package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	keyFile = "secret.key"
	keySize = 32
	keyMode = 0o600
)

var (
	errShort   = errors.New("sealed value is too short")
	errKeySize = errors.New("secret.key has the wrong size; restore it from backup rather than regenerating")
)

// Cipher seals secrets with AES-256-GCM. The key lives in DATA_DIR/secret.key,
// generated on first start, so secrets in the database are useless alone.
type Cipher struct {
	aead cipher.AEAD
}

// Open loads the key (creating it when missing) and prepares the cipher.
func Open(dataDir string) (*Cipher, error) {
	key, err := loadOrCreate(filepath.Join(dataDir, keyFile))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

// Encrypt returns nonce||ciphertext, base64-encoded.
func (c *Cipher) Encrypt(plain []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(c.aead.Seal(nonce, nonce, plain, nil)), nil
}

// Decrypt opens a value produced by Encrypt; tampering makes it fail.
func (c *Cipher) Decrypt(sealed string) ([]byte, error) {
	data, err := base64.RawStdEncoding.DecodeString(sealed)
	if err != nil {
		return nil, fmt.Errorf("decode sealed value: %w", err)
	}

	size := c.aead.NonceSize()
	if len(data) < size {
		return nil, errShort
	}

	plain, err := c.aead.Open(nil, data[:size], data[size:], nil)
	if err != nil {
		return nil, fmt.Errorf("open sealed value: %w", err)
	}

	return plain, nil
}

func loadOrCreate(path string) ([]byte, error) {
	key, err := os.ReadFile(path) //nolint:gosec // path is built from the configured data dir
	if err == nil && len(key) == keySize {
		return key, nil
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read secret key: %w", err)
	}

	if err == nil {
		return nil, errKeySize
	}

	key = make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate secret key: %w", err)
	}

	if err := os.WriteFile(path, key, keyMode); err != nil {
		return nil, fmt.Errorf("write secret key: %w", err)
	}

	return key, nil
}
