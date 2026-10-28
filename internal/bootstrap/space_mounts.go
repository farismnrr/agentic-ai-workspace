package bootstrap

import (
	"github.com/gofiber/fiber/v3"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// newSpaceMounts wires the activity log, the app API and each hosted space.
func newSpaceMounts(
	cfg config.Config, projects *projectapp.Service, tools *agentTools, tasks *taskapp.Service, works *taskapp.Works,
	registry *spaceapp.Registry, log *spaceapp.ActivityLog, permissions *spaceapp.Permissions, oauth *oauthapp.Service,
	sess sessions, sameOrigin fiber.Handler,
) []httpserver.Mount {
	api := spacehttp.NewAPI(registry, sess, permissions, log)
	mounts := []httpserver.Mount{api.Mount(sameOrigin)}

	space := codingSpace()
	mounter := spacehttp.NewMounter(registry, cfg.Connector(), cfg.Connector(), verifier{oauth: oauth})
	handler := codingHandler(appWorkspaces{projects: projects, tools: tools}, codingmcp.Options{
		Gate: gate{permissions: permissions}, Recorder: NewRecorder(log), Tasks: appThreads{tasks: tasks},
		Works: appWorks{works: works}, WorkCalls: appWorkCalls{works: works},
		ResourceMetadataURL: cfg.Connector() + space.MetadataPath(),
	})

	return append(mounts, mounter.Mount(space, handler))
}
