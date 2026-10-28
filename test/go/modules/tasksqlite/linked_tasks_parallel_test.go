package tasksqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/sqlite"
)

func newConcurrentWorkStore(t *testing.T) (context.Context, *sql.DB, *sqlite.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "linked-tasks.sqlite")
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(12)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, testWorkSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users(id) VALUES ('u'), ('other')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects(id,user_id) VALUES ('p','u'), ('p2','other')`); err != nil {
		t.Fatal(err)
	}
	store := sqlite.NewStore(db, &workTestIDs{})
	createTaskFixture(t, ctx, store, "t1", "u", "p")
	createTaskFixture(t, ctx, store, "t2", "other", "p2")
	createTaskFixture(t, ctx, store, "t3", "u", "p")
	assertMultipleSQLiteConnections(t, ctx, db)
	return ctx, db, store
}

func assertMultipleSQLiteConnections(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	first, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	if db.Stats().OpenConnections < 2 {
		t.Fatal("concurrency fixture did not open multiple SQLite connections")
	}
}
