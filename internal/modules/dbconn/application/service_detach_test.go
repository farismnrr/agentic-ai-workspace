package application

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestDetachSQLiteRemovesManagedDatabaseEnvironment(t *testing.T) {
	store := &memoryStore{records: map[string]StoredConnection{"project-a": {
		Connection: domain.Connection{Kind: domain.Postgres, EnvName: "DATABASE_URL", Managed: true},
	}}}
	env := &memoryEnvironment{values: map[string]string{
		"project-a:DATABASE_URL": "postgres://shared", "project-a:MONGODB_URL": "mongodb://custom",
		"project-a:DATABASE_SSL_CA": "ca", "project-a:DATABASE_SSL_CERT": "cert", "project-a:DATABASE_SSL_KEY": "key",
	}}
	service, _, _, provisioner := provisionService(t, store, env)
	if err := service.DetachSQLite(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.records["project-a"]; exists {
		t.Fatal("project connection remains after selecting SQLite")
	}
	if _, exists := env.values["project-a:DATABASE_URL"]; exists {
		t.Fatal("DATABASE_URL remains after selecting SQLite")
	}
	if env.values["project-a:MONGODB_URL"] != "mongodb://custom" {
		t.Fatal("detach removed an environment variable other than DATABASE_URL")
	}
	for _, name := range []string{"DATABASE_SSL_CA", "DATABASE_SSL_CERT", "DATABASE_SSL_KEY"} {
		if _, exists := env.values["project-a:"+name]; exists {
			t.Fatalf("%s remains after detach", name)
		}
	}
	if provisioner.calls != 0 || provisioner.restored != "" {
		t.Fatal("detach called the provisioner; remote database objects must remain")
	}
}

func TestDetachSQLitePreservesDatabaseURLForCustomConnection(t *testing.T) {
	store := &memoryStore{records: map[string]StoredConnection{"project-a": {
		Connection: domain.Connection{Kind: domain.Postgres, EnvName: "DATABASE_URL"},
	}}}
	env := &memoryEnvironment{values: map[string]string{"project-a:DATABASE_URL": "postgres://custom"}}
	service, _, _, _ := provisionService(t, store, env)
	if err := service.DetachSQLite(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	if got := env.values["project-a:DATABASE_URL"]; got != "postgres://custom" {
		t.Fatalf("custom DATABASE_URL changed: %q", got)
	}
}

func TestDetachSQLitePreservesHandTypedDatabaseURLWithoutManagedRecord(t *testing.T) {
	store := &memoryStore{records: map[string]StoredConnection{}}
	env := &memoryEnvironment{values: map[string]string{
		"project-a:DATABASE_URL": "postgres://hand-typed",
	}}
	service, _, _, _ := provisionService(t, store, env)
	if err := service.DetachSQLite(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	if got := env.values["project-a:DATABASE_URL"]; got != "postgres://hand-typed" {
		t.Fatalf("hand-typed DATABASE_URL changed: %q", got)
	}
}
