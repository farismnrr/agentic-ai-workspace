package database

import (
	"context"
	"database/sql"
	"errors"
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
