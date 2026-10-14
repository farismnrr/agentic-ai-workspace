package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const (
	millisPerCore = 1000
	mebibyte      = 1 << 20
	warmTimeout   = 15 * time.Minute
)

// sandboxConfig turns the server settings into what the Docker sandbox needs.
func sandboxConfig(cfg config.Config) sandbox.Config {
	limits := sandbox.DefaultLimits()
	if cfg.SandboxMemoryMB > 0 {
		limits.MemoryBytes = uint64(cfg.SandboxMemoryMB) * mebibyte
	}

	if cfg.SandboxCPUs > 0 {
		limits.CPUMillis = uint64(cfg.SandboxCPUs) * millisPerCore
	}

	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		dataDir = cfg.DataDir
	}

	owner := sha256.Sum256([]byte(dataDir + "|" + cfg.SandboxDataVolume))

	return sandbox.Config{
		Owner: hex.EncodeToString(owner[:6]), Image: cfg.SandboxImage, Runtime: cfg.SandboxRuntime,
		DataVolume: cfg.SandboxDataVolume, DataDir: dataDir, Limits: limits, UID: os.Getuid(), GID: os.Getgid(),
	}
}

// newSandboxEngine builds the infrastructure all runners share. Docker is reached lazily and
// warmed in the background, so a missing daemon never keeps the server from starting; checks
// are answered with a plain reason until it works.
func newSandboxEngine(cfg config.Config) (*codingrunner.Engine, func()) {
	conf := sandboxConfig(cfg)
	hub := egressproxy.NewHub(filepath.Join(conf.DataDir, "egress"), egressproxy.New(cfg.SandboxEgress))

	engine := codingrunner.NewEngine(codingrunner.EngineConfig{
		Size:   cfg.MaxSandboxes,
		Extras: codingrunner.Extras{CacheDir: filepath.Join(conf.DataDir, "caches"), Proxy: hub},
		Open: func(ctx context.Context) (*sandbox.Docker, error) {
			docker, err := sandbox.Open(ctx, conf)
			if err != nil {
				return nil, fmt.Errorf("checks need Docker, which the server cannot use yet: %w", err)
			}

			return docker, nil
		},
	})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), warmTimeout)
		defer cancel()

		if _, err := engine.Docker(ctx); err != nil {
			log.Printf("sandbox: %v", err)

			return
		}

		log.Printf("sandbox: Docker ready (image %s, %d MB and %d CPU per check, %d at a time)",
			conf.Image, conf.Limits.MemoryBytes/mebibyte, conf.Limits.CPUMillis/millisPerCore, cfg.MaxSandboxes)
	}()

	return engine, hub.Close
}
