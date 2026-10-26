package bootstrap

import (
	"fmt"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	eghttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/http"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

func newEgressAPIMount(rt *egressRuntime, projects *projectapp.Service, account *accountapp.Service,
	auth, sameOrigin fiber.Handler,
) (httpserver.Mount, error) {
	if rt == nil {
		return func(*fiber.App) {}, nil
	}
	csrf, err := eghttp.NewCSRF()
	if err != nil {
		return nil, fmt.Errorf("initialize egress csrf: %w", err)
	}
	return eghttp.NewHandler(rt.service, projects, account, csrf).Mount(auth, sameOrigin), nil
}
