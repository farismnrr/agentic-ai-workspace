package database

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func rows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func run(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func TestSweepRemovesOnlyWhatCanNeverMatterAgain(t *testing.T) {
	t.Parallel()

	db := openAt(t, t.TempDir())
	now := time.Now()
	old, fresh := now.Add(-48*time.Hour).Unix(), now.Add(time.Hour).Unix()
	veryOld := now.Add(-90 * 24 * time.Hour).Unix()

	run(t, db, `INSERT INTO users (id, name, email, password_hash, created_at) VALUES ('u', 'U', 'u@x.io', 'x', 0)`)
	run(t, db, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('a', 'u', ?), ('b', 'u', ?)`, old, fresh)
	run(t, db, `INSERT INTO oauth_clients (id, name, redirect_uris, source, created_at)
		VALUES ('stale', 'S', '[]', 'dcr', ?), ('used', 'U', '[]', 'dcr', ?), ('recent', 'R', '[]', 'dcr', ?)`,
		veryOld, veryOld, now.Unix())
	run(t, db, `INSERT INTO oauth_grants
		(id, user_id, client_id, client_name, redirect_uri, resource, scope, created_at, revoked_at)
		VALUES ('live', 'u', 'used', 'U', 'r', 'res', 's', ?, NULL), ('gone', 'u', 'used', 'U', 'r', 'res', 's', ?, ?)`,
		veryOld, veryOld, veryOld)
	run(t, db, `INSERT INTO oauth_tokens (hash, grant_id, kind, expires_at)
		VALUES ('t1', 'live', 'access', ?), ('t2', 'live', 'access', ?)`, old, fresh)

	if err := Sweep(context.Background(), db, now); err != nil {
		t.Fatal(err)
	}

	if got := rows(t, db, "sessions"); got != 1 {
		t.Errorf("only the live session stays, got %d", got)
	}

	if got := rows(t, db, "oauth_tokens"); got != 1 {
		t.Errorf("only the unexpired token stays, got %d", got)
	}

	if got := rows(t, db, "oauth_grants"); got != 1 {
		t.Errorf("a grant revoked long ago goes, a live one stays, got %d", got)
	}

	if got := rows(t, db, "oauth_clients"); got != 2 {
		t.Errorf("only the stale client without grants goes, got %d", got)
	}
}
