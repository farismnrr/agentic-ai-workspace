package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	sitehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/http"
	siteruntime "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/runtime"
	sitesqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/sqlite"
	siteapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// hubNotifier tells a person's open pages that their sites changed; pages then reload them.
type hubNotifier struct{ hub *taskapp.Hub }

func (n hubNotifier) Changed(userID string) { n.hub.Changed(userID, "") }

// uuids mints the random part of site addresses.
type uuids struct{}

func (uuids) UUID() (string, error) { return opaque.Generator{}.UUID() } //nolint:wrapcheck // already worded

// newSites wires sites: the use cases, the sandbox runtime and the HTTP adapter. Archiving a
// project stops its site, and nothing survives a restart, so the table is reset at startup.
func newSites(
	db *sql.DB, cfg config.Config, engine *codingrunner.Engine, folders *folder.Folders,
	projects *projectapp.Service, hub *taskapp.Hub, sess sessions, auth, sameOrigin fiber.Handler,
) (*siteapp.Service, httpserver.Mount, error) {
	base, err := filepath.Abs(filepath.Join(cfg.DataDir, "sites"))
	if err != nil {
		return nil, nil, fmt.Errorf("resolve sites folder: %w", err)
	}

	// Tickets are signed with a key made at startup; sites do not outlive a restart either.
	secret, _, err := opaque.Generator{}.New()
	if err != nil {
		return nil, nil, fmt.Errorf("make ticket key: %w", err)
	}

	service := siteapp.NewService(siteapp.Deps{
		Secret: []byte(secret),
		Store:  sitesqlite.NewStore(db), Runtime: siteruntime.New(engine, folders, base), Projects: projects,
		Notify: hubNotifier{hub: hub}, IDs: uuids{}, Now: time.Now,
		Limits: siteapp.Limits{PerUser: cfg.SiteMaxPerUser, Total: cfg.SiteMaxTotal},
	})

	if err = service.Reconcile(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("reset sites: %w", err)
	}

	projects.OnArchive(service.StopProject)

	return service, sitehttp.NewHandler(service, sess, cfg.ClientIPHeader != "").Mount(auth, sameOrigin), nil
}

// newAgentAndSites wires the two features that need each other: the agent starts sites through
// its tools, and sites tell the person's pages about changes through the agent's hub.
func newAgentAndSites(
	db *sql.DB, cfg config.Config, models modelsWiring, projects *projectapp.Service, tools *agentTools,
	folders *folder.Folders, sess sessions, auth, sameOrigin fiber.Handler,
) (*taskapp.Service, *siteapp.Service, []httpserver.Mount, func(), error) {
	hub := taskapp.NewHub()

	sites, siteMount, err := newSites(db, cfg, tools.engine, folders, projects, hub, sess, auth, sameOrigin)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	tools.sites = sites

	tasks, agentMount, stop, err := newAgent(db, cfg, models, projects, tools, hub, auth, sameOrigin)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return tasks, sites, []httpserver.Mount{siteMount, agentMount}, stop, nil
}
