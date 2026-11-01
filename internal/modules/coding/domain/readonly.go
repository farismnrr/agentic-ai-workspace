package domain

import "context"

// readOnlyWorkspaceKey is server-controlled; external model input cannot set it.
type readOnlyWorkspaceKey struct{}
type gitOnlyKey struct{}
type vcsRoleKey struct{}

// WithVCSRole records the authenticated teammate role for operation checks.
// It is set by the server, never copied from a tool's JSON input.
func WithVCSRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, vcsRoleKey{}, role)
}

func VCSRole(ctx context.Context) string {
	role, _ := ctx.Value(vcsRoleKey{}).(string)
	return role
}

// WithGitOnly runs a fixed Git argv with Git credentials, but no project
// environment secrets. It does not unlock arbitrary shell execution.
func WithGitOnly(ctx context.Context) context.Context {
	return context.WithValue(ctx, gitOnlyKey{}, true)
}

func GitOnly(ctx context.Context) bool {
	flag, _ := ctx.Value(gitOnlyKey{}).(bool)
	return flag
}

// WithReadOnlyWorkspace prevents a sandboxed command from changing the project
// mount. Use for named QA/security checks and commands in external role sessions.
func WithReadOnlyWorkspace(ctx context.Context) context.Context {
	return context.WithValue(ctx, readOnlyWorkspaceKey{}, true)
}

func ReadOnlyWorkspace(ctx context.Context) bool {
	flag, _ := ctx.Value(readOnlyWorkspaceKey{}).(bool)
	return flag
}
