package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	oauthcimd "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/cimd"
	oauthhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/http"
	oauthjwt "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/jwt"
	oauthsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/sqlite"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

func newOAuth(
	db *sql.DB, cfg config.Config, registry *spaceapp.Registry, sess sessions, users directory,
	sameOrigin, cors fiber.Handler, limits oauthhttp.Limits,
) (*oauthapp.Service, httpserver.Mount, error) {
	signer, err := oauthjwt.LoadOrCreate(cfg.DataDir, cfg.Connector())
	if err != nil {
		return nil, nil, fmt.Errorf("load signing key: %w", err)
	}

	store := oauthsqlite.NewStore(db)
	resources := catalog{registry: registry}
	service := oauthapp.NewService(oauthapp.Deps{
		Issuer: cfg.Connector(), Clients: store, Requests: store, Grants: store, Credentials: store,
		Metadata: oauthcimd.NewFetcher(), Signer: signer, Users: users, Resources: resources,
		Tokens: opaque.Generator{}, Now: time.Now,
	})

	handler := oauthhttp.NewHandler(cfg.Connector(), service, sess, signer, resources.All())

	return service, handler.Mount(sameOrigin, cors, limits), nil
}
