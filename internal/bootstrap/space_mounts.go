package bootstrap

import (
	"time"

	"github.com/gofiber/fiber/v3"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	orchestratormcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/mcp"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

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
	orchestrator := orchestratorSpace()
	orchestratorHandler := orchestratormcp.HTTPHandler(tasks, orchestratormcp.Options{
		Gate: orchestratorGate{permissions: permissions}, Activity: orchestratorActivity{log: log},
		ResourceMetadataURL: cfg.Connector() + orchestrator.MetadataPath(), TaskURLBase: cfg.PublicURL,
	})
	orchestratorLimit := orchestratorRateLimit(
		verifier{oauth: oauth}, orchestrator.ResourceURI(cfg.Connector()),
	)

	return append(mounts, mounter.Mount(space, handler), mounter.Mount(
		orchestrator, orchestratorHandler, orchestratorLimit,
		httpserver.Limit(orchestratorPerMinute, time.Minute, orchestratorUserKey, false),
	))
}
