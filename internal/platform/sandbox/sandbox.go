// Package sandbox runs a command in a throwaway Docker container. The container
// sees only the image's programs, a private /tmp, and exactly one writable folder
// (/workspace). It has no network, no capabilities, a read-only root filesystem,
// and is held to memory, CPU and process limits by the Docker daemon.
package sandbox

import "errors"

// ErrUnavailable means the Docker daemon could not be reached or used.
var ErrUnavailable = errors.New("docker is not available")

// Workspace is where the writable folder appears inside the container.
const Workspace = "/workspace"

// Path is the only PATH a sandboxed command gets.
const Path = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

// Spec describes one sandboxed command.
type Spec struct {
	// Workspace is the one writable folder, shown at /workspace (an absolute path).
	Workspace string
	Argv      []string
	Env       []string
}

// Result is how a sandboxed command ended.
type Result struct {
	ExitCode  int
	OOMKilled bool
}
