package bootstrap

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v3"

	dbconnhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/http"
	dbconnprobe "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/probe"
	dbconnprovision "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provision"
	dbconnsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/sqlite"
	dbconnapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	envapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

func newDBConnMount(
	db *sql.DB, cfg config.Config, projects dbconnapp.ProjectOwner, env *envapp.Service,
	auth, sameOrigin fiber.Handler,
) (httpserver.Mount, error) {
	cipher, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open database connection secret key: %w", err)
	}
	var provisioners []dbconnapp.Provisioner
	if cfg.PostgresAdminURL != "" {
		provisioners = append(provisioners, dbconnprovision.New(cfg.PostgresAdminURL, cfg.Port))
	}
	service := dbconnapp.NewService(dbconnsqlite.NewStore(db), cipher, projects, env,
		dbconnprobe.New(cfg.Port), provisioners...)
	return dbconnhttp.NewHandler(service).Mount(auth, sameOrigin, databaseTestLimit(), databaseSharedLimit()), nil
}

var (
	_ dbconnapp.ProjectOwner = (*projectapp.Service)(nil)
	_ dbconnapp.Environment  = (*envapp.Service)(nil)
	_ dbconnapp.Provisioner  = (*dbconnprovision.Provisioner)(nil)
)
