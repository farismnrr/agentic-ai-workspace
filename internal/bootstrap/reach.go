package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"

	"github.com/gofiber/fiber/v3"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	dbconnsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/sqlite"
	dbconnapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	gatewayhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/adapter/http"
	gatewaysqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/adapter/sqlite"
	gatewayapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/application"
	gatewaydomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	providerapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	providerdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

// reach is what a project's sandbox may reach besides the registries: the database an admin saved in Settings (for
// a project whose connection was made from it) and the model gateway the project's owner chose. Commands, sites and
// the vendors' programs all ask it per run.
type reach struct {
	database *dbconnapp.Tunnels
	gateway  *gatewayapp.Service
}

func (r reach) For(ctx context.Context, projectID string) codingrunner.Granted {
	var granted codingrunner.Granted

	if host, port, ok := r.database.Target(ctx, projectID); ok {
		granted.Targets = append(granted.Targets, net.JoinHostPort(host, strconv.Itoa(port)))
	}

	// Every target gets its own port on loopback, so the database keeps a port both want.
	if target, env, ok := r.gateway.Grant(ctx, projectID); ok && !codingrunner.SamePort(granted.Targets, target) {
		granted.Targets = append(granted.Targets, target)
		granted.Env = env
	}

	return granted
}

// newReach builds the grants and the endpoints that let a project owner choose a gateway.
func newReach(
	db *sql.DB, cfg config.Config, providers *providerapp.Service, projects *projectapp.Service,
	auth, sameOrigin fiber.Handler,
) (codingrunner.Reach, httpserver.Mount, error) {
	cipher, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("open database connection secret key: %w", err)
	}

	settings := dbconnapp.NewSharedSettingsService(dbconnsqlite.NewSharedSettingsStore(db), cipher)
	database := dbconnapp.NewTunnels(dbconnsqlite.NewStore(db), settings)
	gateway := gatewayapp.NewService(gatewaysqlite.NewStore(db), projects, gatewayProviders{providers: providers})

	return reach{database: database, gateway: gateway}, gatewayhttp.NewHandler(gateway).Mount(auth, sameOrigin), nil
}

// gatewayProviders offers a person's saved model providers to the gateway feature: only OpenAI-compatible
// gateways that are enabled and sit on this computer can be chosen by a project.
type gatewayProviders struct{ providers *providerapp.Service }

func (g gatewayProviders) Eligible(ctx context.Context, userID string) ([]gatewaydomain.Choice, error) {
	list, err := g.providers.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	choices := make([]gatewaydomain.Choice, 0, len(list))

	for _, p := range list {
		if _, _, ok := gatewaydomain.Target(p.BaseURL); ok && p.Kind == providerdomain.KindCompatible && p.Enabled {
			choices = append(choices, gatewaydomain.Choice{ID: p.ID, Name: p.Name, BaseURL: p.BaseURL})
		}
	}

	return choices, nil
}

func (g gatewayProviders) Credential(ctx context.Context, userID, id string) (baseURL, apiKey string, err error) {
	credential, err := g.providers.Credential(ctx, userID, id)
	if err != nil {
		return "", "", fmt.Errorf("load provider: %w", err)
	}

	return credential.BaseURL, credential.APIKey, nil
}
