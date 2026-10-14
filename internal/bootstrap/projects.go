package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projecthttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/http"
	projectsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/sqlite"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

func newProjects(
	db *sql.DB, folders *folder.Folders, tools *agentTools, auth, sameOrigin fiber.Handler,
) (*projectapp.Service, httpserver.Mount) {
	service := projectapp.NewService(
		projectsqlite.NewStore(db), folders, projectChecks{tools: tools}, opaque.Generator{}, time.Now)

	return service, projecthttp.NewHandler(service).Mount(auth, sameOrigin)
}
