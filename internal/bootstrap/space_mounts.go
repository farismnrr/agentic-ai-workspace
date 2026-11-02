package bootstrap

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	orchestratorfiles "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/adapter/files"
	orchestratormcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/adapter/mcp"
	orchestratordomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"
	oauthdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
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

	orchestratorSpace := orchestratordomain.Space()
	orchestratorHandler := newOrchestratorHandler(orchestratormcp.Services{
		Projects: projects, Tasks: tasks, Permissions: permissions,
		Files: orchestratorFiles{workspaces: ownWorkspaces{projects: projects, tools: tools}, tasks: tasks},
		Uploads: &orchestratorfiles.Manager{
			Projects: projects, Folders: tools.folders, Notify: tasks.NotifyExternalFileChanged,
		},
		MetadataURL: cfg.Connector() + orchestratorSpace.MetadataPath(),
	})
	mounts = append(mounts, mounter.Mount(space, handler))
	return append(mounts, mounter.MountWithMiddleware(
		orchestratorSpace, orchestratorHandler, orchestratorIngress,
	))
}

func orchestratorIngress(next http.Handler) http.Handler {
	recommendedScopes := []string{
		oauthdomain.ScopeOpenID,
		oauthdomain.ScopeEmail,
		orchestratordomain.ScopeRead,
		orchestratordomain.ScopeTasks,
		orchestratordomain.ScopeDelegate,
		orchestratordomain.ScopeFiles,
		orchestratordomain.ScopeApprove,
	}
	return measureOrchestratorIngress(spacehttp.WithInitialScopeHint(next, recommendedScopes))
}

func measureOrchestratorIngress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		outcome := "returned"
		defer func() {
			slog.Info("orchestrator mcp ingress", "stage", "http_ingress",
				"outcome", outcome, "total_ms", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(w, r)
	})
}
