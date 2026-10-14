package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	taskhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/http"
	tasksqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/sqlite"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// newAgent wires the agent loop: stores, model clients, project tools, the
// scheduler, and the task API. The returned function stops the workers.
func newAgent(
	db *sql.DB, cfg config.Config, models modelsWiring, projects *projectapp.Service, tools *agentTools,
	auth, sameOrigin fiber.Handler,
) (*taskapp.Service, httpserver.Mount, func(), error) {
	store := tasksqlite.NewStore(db, opaque.Generator{})
	hub := taskapp.NewHub()
	deps := taskapp.Observe(taskapp.Deps{
		Tasks: store, Events: store, Transcript: store, Usage: store,
		Models:   models.agentModels(),
		Persona:  agentPersona{prefs: models.prefs},
		Projects: agentProjects{projects: projects},
		Tools:    tools, IDs: opaque.Generator{}, Now: time.Now,
	}, hub)

	runner := taskapp.NewRunner(deps, taskapp.DefaultLimits())
	scheduler := taskapp.NewScheduler(runner.Run)
	service := taskapp.NewService(deps, scheduler)

	if err := service.Recover(context.Background()); err != nil {
		return nil, nil, nil, fmt.Errorf("recover interrupted tasks: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx, cfg.AgentWorkers)

	stop := func() {
		cancel()
		scheduler.Stop()
		hub.Close()
	}

	return service, taskhttp.NewHandler(service, hub, time.Now).Mount(auth, sameOrigin), stop, nil
}
