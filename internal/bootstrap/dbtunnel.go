package bootstrap

import (
	"database/sql"
	"fmt"

	dbconnsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/sqlite"
	dbconnapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

// newDBTunnels decides which database a project's sandbox may reach: the one server an admin saved in
// Settings, for a project whose own connection was made from it. Commands and sites ask it per run.
func newDBTunnels(db *sql.DB, cfg config.Config) (*dbconnapp.Tunnels, error) {
	cipher, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open database connection secret key: %w", err)
	}

	settings := dbconnapp.NewSharedSettingsService(dbconnsqlite.NewSharedSettingsStore(db), cipher)

	return dbconnapp.NewTunnels(dbconnsqlite.NewStore(db), settings), nil
}
