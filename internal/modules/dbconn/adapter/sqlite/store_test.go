package sqlite

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
)

func TestManagedConnectionRoundTrips(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, execErr := db.ExecContext(context.Background(), `INSERT INTO users (id, name, email, password_hash, created_at)
		VALUES ('user-1', 'Test', 'test@example.com', 'hash', 1)`); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := db.ExecContext(context.Background(), `INSERT INTO projects (id, user_id, name, created_at)
		VALUES ('project-1', 'user-1', 'Test', 1)`); execErr != nil {
		t.Fatal(execErr)
	}
	store := NewStore(db)
	want := application.StoredConnection{
		Connection: domain.Connection{
			Kind: domain.Postgres, Host: "db.example", Port: 5432, Database: "workspace",
			User: "project", Password: "unused", EnvName: "DATABASE_URL", Managed: true,
		},
		PasswordSealed: "sealed",
	}
	if saveErr := store.Save(context.Background(), "project-1", want); saveErr != nil {
		t.Fatal(saveErr)
	}
	got, err := store.Get(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Connection.Managed {
		t.Fatal("managed flag was not persisted")
	}

	want.Connection.Managed = false
	if saveErr := store.Save(context.Background(), "project-1", want); saveErr != nil {
		t.Fatal(saveErr)
	}
	got, err = store.Get(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Connection.Managed {
		t.Fatal("manual connection did not retain the false default")
	}
}
