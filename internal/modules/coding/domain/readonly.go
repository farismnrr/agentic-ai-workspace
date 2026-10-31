package domain

import "context"

// readOnlyWorkspaceKey is server-controlled; external model input cannot set it.
type readOnlyWorkspaceKey struct{}

// WithReadOnlyWorkspace prevents a sandboxed command from changing the project
// mount. Use for named QA/security checks and commands in external role sessions.
func WithReadOnlyWorkspace(ctx context.Context) context.Context {
	return context.WithValue(ctx, readOnlyWorkspaceKey{}, true)
}

func ReadOnlyWorkspace(ctx context.Context) bool {
	flag, _ := ctx.Value(readOnlyWorkspaceKey{}).(bool)
	return flag
}
