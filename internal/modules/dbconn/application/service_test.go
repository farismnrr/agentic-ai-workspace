package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

var errProjectMissing = errors.New("project missing")

func TestSavedConnectionIsOwnerScoped(t *testing.T) {
	service, store, env, input := savedConnection(t)
	if _, err := service.Get(context.Background(), "user-b", "project-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other project owner should not read connection: %v", err)
	}
	if _, exists := store.records["project-b"]; exists {
		t.Fatal("connection crossed project boundary")
	}
	if env.values["project-a:DATABASE_URL"] != input.URL() {
		t.Fatal("generated URL was not stored as the project secret")
	}
}

func TestReadMetadataDoesNotReturnPassword(t *testing.T) {
	service, _, _, input := savedConnection(t)
	read, err := service.Get(context.Background(), "user-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(read)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), input.Password) || strings.Contains(string(encoded), input.URL()) {
		t.Fatal("connection response contains a secret")
	}
}

func savedConnection(t *testing.T) (*Service, *memoryStore, *memoryEnvironment, domain.Connection) {
	t.Helper()
	store := memoryStore{records: map[string]StoredConnection{}}
	env := memoryEnvironment{values: map[string]string{}}
	service := NewService(&store, testCipher{}, fixedOwner{"project-a": "user-a", "project-b": "user-b"},
		&env, healthyProbe{})
	input := domain.Connection{
		Kind: domain.Postgres, Host: "db.local", Port: 5432, Database: "records",
		User: "db-user", Password: strings.Join([]string{"s:", "@/?", "#%"}, ""), TLS: true,
	}
	metadata, err := service.Save(context.Background(), "user-a", "project-a", input)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.EnvName != "DATABASE_URL" || !metadata.Configured {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	return service, &store, &env, input
}

type fixedOwner map[string]string

func (f fixedOwner) Owner(_ context.Context, projectID string) (string, string, error) {
	owner, ok := f[projectID]
	if !ok {
		return "", "", errProjectMissing
	}
	return owner, "", nil
}

type testCipher struct{}

func (testCipher) Encrypt(value []byte) (string, error) { return "sealed:" + string(value), nil }
func (testCipher) Decrypt(value string) ([]byte, error) { return []byte(value[len("sealed:"):]), nil }

type memoryStore struct{ records map[string]StoredConnection }

func (m *memoryStore) Get(_ context.Context, projectID string) (StoredConnection, error) {
	item, ok := m.records[projectID]
	if !ok {
		return item, ErrNotFound
	}
	return item, nil
}

func (m *memoryStore) Save(_ context.Context, projectID string, item StoredConnection) error {
	m.records[projectID] = item
	return nil
}

func (m *memoryStore) Delete(_ context.Context, projectID string) error {
	delete(m.records, projectID)
	return nil
}

type memoryEnvironment struct{ values map[string]string }

func (m *memoryEnvironment) SetSecret(_ context.Context, projectID, name, value string) error {
	m.values[projectID+":"+name] = value
	return nil
}

func (m *memoryEnvironment) DeleteSecret(_ context.Context, projectID, name string) error {
	delete(m.values, projectID+":"+name)
	return nil
}

type healthyProbe struct{}

func (healthyProbe) Probe(context.Context, domain.Connection) error { return nil }
