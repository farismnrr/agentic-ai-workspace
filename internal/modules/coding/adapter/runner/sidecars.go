package runner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const (
	cacheMode   = 0o750
	dirWritable = 0o700
	// DefaultCacheBytes is how large one workspace's package cache may grow before it is emptied.
	DefaultCacheBytes = 5 << 30
	proxyAddress      = "http://127.0.0.1:3128"
)

// Extras are the optional folders a command gets next to its workspace: a package
// cache that survives between commands and the socket of the registry-only proxy.
// Both belong to one workspace, so one person's downloads and traffic never mix with another's.
type Extras struct {
	// Proxy serves the socket folders; nil means the command has no network at all.
	Proxy *egressproxy.Hub
	// CacheDir holds one cache folder per workspace; empty means no cache.
	CacheDir string
}

// prepare creates (and when needed empties) the workspace's cache and starts its proxy socket.
// It never fails the command: without the extras the command still runs, just offline and uncached.
// Images without the toolbox's start script (any plain image) get no proxy socket.
func (e *Engine) prepare(ctx context.Context, dir string, spec *sandbox.Spec, docker *sandbox.Docker) {
	id := filepath.Base(dir)

	if e.extras.CacheDir != "" {
		cache := filepath.Join(e.extras.CacheDir, id)
		if sizeOver(cache, DefaultCacheBytes) {
			wipe(cache)
		}

		if os.MkdirAll(cache, cacheMode) == nil {
			spec.Cache = cache
		}
	}

	if e.extras.Proxy != nil && len(docker.Toolchains()) > 0 {
		if socket, err := e.extras.Proxy.Dir(ctx, id); err == nil {
			spec.Egress = socket
		}
	}
}

// wipe deletes a cache folder. Some tools (Go's module cache by default) make their files
// read-only, which RemoveAll cannot delete, so folders are made writable first. The cache is
// written by untrusted commands, so every chmod goes through os.Root and can never follow a
// planted symlink out of the folder.
func wipe(dir string) {
	if root, err := os.OpenRoot(dir); err == nil {
		_ = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				_ = root.Chmod(path, dirWritable)
			}

			return nil
		})

		_ = root.Close()
	}

	_ = os.RemoveAll(dir)
}

// Forget removes what a workspace left behind outside its own folder.
func (e *Engine) Forget(dir string) {
	id := filepath.Base(dir)

	if e.extras.CacheDir != "" {
		wipe(filepath.Join(e.extras.CacheDir, id))
	}

	if e.extras.Proxy != nil {
		e.extras.Proxy.Remove(id)
	}
}

// environment is fixed: server settings and the host's home never reach commands.
// Package managers keep their downloads in /cache and reach registries through the proxy.
func environment(spec sandbox.Spec) []string {
	env := []string{
		"LANG=C.UTF-8", "CI=1", "PATH=" + sandbox.Path, "HOME=/tmp", "TMPDIR=/tmp",
		"RUSTUP_HOME=/opt/rustup", "JAVA_HOME=/opt/java/openjdk", "GOFLAGS=-modcacherw -buildvcs=false",
	}

	if spec.Cache != "" {
		c := sandbox.Cache
		env = append(env, "GOMODCACHE="+c+"/go/mod", "GOCACHE="+c+"/go/build", "GOPATH="+c+"/go",
			"CARGO_HOME="+c+"/cargo", "npm_config_cache="+c+"/npm", "UV_CACHE_DIR="+c+"/uv",
			"PIP_CACHE_DIR="+c+"/pip", "MAVEN_OPTS=-Dmaven.repo.local="+c+"/m2")
	}

	if spec.Egress != "" {
		env = append(env, "HTTPS_PROXY="+proxyAddress, "HTTP_PROXY="+proxyAddress,
			"https_proxy="+proxyAddress, "http_proxy="+proxyAddress,
			"JAVA_TOOL_OPTIONS=-Dhttps.proxyHost=127.0.0.1 -Dhttps.proxyPort=3128")
	}

	return env
}
