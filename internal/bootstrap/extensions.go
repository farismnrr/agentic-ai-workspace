package bootstrap

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"

	extensionarchive "github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/adapter/archive"
	extensionhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/adapter/http"
	extensionsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/adapter/sqlite"
	extensionapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// newExtensions wires the browser extensions the AI builds: the packages sit in the data folder, and the person's
// pages are told when one is deployed or removed. Archiving a project removes its extension.
func newExtensions(
	db *sql.DB, cfg config.Config, folders *folder.Folders, projects *projectapp.Service, hub *taskapp.Hub,
	auth, sameOrigin fiber.Handler,
) (*extensionapp.Service, httpserver.Mount, error) {
	base, err := filepath.Abs(filepath.Join(cfg.DataDir, "extensions"))
	if err != nil {
		return nil, nil, fmt.Errorf("resolve extensions folder: %w", err)
	}

	service := extensionapp.NewService(extensionapp.Deps{
		Store: extensionsqlite.NewStore(db), Packer: extensionarchive.New(folders, base), Projects: projects,
		Notify: hubNotifier{hub: hub}, Now: time.Now,
	})

	projects.OnArchive(service.Forget)

	return service, extensionhttp.NewHandler(service).Mount(auth, sameOrigin), nil
}
