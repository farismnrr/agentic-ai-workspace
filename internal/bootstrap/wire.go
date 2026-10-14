package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	spacesqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/sqlite"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// wire builds every feature and returns the route mounts in registration order.
func wire(db *sql.DB, cfg config.Config) ([]httpserver.Mount, error) {
	sameOrigin := httpserver.SameOrigin(cfg.PublicURL, cfg.AllowedOrigins)
	cors := httpserver.CORS(cfg.CORSOrigins)

	account, accountMount := newAccount(db, cfg, sameOrigin)
	sess := sessions{account: account}
	registry := newRegistry(cfg)

	oauth, oauthMount, err := newOAuth(db, cfg, registry, sess, directory{account: account}, sameOrigin, cors)
	if err != nil {
		return nil, err
	}

	spaceMounts, err := newSpaceMounts(db, cfg, registry, oauth, sess, sameOrigin)
	if err != nil {
		return nil, err
	}

	auth := httpserver.RequireSession(sess)

	_, modelMounts, err := newModels(db, cfg, auth, sameOrigin)
	if err != nil {
		return nil, err
	}

	_, _, projectMount, err := newProjects(db, cfg, auth, sameOrigin)
	if err != nil {
		return nil, err
	}

	mounts := []httpserver.Mount{accountMount, oauthMount, projectMount}
	mounts = append(mounts, modelMounts...)

	return append(mounts, spaceMounts...), nil
}

// newSpaceMounts wires permissions, the activity log, the app API and each hosted space.
func newSpaceMounts(
	db *sql.DB, cfg config.Config, registry *spaceapp.Registry, oauth *oauthapp.Service, sess sessions,
	sameOrigin fiber.Handler,
) ([]httpserver.Mount, error) {
	store := spacesqlite.NewStore(db)
	permissions := spaceapp.NewPermissions(registry, store, time.Now)
	undoers := map[string]spaceapp.Undoer{}

	var services codingmcp.Services

	if cfg.CodingRoot != "" {
		var err error
		if services, err = codingServices(cfg); err != nil {
			return nil, err
		}

		undoers["coding"] = undoer{editor: services.Editor}
	}

	log := spaceapp.NewActivityLog(store, opaque.Generator{}, undoers, time.Now)
	api := spacehttp.NewAPI(registry, sess, permissions, log)
	mounts := []httpserver.Mount{api.Mount(sameOrigin)}

	if cfg.CodingRoot != "" {
		space := codingSpace()
		mounter := spacehttp.NewMounter(registry, cfg.PublicURL, cfg.PublicURL, verifier{oauth: oauth})
		handler := codingHandler(services, codingmcp.Options{
			Gate: gate{permissions: permissions}, Recorder: recorder{log: log},
			ResourceMetadataURL: cfg.PublicURL + space.MetadataPath(),
		})
		mounts = append(mounts, mounter.Mount(space, handler))
	}

	return mounts, nil
}
