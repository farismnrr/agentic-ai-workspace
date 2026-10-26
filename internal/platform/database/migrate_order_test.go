package database

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// Two migrations share the number 051 and both are already applied on real databases. The runner keys its journal
// by the whole file name and runs names in sorted order, so that is safe as long as names stay unique and sorted
// order never changes under an applied database. Never rename or renumber an applied file.
func TestMigrationNamesAreUniqueAndKeepAFixedOrder(t *testing.T) {
	t.Parallel()

	names, err := migrationNames()
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	position := map[string]int{}

	for i, name := range names {
		if seen[name] || !strings.HasSuffix(name, ".sql") {
			t.Fatalf("bad or repeated migration name %q", name)
		}

		seen[name], position[name] = true, i
	}

	first, second := "051_project_gateway.sql", "051_task_archived_at.sql"
	if !seen[first] || !seen[second] || position[first] >= position[second] {
		t.Fatalf("the two 051 files must stay as they are and run in this order: %v", names)
	}

	if position["052_egress_grants.sql"] != position[second]+1 ||
		position["053_egress_audit.sql"] != position["052_egress_grants.sql"]+1 {
		t.Fatalf("052 and 053 must follow both 051 files in order")
	}
}

// A database that already applied everything before 052 must take 052 without any earlier journal row changing.
func TestAnExistingDatabaseTakes052WithoutTouchingTheJournal(t *testing.T) {
	t.Parallel()

	db := rawDatabase(t)
	ctx := context.Background()
	applyBefore(t, db, "052_egress_grants.sql")

	before := journal(t, db)

	if err := migrate(ctx, db); err != nil {
		t.Fatalf("an upgraded database must migrate: %v", err)
	}

	after := journal(t, db)
	for name, sum := range before {
		if after[name] != sum {
			t.Errorf("journal row %s changed", name)
		}
	}

	if after["052_egress_grants.sql"] == "" || after["053_egress_audit.sql"] == "" ||
		len(after) != len(before)+2 {
		t.Fatalf("exactly 052 and 053 should have been added, journal has %d rows (was %d)", len(after), len(before))
	}
}

// applyBefore applies every migration that sorts before stop, the way an older release would have left the database.
func applyBefore(t *testing.T, db *sql.DB, stop string) {
	t.Helper()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, createJournal); err != nil {
		t.Fatal(err)
	}

	names, err := migrationNames()
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range names {
		if name == stop {
			return
		}

		if err = apply(ctx, db, name); err != nil {
			t.Fatal(err)
		}
	}
}

func journal(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), `SELECT name, checksum FROM schema_migrations`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	out := map[string]string{}

	for rows.Next() {
		var name, sum string
		if err = rows.Scan(&name, &sum); err != nil {
			t.Fatal(err)
		}

		out[name] = sum
	}

	return out
}
