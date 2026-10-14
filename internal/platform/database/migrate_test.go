package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

func openAt(t *testing.T, dir string) *sql.DB {
	t.Helper()

	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func exec(t *testing.T, db *sql.DB, query string) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(), query); err != nil {
		t.Fatal(err)
	}
}

func TestJournalRecordsChecksums(t *testing.T) {
	t.Parallel()

	db := openAt(t, t.TempDir())

	var count int

	query := `SELECT COUNT(*) FROM schema_migrations WHERE checksum != ''`
	if err := db.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count < 2 {
		t.Fatalf("every applied migration needs a checksum, got %d", count)
	}
}

func TestReopeningDoesNotReapply(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_ = openAt(t, dir)
	_ = openAt(t, dir)
}

func TestEditedMigrationIsRejected(t *testing.T) {
	t.Parallel()

	db := openAt(t, t.TempDir())
	exec(t, db, `UPDATE schema_migrations SET checksum = 'tampered' WHERE name = '001_accounts.sql'`)

	if err := migrate(context.Background(), db); !errors.Is(err, ErrMigrationEdited) {
		t.Fatalf("edited migration must be rejected, got %v", err)
	}
}

func TestRemovedMigrationIsRejected(t *testing.T) {
	t.Parallel()

	db := openAt(t, t.TempDir())
	exec(t, db, `UPDATE schema_migrations SET name = '999_gone.sql' WHERE name = '001_accounts.sql'`)

	if err := migrate(context.Background(), db); !errors.Is(err, ErrMigrationMissing) {
		t.Fatalf("removed migration must be rejected, got %v", err)
	}
}

func TestProvidersAcceptGeminiAndStillRejectUnknownKinds(t *testing.T) {
	t.Parallel()

	db := openAt(t, t.TempDir())
	exec(t, db, `INSERT INTO users (id, name, email, password_hash, created_at) VALUES ('u1', 'A', 'a@b.c', 'x', 1)`)

	insert := `INSERT INTO providers (id, user_id, kind, name, base_url, key_sealed, key_hint, created_at)
		VALUES ('%s', 'u1', '%s', 'n', 'http://x', 's', 'h', 1)`

	exec(t, db, fmt.Sprintf(insert, "p1", "gemini"))

	if _, err := db.ExecContext(context.Background(), fmt.Sprintf(insert, "p2", "bogus")); err == nil {
		t.Fatal("an unknown provider kind must still be rejected")
	}
}
