package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	prefhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/adapter/http"
	prefsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/adapter/sqlite"
	prefapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/aesgcm"
	providerhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/probe"
	providersqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/sqlite"
	providerapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// newModels wires the pages behind Settings: AI models (providers) and persona/rules (preferences).
func newModels(
	db *sql.DB, cfg config.Config, auth, sameOrigin fiber.Handler,
) (*providerapp.Service, []httpserver.Mount, error) {
	cipher, err := aesgcm.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // already descriptive
	}

	providers := providerapp.NewService(providersqlite.NewStore(db), cipher, probe.New(), opaque.Generator{}, time.Now)
	prefs := prefapp.NewService(prefsqlite.NewStore(db), opaque.Generator{}, time.Now)

	return providers, []httpserver.Mount{
		providerhttp.NewHandler(providers).Mount(auth, sameOrigin),
		prefhttp.NewHandler(prefs).Mount(auth, sameOrigin),
	}, nil
}
