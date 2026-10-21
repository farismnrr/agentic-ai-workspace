package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/argon2"
	accounthttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/http"
	accountsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/sqlite"
	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

func newAccount(
	db *sql.DB, cfg config.Config, sameOrigin fiber.Handler, limits accounthttp.Limits,
) (*accountapp.Service, httpserver.Mount) {
	store := accountsqlite.NewStore(db)
	service := accountapp.NewService(store, store, argon2.Hasher{}, opaque.Generator{}, time.Now, cfg.AdminEmail)

	return service, accounthttp.NewHandler(service, cfg.CookieSecure).Mount(sameOrigin, limits)
}
