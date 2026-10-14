package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Pure-Go SQLite driver keeps the binary static (CGO_ENABLED=0).
	_ "modernc.org/sqlite"
)

const (
	fileName   = "workspace.db"
	dirMode    = 0o750
	driverName = "sqlite"
)

// Open opens (creating if needed) the SQLite database under dataDir and
// applies every pending embedded migration.
func Open(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, dirMode); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := "file:" + filepath.Join(dataDir, fileName) +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
