package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

func TestSharedDatabaseSettingsMigrationCarriesPostgresRowForward(t *testing.T) {
	t.Parallel()
	db := rawDatabase(t)
	applyLegacyMigrationPrefix(t, db)
	insertLegacyPostgresSettings(t, db)
	applyNewSettingsMigration(t, db)
	assertPostgresSettingsCarriedForward(t, db)
	assertLegacySettingsRemain(t, db)
}

func applyLegacyMigrationPrefix(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), createJournal); err != nil {
		t.Fatal(err)
	}
	names, err := migrationNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name == "050_shared_database_kinds.sql" {
			return
		}
		if err := apply(context.Background(), db, name); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
}

func insertLegacyPostgresSettings(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO shared_database_settings
		(id, host, port, database_name, user_name, password_sealed, ssl_mode,
		 ca_cert_sealed, client_cert_sealed, client_key_sealed, updated_at)
		VALUES (1, 'db.example', 5432, 'workspace', 'admin', 'sealed-pw',
		 'verify-full', 'sealed-ca', '', '', 42)`)
	if err != nil {
		t.Fatalf("insert legacy settings: %v", err)
	}
}

func applyNewSettingsMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := apply(context.Background(), db, "050_shared_database_kinds.sql"); err != nil {
		t.Fatal(err)
	}
}

func assertPostgresSettingsCarriedForward(t *testing.T, db *sql.DB) {
	t.Helper()
	var host, password, sslMode, tlsMode, ca string
	var port int
	err := db.QueryRowContext(context.Background(), `SELECT host, port, password_sealed,
		ssl_mode, tls, ca_cert_sealed FROM shared_database_servers WHERE kind = 'postgres'`).
		Scan(&host, &port, &password, &sslMode, &tlsMode, &ca)
	if err != nil {
		t.Fatal(err)
	}
	if host != "db.example" || port != 5432 || password != "sealed-pw" || sslMode != "verify-full" ||
		tlsMode != "ca" || ca != "sealed-ca" {
		t.Fatalf("carried settings = %q %d %q %q %q %q", host, port, password, sslMode, tlsMode, ca)
	}
}

func assertLegacySettingsRemain(t *testing.T, db *sql.DB) {
	t.Helper()
	var oldRows int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM shared_database_settings`).Scan(&oldRows); err != nil {
		t.Fatal(err)
	}
	if oldRows != 1 {
		t.Fatalf("legacy settings rows = %d, want 1", oldRows)
	}
}

func TestSharedDatabaseSettingsMigrationMapsLegacyTLS(t *testing.T) {
	for _, test := range []struct{ ssl, cert, want string }{
		{ssl: "disable", want: "none"},
		{ssl: "verify-full", want: "ca"},
		{ssl: "verify-full", cert: "sealed-cert", want: "full"},
	} {
		db := rawDatabase(t)
		if _, err := db.ExecContext(context.Background(), `CREATE TABLE shared_database_settings (
			id INTEGER PRIMARY KEY, host TEXT, port INTEGER, database_name TEXT, user_name TEXT,
			password_sealed TEXT, ssl_mode TEXT, ca_cert_sealed TEXT DEFAULT '',
			client_cert_sealed TEXT DEFAULT '', client_key_sealed TEXT DEFAULT '', updated_at INTEGER)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(context.Background(), `INSERT INTO shared_database_settings VALUES
			(1, 'db', 5432, 'app', 'user', 'sealed-pw', ?, '', ?, '', 1)`, test.ssl, test.cert); err != nil {
			t.Fatal(err)
		}
		if err := applyMigrationFile(t, db, "050_shared_database_kinds.sql"); err != nil {
			t.Fatal(err)
		}
		var got string
		if err := db.QueryRowContext(context.Background(),
			`SELECT tls FROM shared_database_servers WHERE kind='postgres'`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("legacy ssl_mode=%s cert=%t mapped to %q, want %q", test.ssl, test.cert != "", got, test.want)
		}
		_ = db.Close()
	}
}

func applyMigrationFile(t *testing.T, db *sql.DB, name string) error {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), createJournal); err != nil {
		return fmt.Errorf("create migration journal: %w", err)
	}
	return apply(context.Background(), db, name)
}
