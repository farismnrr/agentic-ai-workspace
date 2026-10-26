package runner

import (
	"context"
	"log"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

type scopeKey struct{}

// taskScope is whom a command runs for. The server sets it from the task it is executing (never from tool input),
// and it only matters when the proxy hub has grants enabled.
type taskScope struct {
	project, task string
	bad           bool
}

// WithTask marks ctx as a command of one task of one project. Only the server's own task loop calls it.
func WithTask(ctx context.Context, projectID, taskID string) context.Context {
	return context.WithValue(ctx, scopeKey{}, taskScope{project: projectID, task: taskID})
}

// ScopeCheck says whether a task exists in a project and is still open.
type ScopeCheck interface {
	TaskLive(ctx context.Context, projectID, taskID string) (bool, error)
}

// bind drops trust in a scope that does not belong to this runner's project: the command then gets no socket.
func (r *Runner) bind(ctx context.Context) context.Context {
	scope, ok := ctx.Value(scopeKey{}).(taskScope)
	if !ok || (scope.project != "" && scope.task != "" && scope.project == r.projectID) {
		return ctx
	}

	scope.bad = true

	return context.WithValue(ctx, scopeKey{}, scope)
}

// socket returns the folder of the proxy socket a command may be given. Without grants enabled it is the workspace's
// socket, exactly as before. With grants enabled, a command that carries a task scope gets that task's own socket,
// and only if the task is live; a scope that cannot be confirmed gets no socket at all (no network), never the
// workspace's socket in its place.
func (e *Engine) socket(ctx context.Context, id string, hosts []string) (string, bool) {
	hub := e.extras.Proxy
	scope, scoped := ctx.Value(scopeKey{}).(taskScope)

	if !hub.Enabled() || !scoped {
		dir, err := hub.Dir(ctx, id, hosts...)

		return dir, err == nil
	}

	if scope.bad || e.extras.Scopes == nil {
		return "", false
	}

	live, err := e.extras.Scopes.TaskLive(ctx, scope.project, scope.task)
	if err != nil || !live {
		return "", false
	}

	dir, err := hub.TaskDir(ctx, id, egressproxy.Scope{Project: scope.project, Task: scope.task}, hosts...)
	if err != nil {
		log.Printf("egress: no task socket: %v", err)

		return "", false
	}

	return dir, true
}
