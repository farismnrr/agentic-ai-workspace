package bootstrap

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
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

	engine, closeProxy := newSandboxEngine(cfg)

	if err = removeOldWorkspaces(cfg.DataDir); err != nil {
		return nil, nil, err
	}

	spaceStore, permissions := newPermissions(db, registry)

	auth := httpserver.RequireSession(sess)

	models, err := newModels(db, cfg, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	folders, err := folder.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("open project folders: %w", err)
	}

	tools, err := newAgentTools(folders, cfg.DataDir, cfg.PublicURL, engine)
	if err != nil {
		return nil, nil, err
	}

	projects, projectMount := newProjects(db, forgetting{Folders: folders, tools: tools}, tools, auth, sameOrigin)

	tasks, sites, agentMounts, stop, err := newAgentAndSites(
		db, cfg, models, projects, tools, folders, toolGate{permissions: permissions}, sess, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	account.AddEraser(eraser(sites.StopUser))
	account.AddEraser(eraser(tasks.DeleteAll))
	account.AddEraser(eraser(projects.DeleteAll))

	spaceMounts := newSpaceMounts(cfg, projects, tools, registry, spaceStore, permissions, oauth, sess, sameOrigin)

	exportMount := newExport(account, models, projects, tasks, folders, auth)
	mounts := append([]httpserver.Mount{accountMount, oauthMount, projectMount}, agentMounts...)
	mounts = append(mounts, exportMount)
	mounts = append(mounts, models.mounts...)

	return append(mounts, spaceMounts...), func() { stop(); closeProxy() }, nil
}

// newSpaceMounts wires the activity log, the app API and each hosted space.
func newSpaceMounts(
	cfg config.Config, projects *projectapp.Service, tools *agentTools, registry *spaceapp.Registry,
	store *spacesqlite.Store, permissions *spaceapp.Permissions, oauth *oauthapp.Service, sess sessions,
	sameOrigin fiber.Handler,
) []httpserver.Mount {
	undoers := map[string]spaceapp.Undoer{"coding": undoer{tools: tools}}
	log := spaceapp.NewActivityLog(store, opaque.Generator{}, undoers, time.Now)
	api := spacehttp.NewAPI(registry, sess, permissions, log)
	mounts := []httpserver.Mount{api.Mount(sameOrigin)}

	space := codingSpace()
	mounter := spacehttp.NewMounter(registry, cfg.Connector(), cfg.Connector(), verifier{oauth: oauth})
	handler := codingHandler(appWorkspaces{projects: projects, tools: tools}, codingmcp.Options{
		Gate: gate{permissions: permissions}, Recorder: recorder{log: log},
		ResourceMetadataURL: cfg.Connector() + space.MetadataPath(),
	})

	return append(mounts, mounter.Mount(space, handler))
}

// removeOldWorkspaces deletes the per-person folders connected apps used to work in: they work in
// projects now.
func removeOldWorkspaces(dataDir string) error {
	if err := os.RemoveAll(filepath.Join(dataDir, "workspaces")); err != nil {
		return fmt.Errorf("remove old workspaces: %w", err)
	}

	return nil
}

// newPermissions wires the tool switches and approval settings every part of the app obeys.
func newPermissions(db *sql.DB, registry *spaceapp.Registry) (*spacesqlite.Store, *spaceapp.Permissions) {
	store := spacesqlite.NewStore(db)

	return store, spaceapp.NewPermissions(registry, store, time.Now)
}
