package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/env/domain"
)

type memoryStore struct{ items map[string][]StoredVariable }

func (s *memoryStore) List(_ context.Context, projectID string) ([]StoredVariable, error) {
	return s.items[projectID], nil
}

func (s *memoryStore) Put(_ context.Context, projectID string, item StoredVariable) error {
	s.items[projectID] = append(s.items[projectID], item)
	return nil
}

func (s *memoryStore) Delete(context.Context, string, string) error { return nil }

type identityCipher struct{}

func (identityCipher) Encrypt(value []byte) (string, error) { return string(value), nil }
func (identityCipher) Decrypt(value string) ([]byte, error) { return []byte(value), nil }

type owners map[string]string

func (o owners) Owner(_ context.Context, id string) (string, string, error) {
	user, ok := o[id]
	if !ok {
		return "", "", domain.ErrNotFound
	}
	return user, "", nil
}

func TestValuesReturnsOnlyTheRequestedProjectsDecryptedMap(t *testing.T) {
	store := &memoryStore{items: map[string][]StoredVariable{
		"one": {{Name: "TOKEN", Sealed: "first", IsSecret: true}},
		"two": {{Name: "TOKEN", Sealed: "second", IsSecret: true}},
	}}
	service := NewService(store, identityCipher{}, owners{"one": "alice"}, time.Now)

	values, err := service.Values(context.Background(), "one")
	if err != nil || values["TOKEN"] != "first" || len(values) != 1 {
		t.Fatalf("Values(one) = %#v, %v", values, err)
	}
}

func TestSecretResolutionRequiresProjectOwnerAndSecretVariable(t *testing.T) {
	store := &memoryStore{items: map[string][]StoredVariable{
		"one": {
			{Name: "LOGIN_EMAIL", Sealed: "person@example.test", IsSecret: true},
			{Name: "PUBLIC_LABEL", Sealed: "hello", IsSecret: false},
		},
	}}
	service := NewService(store, identityCipher{}, owners{"one": "alice"}, time.Now)

	value, err := service.Secret(context.Background(), "alice", "one", "LOGIN_EMAIL")
	if err != nil || value != "person@example.test" {
		t.Fatalf("owned secret = %q, %v", value, err)
	}
	if _, err = service.Secret(context.Background(), "bob", "one", "LOGIN_EMAIL"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign secret error = %v", err)
	}
	if _, err = service.Secret(context.Background(), "alice", "one", "PUBLIC_LABEL"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("non-secret error = %v", err)
	}
}

func TestPutRequiresProjectOwnershipAndValidValues(t *testing.T) {
	store := &memoryStore{items: map[string][]StoredVariable{}}
	service := NewService(store, identityCipher{}, owners{"p1": "alice"}, time.Now)

	if _, err := service.Put(
		context.Background(), "bob", "p1", "TOKEN", "secret", true,
	); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("wrong owner save error = %v", err)
	}
	if _, err := service.Put(
		context.Background(), "alice", "p1", "PATH", "secret", true,
	); !errors.Is(err, domain.ErrReserved) {
		t.Fatalf("reserved name save error = %v", err)
	}
}

func TestSetSecretAllowsBoundedPEMSizeWithoutChangingUserPutLimit(t *testing.T) {
	store := &memoryStore{items: map[string][]StoredVariable{}}
	service := NewService(store, identityCipher{}, owners{"p1": "alice"}, time.Now)
	largePEM := "-----BEGIN CERTIFICATE-----\n" + strings.Repeat("x", 12*1024) + "\n-----END CERTIFICATE-----"
	if err := service.SetSecret(context.Background(), "p1", "DATABASE_SSL_CA", largePEM); err != nil {
		t.Fatalf("server-managed PEM secret rejected: %v", err)
	}
	_, err := service.Put(context.Background(), "alice", "p1", "TOKEN", largePEM, true)
	if !errors.Is(err, domain.ErrInvalidValue) {
		t.Fatalf("user variable exceeded its 4 KiB limit: %v", err)
	}
}
