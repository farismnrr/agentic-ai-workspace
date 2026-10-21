package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	envhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/env/adapter/http"
	envsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/env/adapter/sqlite"
	envapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

func newEnvMount(
	db *sql.DB, cfg config.Config, projects envapp.ProjectOwner, auth, sameOrigin fiber.Handler,
) (*envapp.Service, httpserver.Mount, error) {
	cipher, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // already descriptive
	}

	service := envapp.NewService(envsqlite.NewStore(db), cipher, projects, time.Now)
	return service, envhttp.NewHandler(service).Mount(auth, sameOrigin), nil
}

func newProjectSettings(
	db *sql.DB, cfg config.Config, projects *projectapp.Service, auth, sameOrigin fiber.Handler,
) (*envapp.Service, []httpserver.Mount, error) {
	env, envMount, err := newEnvMount(db, cfg, projects, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}
	dbMount, err := newDBConnMount(db, cfg, projects, env, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}
	return env, []httpserver.Mount{envMount, dbMount}, nil
}

var _ envapp.ProjectOwner = (*projectapp.Service)(nil)
