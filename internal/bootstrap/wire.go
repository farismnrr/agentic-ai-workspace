package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	spacesqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/sqlite"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// wire builds every feature and returns the route mounts in registration order.
func wire(db *sql.DB, cfg config.Config) ([]httpserver.Mount, func(), error) {
	sameOrigin := httpserver.SameOrigin(cfg.PublicURL, cfg.AllowedOrigins)
	cors := httpserver.CORS(cfg.CORSOrigins)

	account, accountMount := newAccount(db, cfg, sameOrigin, accountLimits())
	sess := sessions{account: account}
	registry := newRegistry(cfg)

	users := directory{account: account}

	oauth, oauthMount, err := newOAuth(db, cfg, registry, sess, users, sameOrigin, cors, oauthLimits())
	if err != nil {
		return nil, nil, err
	}

	spaceMounts, err := newSpaceMounts(db, cfg, registry, oauth, sess, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	auth := httpserver.RequireSession(sess)

	models, err := newModels(db, cfg, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	folders, err := folder.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("open project folders: %w", err)
	}

	tools, err := newAgentTools(folders, cfg.DataDir)
	if err != nil {
		return nil, nil, err
	}

	projects, projectMount := newProjects(db, folders, tools, auth, sameOrigin)

	tasks, agentMount, stop, err := newAgent(db, cfg, models, projects, tools, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	_, chatMount := newChat(db, models, projects, tasks, auth, sameOrigin)

	account.AddEraser(eraser(tasks.DeleteAll))
	account.AddEraser(eraser(projects.DeleteAll))

	exportMount := newExport(account, models, projects, tasks, folders, auth)
	mounts := []httpserver.Mount{accountMount, oauthMount, projectMount, agentMount, chatMount, exportMount}
	mounts = append(mounts, models.mounts...)

	return append(mounts, spaceMounts...), stop, nil
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
