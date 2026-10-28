package database_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
)

func openAt(t *testing.T, dir string) *sql.DB {
	t.Helper()

	db, err := database.Open(dir)
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

	dir := t.TempDir()
	db := openAt(t, dir)

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

	assertStartupRejectsJournalTampering(
		t,
		`UPDATE schema_migrations SET checksum = 'tampered' WHERE name = '001_accounts.sql'`,
		database.ErrMigrationEdited,
	)
}

func TestRemovedMigrationIsRejected(t *testing.T) {
	t.Parallel()

	assertStartupRejectsJournalTampering(
		t,
		`UPDATE schema_migrations SET name = '999_gone.sql' WHERE name = '001_accounts.sql'`,
		database.ErrMigrationMissing,
	)
}

func assertStartupRejectsJournalTampering(t *testing.T, update string, want error) {
	t.Helper()

	dir := t.TempDir()
	db := openAt(t, dir)
	exec(t, db, update)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(dir)
	if err == nil {
		_ = reopened.Close()
		t.Fatalf("startup must reject journal tampering, wanted %v", want)
	}
	if !errors.Is(err, want) {
		t.Fatalf("startup error = %v, want %v", err, want)
	}
}
