package bootstrap

import (
	"context"
	"fmt"
	"log"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	eghttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/http"
	egdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

type egressThread struct{ tasks *taskapp.Service }

func (n egressThread) Notice(ctx context.Context, grant egdomain.Grant, action string) {
	text := fmt.Sprintf("Internet access request %s for %s (HTTPS only). Scope: this task; expires %s UTC.",
		action, grant.Host, grant.ExpiresAt.UTC().Format("2006-01-02 15:04"))
	if err := n.tasks.NoteEgress(ctx, grant.TaskID, text); err != nil {
		log.Printf("egress grant: task notification unavailable")
	}
}

func newEgressAPIMount(rt *egressRuntime, projects *projectapp.Service, account *accountapp.Service,
	tasks *taskapp.Service, auth, sameOrigin fiber.Handler,
) (httpserver.Mount, error) {
	if rt == nil {
		return func(*fiber.App) {}, nil
	}
	csrf, err := eghttp.NewCSRF()
	if err != nil {
		return nil, fmt.Errorf("initialize egress csrf: %w", err)
	}
	handler := eghttp.NewHandler(rt.service, projects, account, csrf).WithNotifier(egressThread{tasks})
	return handler.Mount(auth, sameOrigin), nil
}
