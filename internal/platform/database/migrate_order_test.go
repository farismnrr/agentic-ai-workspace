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

	last := position[second]
	for _, name := range []string{
		"052_egress_grants.sql", "053_egress_audit.sql", "054_egress_actor_attribution.sql",
	} {
		if position[name] != last+1 {
			t.Fatalf("migration %s must follow the prior migration", name)
		}
		last = position[name]
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
		after["054_egress_actor_attribution.sql"] == "" || len(after) != len(before)+3 {
		t.Fatalf("052 through 054 should have been added, journal has %d rows (was %d)", len(after), len(before))
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

// An installation with 053 already applied must add 054 without rewriting an old checksum.
func TestExistingAuditMigrationGetsAppendOnlyActorAttribution(t *testing.T) {
	t.Parallel()
	db := rawDatabase(t)
	ctx := context.Background()
	applyBefore(t, db, "054_egress_actor_attribution.sql")
	before := journal(t, db)
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	after := journal(t, db)
	if len(after) != len(before)+1 || after["054_egress_actor_attribution.sql"] == "" {
		t.Fatalf("actor migration not appended: before=%d after=%d", len(before), len(after))
	}
	for name, checksum := range before {
		if after[name] != checksum {
			t.Errorf("modified prior migration %s", name)
		}
	}
}
