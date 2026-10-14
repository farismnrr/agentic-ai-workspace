// Package sandbox runs a command in its own Linux namespaces with a private root
// filesystem, the same isolation primitives containers use, without a container
// runtime. The command sees a read-only toolbox (its own copy of Node.js, Go, Rust,
// Git and friends, never the host's programs), a private /tmp, and exactly one
// writable folder (/workspace). It has no network, cannot see other processes,
// holds no privileges, is confined by a system call filter, and, when cgroups are
// delegated to the server, is held to real CPU, memory and process limits.
package sandbox

import "errors"

var (
	// ErrUnsupported means this platform or kernel cannot create the sandbox.
	ErrUnsupported = errors.New("sandbox is not available: this needs Linux with unprivileged user namespaces")

	// ErrNoToolbox means the programs a task needs have not been installed yet.
	ErrNoToolbox = errors.New(
		"the sandbox toolbox is not installed; the server's owner must run: agentic-ai-workspace sandbox install")
)

// initFlag marks a re-executed copy of this binary as the sandbox's init process.
const initFlag = "--sandbox-init"

// Workspace is where the writable folder appears inside the sandbox.
const Workspace = "/workspace"

// Path is the only PATH a sandboxed command gets.
const Path = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

// Spec describes one sandboxed command.
type Spec struct {
	// Group holds the command under cgroup limits; nil means rlimits only.
	Group *Group
	// Toolbox is the read-only root holding the programs (an absolute path).
	Toolbox string
	// Workspace is the one writable folder, shown at /workspace (an absolute path).
	Workspace string
	Argv      []string
	Env       []string
	Limits    Limits
}
