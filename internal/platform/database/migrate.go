package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// The journal records every applied migration with the checksum of its file.
// Migrations are append-only: changing a query means adding a new numbered
// file, never editing one that has already been applied.
const createJournal = `CREATE TABLE IF NOT EXISTS schema_migrations (
	name       TEXT PRIMARY KEY,
	checksum   TEXT NOT NULL,
	applied_at INTEGER NOT NULL
)`

var (
	// ErrMigrationEdited means an applied migration file no longer matches its journal checksum.
	ErrMigrationEdited = errors.New("applied migration was edited; add a new migration instead")
	// ErrMigrationMissing means the journal lists a migration whose file was removed.
	ErrMigrationMissing = errors.New("applied migration file is missing")
)

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createJournal); err != nil {
		return fmt.Errorf("create migration journal: %w", err)
	}

	names, err := migrationNames()
	if err != nil {
		return err
	}

	if err := verifyJournal(ctx, db, names); err != nil {
		return err
	}

	for _, name := range names {
		if err := apply(ctx, db, name); err != nil {
			return err
		}
	}

	return nil
}

func migrationNames() ([]string, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	sort.Strings(names)

	return names, nil
}

// verifyJournal rejects edited or deleted migrations before anything runs.
func verifyJournal(ctx context.Context, db *sql.DB, names []string) error {
	rows, err := db.QueryContext(ctx, `SELECT name, checksum FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read migration journal: %w", err)
	}

	defer func() { _ = rows.Close() }()

	known := map[string]bool{}
	for _, name := range names {
		known[name] = true
	}

	for rows.Next() {
		var name, stored string
		if err := rows.Scan(&name, &stored); err != nil {
			return fmt.Errorf("scan migration journal: %w", err)
		}

		if !known[name] {
			return fmt.Errorf("%w: %s", ErrMigrationMissing, name)
		}

		body, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if checksum(body) != stored {
			return fmt.Errorf("%w: %s", ErrMigrationEdited, name)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("read migration journal: %w", err)
	}

	return nil
}

func apply(ctx context.Context, db *sql.DB, name string) error {
	var applied int
	query := `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`
	if err := db.QueryRowContext(ctx, query, name).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %s: %w", name, err)
	}

	if applied > 0 {
		return nil
	}

	body, err := migrationFiles.ReadFile("migrations/" + name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}

	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name, checksum, applied_at) VALUES (?, ?, ?)`,
		name, checksum(body), time.Now().Unix()); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}

	return nil
}

func checksum(body []byte) string {
	// Normalise line endings so a CRLF checkout does not look like an edit.
	normalised := strings.ReplaceAll(string(body), "\r\n", "\n")
	sum := sha256.Sum256([]byte(normalised))

	return hex.EncodeToString(sum[:])
}
