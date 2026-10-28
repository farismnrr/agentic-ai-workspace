// Package apptest provides an in-process application harness for Go integration tests.
package apptest

import (
	"database/sql"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const publicURL = "https://example.test"

// Environment owns an in-process app and its isolated database and files.
type Environment struct {
	App     *fiber.App
	DB      *sql.DB
	DataDir string
}

// NewEnv builds an app with its default test configuration.
func NewEnv(t *testing.T) Environment {
	t.Helper()

	return NewEnvWith(t, nil)
}

// NewEnvWith builds an app after applying an optional configuration adjustment.
func NewEnvWith(t *testing.T, adjust func(*config.Config)) Environment {
	t.Helper()

	return NewEnvServing(t, adjust, fstest.MapFS{"index.html": {Data: []byte("<html>")}})
}

// NewEnvServing builds an app with the supplied frontend files.
func NewEnvServing(t *testing.T, adjust func(*config.Config), frontend fs.FS) Environment {
	t.Helper()

	cfg := config.Config{
		Port: "0", PublicURL: publicURL, DataDir: t.TempDir(), CookieSecure: true,
		SandboxImage: "busybox:1.37", MaxSandboxes: maxSandboxCount,
		SiteMaxPerUser: sitesPerUser, SiteMaxTotal: maxSitesTotal,
		AllowedOrigins: []string{"http://localhost:8081", "http://127.0.0.1:8081"},
		CORSOrigins:    []string{"http://localhost:8081"},
	}
	if adjust != nil {
		adjust(&cfg)
	}

	db, err := database.Open(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mounts, stop, err := bootstrap.Wire(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)

	return Environment{
		App: httpserver.New(frontend, httpserver.Settings{}, mounts...), DataDir: cfg.DataDir, DB: db,
	}
}
