package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	prefhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/adapter/http"
	prefsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/adapter/sqlite"
	prefapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/application"
	providerhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/probe"
	providersqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/adapter/sqlite"
	providerapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

// newModels wires the pages behind Settings: AI models (providers) and persona/rules (preferences).
// modelsWiring is what the agent needs from the settings pages.
type modelsWiring struct {
	ownPort   string
	providers *providerapp.Service
	prefs     *prefapp.Service
	mounts    []httpserver.Mount
}

func newModels(db *sql.DB, cfg config.Config, auth, sameOrigin fiber.Handler) (modelsWiring, error) {
	cipher, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return modelsWiring{}, err //nolint:wrapcheck // already descriptive
	}

	checker := probe.New(hostedProviders, gateways(cfg.Port))
	providers := providerapp.NewService(providersqlite.NewStore(db), cipher, checker, opaque.Generator{}, time.Now)
	prefs := prefapp.NewService(prefsqlite.NewStore(db), opaque.Generator{}, time.Now)

	return modelsWiring{ownPort: cfg.Port, providers: providers, prefs: prefs, mounts: []httpserver.Mount{
		providerhttp.NewHandler(providers).Mount(auth, sameOrigin),
		prefhttp.NewHandler(prefs).Mount(auth, sameOrigin),
	}}, nil
}

// agentModels hands the agent the models a person connected.
func (m modelsWiring) agentModels() agentModels {
	return agentModels{providers: m.providers, prefs: m.prefs, ownPort: m.ownPort}
}
