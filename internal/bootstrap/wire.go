package bootstrap

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	spacesqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/sqlite"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
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

	oauth, oauthMount, err := newOAuth(
		db, cfg, registry, sess, directory{account: account}, sameOrigin, cors, oauthLimits())
	if err != nil {
		return nil, nil, err
	}

	engine, closeProxy := newSandboxEngine(cfg)

	spaceStore, permissions := newPermissions(db, registry)

	auth := httpserver.RequireSession(sess)

	models, err := newModels(db, cfg, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	folders, err := openFolders(cfg.DataDir)
	if err != nil {
		return nil, nil, err
	}

	tools, err := newAgentTools(folders, cfg.DataDir, cfg.PublicURL, engine)
	if err != nil {
		return nil, nil, err
	}

	projects, projectMount := newProjects(db, forgetting{Folders: folders, tools: tools}, tools, auth, sameOrigin)

	githubMount, err := connectGitHub(db, cfg, tools, projects, auth, sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	log := newActivityLog(spaceStore, tools)

	tasks, sites, agentMounts, stop, err := newAgentAndSites(
		db, cfg, models, projects, tools, folders, toolGate{permissions: permissions}, agentSteps{log: log}, sess, auth,
		sameOrigin)
	if err != nil {
		return nil, nil, err
	}

	eraseOnDelete(account, sites.StopUser, tasks.DeleteAll, projects.DeleteAll)

	spaceMounts := newSpaceMounts(cfg, projects, tools, tasks, registry, log, permissions, oauth, sess, sameOrigin)

	mounts := joinMounts(
		[]httpserver.Mount{accountMount, oauthMount, projectMount}, agentMounts,
		[]httpserver.Mount{newExport(account, models, projects, tasks, folders, auth)},
		models.mounts, []httpserver.Mount{githubMount}, spaceMounts)

	return mounts, func() { stop(); closeProxy() }, nil
}

// newActivityLog records tool calls and knows how to undo the ones the coding space made.
func newActivityLog(store spaceapp.ActivityStore, tools *agentTools) *spaceapp.ActivityLog {
	undoers := map[string]spaceapp.Undoer{"coding": undoer{tools: tools}}

	return spaceapp.NewActivityLog(store, opaque.Generator{}, undoers, time.Now)
}

// eraseOnDelete has the account remove each feature's data when a person deletes their account.
func eraseOnDelete(account *accountapp.Service, erasers ...eraser) {
	for _, erase := range erasers {
		account.AddEraser(erase)
	}
}

// joinMounts keeps the route groups in registration order.
func joinMounts(groups ...[]httpserver.Mount) []httpserver.Mount {
	var all []httpserver.Mount

	for _, group := range groups {
		all = append(all, group...)
	}

	return all
}

// newSpaceMounts wires the activity log, the app API and each hosted space.
func newSpaceMounts(
	cfg config.Config, projects *projectapp.Service, tools *agentTools, tasks *taskapp.Service,
	registry *spaceapp.Registry, log *spaceapp.ActivityLog, permissions *spaceapp.Permissions, oauth *oauthapp.Service,
	sess sessions, sameOrigin fiber.Handler,
) []httpserver.Mount {
	api := spacehttp.NewAPI(registry, sess, permissions, log)
	mounts := []httpserver.Mount{api.Mount(sameOrigin)}

	space := codingSpace()
	mounter := spacehttp.NewMounter(registry, cfg.Connector(), cfg.Connector(), verifier{oauth: oauth})
	handler := codingHandler(appWorkspaces{projects: projects, tools: tools}, codingmcp.Options{
		Gate: gate{permissions: permissions}, Recorder: recorder{log: log}, Tasks: appThreads{tasks: tasks},
		ResourceMetadataURL: cfg.Connector() + space.MetadataPath(),
	})

	return append(mounts, mounter.Mount(space, handler))
}

// openFolders opens the project folders after deleting the old per-person ones.
func openFolders(dataDir string) (*folder.Folders, error) {
	if err := removeOldWorkspaces(dataDir); err != nil {
		return nil, err
	}

	folders, err := folder.Open(dataDir)
	if err != nil {
		return nil, fmt.Errorf("open project folders: %w", err)
	}

	return folders, nil
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
