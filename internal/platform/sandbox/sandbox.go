// Package sandbox runs a command in a throwaway Docker container. The container
// sees only the image's programs, a private /tmp, and exactly one writable folder
// (/workspace). It has no network, no capabilities, a read-only root filesystem,
// and is held to memory, CPU and process limits by the Docker daemon.
package sandbox

import (
	"errors"
	"io"
)

// ErrUnavailable means the Docker daemon could not be reached or used.
var ErrUnavailable = errors.New("docker is not available")

// Workspace is where the writable folder appears inside the container.
const Workspace = "/workspace"

// Cache is where the person's package caches appear; unlike the root filesystem it survives between commands.
const Cache = "/cache"

// Egress is where the folder holding the proxy socket appears.
const Egress = "/run/egress"

// SiteDir is where the folder holding a site's socket appears. Inside, the start script
// publishes the app's local port as a Unix socket that only the server can reach.
const SiteDir = "/run/site"

// BridgeDir is where the folder holding the socket of tools the server runs for a program appears.
const BridgeDir = "/run/bridge"

// Home is where a person's own folder for a vendor CLI (its login and sessions) appears.
const Home = "/home/cli"

// Entry is the image's start script. It makes the proxy socket reachable as a local port, then runs the command.
const Entry = "/usr/local/bin/sandbox-entry"

// Path is the only PATH a sandboxed command gets: the image's toolchains, then the system.
const Path = "/usr/local/go/bin:/opt/cargo/bin:/opt/java/openjdk/bin:/opt/maven/bin:" +
	"/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

// Spec describes one sandboxed command.
type Spec struct {
	// Workspace is the one writable folder, shown at /workspace (an absolute path).
	Workspace string
	// Cache is an optional second writable folder for package downloads, shown at /cache.
	Cache string
	// Egress is an optional folder holding the proxy socket, shown read-only at /run/egress.
	Egress string
	// Site is an optional writable folder for a site's socket, shown at /run/site.
	Site string
	// Home is an optional writable folder shown at /home/cli, where a vendor CLI keeps its login.
	Home string
	// Bridge is an optional folder holding the socket of tools the server runs, shown at /run/bridge.
	Bridge string
	// Stdin, when set, is copied to the command's standard input and then closed.
	Stdin io.Reader
	Argv  []string
	Env   []string
	// WorkspaceReadOnly shows the workspace so it can be read and not changed.
	WorkspaceReadOnly bool
}

// Result is how a sandboxed command ended.
type Result struct {
	ExitCode  int
	OOMKilled bool
}
