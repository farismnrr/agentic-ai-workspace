package runner

import (
	"context"
	"errors"

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

// ErrTaskNotOpen means a run carries a task scope that cannot be confirmed (the task is over, belongs to another
// project, or the check failed), so it gets no network at all.
var ErrTaskNotOpen = errors.New("the task is not open, so it gets no network")

// withoutTask removes any task scope from ctx. Work that is not a task's own command, such as a site, uses it.
func withoutTask(ctx context.Context) context.Context {
	return context.WithValue(ctx, scopeKey{}, nil)
}

// socket returns the folder of the proxy socket a command may be given. Without grants enabled it is the workspace's
// socket, exactly as before. With grants enabled, a command that carries a task scope gets that task's own socket,
// and only if the task is live; a scope that cannot be confirmed gets ErrTaskNotOpen and no socket at all (no
// network), never the workspace's socket in its place.
func (e *Engine) socket(ctx context.Context, id string, hosts []string) (string, error) {
	hub := e.extras.Proxy
	scope, scoped := ctx.Value(scopeKey{}).(taskScope)

	if !hub.Enabled() || !scoped {
		return hub.Dir(ctx, id, hosts...) //nolint:wrapcheck // worded by the caller
	}

	if scope.bad || e.extras.Scopes == nil {
		return "", ErrTaskNotOpen
	}

	live, err := e.extras.Scopes.TaskLive(ctx, scope.project, scope.task)
	if err != nil || !live {
		return "", ErrTaskNotOpen
	}

	task := egressproxy.Scope{Project: scope.project, Task: scope.task}

	return hub.TaskDir(ctx, id, task, hosts...) //nolint:wrapcheck // worded by the caller
}
