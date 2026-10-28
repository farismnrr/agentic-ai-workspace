package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"

	egressruntime "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/runtime"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	taskhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/http"
	tasknotes "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/notefiles"
	taskshots "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/shots"
	tasksqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/sqlite"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

// newAgent wires the agent loop: stores, model clients, project tools, the
// scheduler, and the task API. The returned function stops the workers.
func newAgent(
	db *sql.DB, cfg config.Config, models modelsWiring, projects *projectapp.Service, tools *agentTools,
	gate taskapp.ToolGate, steps taskapp.ActivitySink, hub *taskapp.Hub, auth, sameOrigin fiber.Handler,
	agentSessions taskapp.AgentSessionCleanup,
) (*taskapp.Service, *taskapp.Works, httpserver.Mount, func(), error) {
	store := tasksqlite.NewStoreWithClock(db, opaque.Generator{}, time.Now)

	var taskStore taskapp.TaskStore = store
	if tools.egress != nil {
		taskStore = egressruntime.Tasks(store, tools.egress.TaskEnded)
	}

	deps := taskapp.Observe(taskapp.Deps{
		Notes: tasknotes.New(tools.folders.Dir), Tasks: taskStore, Events: store, Transcript: store, Usage: store,
		LinkedTasks: store,
		Models:      models.agentModels(),
		Windows:     models.providers,
		Efforts:     models.providers,
		Persona:     agentPersona{prefs: models.prefs},
		Projects:    agentProjects{projects: projects},
		Tools:       tools, Gate: gate, IDs: opaque.Generator{}, Now: time.Now, Activity: steps,
		Computer: computer{engine: tools.engine, github: tools.github},
		Team:     agentTeam{team: models.team}, Scopes: store, Delegations: store,
		Shots: taskshots.New(filepath.Join(cfg.DataDir, "shots")), Diffs: tools,
		CLI: tools.plans, Sessions: store,
		AgentSessions: agentSessions,
	}, hub)

	runner := taskapp.NewRunner(deps, taskapp.DefaultLimits())
	scheduler := taskapp.NewScheduler(runner.Run)
	service := taskapp.NewService(deps, scheduler)
	works, handoffs := wireExternalWorkflows(service, store, hub, deps, cfg)
	service.UseWorkIdleAfter(cfg.WorkIdleAfter)

	if err := handoffs.PrepareRecovery(context.Background()); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("prepare plan handoff recovery: %w", err)
	}
	if err := service.Recover(context.Background()); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("recover interrupted tasks: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx, cfg.AgentWorkers)
	handoffDone := make(chan struct{})
	go func() {
		defer close(handoffDone)
		handoffs.Run(ctx, time.Minute)
	}()

	stop := func() {
		cancel()
		scheduler.Stop()
		<-handoffDone
		hub.Close()
	}

	return service, works, taskhttp.NewHandler(service, hub, time.Now).Mount(auth, sameOrigin), stop, nil
}

func wireExternalWorkflows(service *taskapp.Service, store *tasksqlite.Store, hub *taskapp.Hub,
	deps taskapp.Deps, cfg config.Config,
) (*taskapp.Works, *taskapp.Handoffs) {
	works := taskapp.NewWorks(deps, store, store, store, taskdomain.WorkPhaseThresholds{
		IdleAfter: cfg.WorkIdleAfter, AbandonAfter: cfg.WorkAbandonAfter,
	})
	handoffs := taskapp.NewHandoffs(service, store)
	reviews := taskapp.NewPlanReviews(store, opaque.Generator{}, time.Now, handoffs, hub)
	works.UsePlanReviews(reviews)
	service.UsePlanReviewServices(reviews, handoffs, store)
	service.UseWorkServices(works, store)
	service.UseExternalQuestionStore(store)
	return works, handoffs
}
