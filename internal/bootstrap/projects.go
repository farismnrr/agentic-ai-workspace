package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projecthttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/http"
	projectsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/sqlite"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

func newProjects(
	db *sql.DB, cfg config.Config, auth, sameOrigin fiber.Handler,
) (*projectapp.Service, *folder.Folders, httpserver.Mount, error) {
	folders, err := folder.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open project folders: %w", err)
	}

	service := projectapp.NewService(projectsqlite.NewStore(db), folders, opaque.Generator{}, time.Now)

	return service, folders, projecthttp.NewHandler(service).Mount(auth, sameOrigin), nil
}
