package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

var errUsage = errors.New("usage: agentic-ai-workspace sandbox install | status")

// SandboxCommand runs `agentic-ai-workspace sandbox <install|status>`, the owner's way to
// set up the toolbox every sandbox is built from.
func SandboxCommand(args []string) error {
	if len(args) != 1 {
		return errUsage
	}

	cfg := config.Load()

	switch args[0] {
	case "install":
		return installToolbox(cfg)
	case "status":
		return toolboxStatus(cfg)
	}

	return errUsage
}

func installToolbox(cfg config.Config) error {
	list := cfg.SandboxPackages

	switch list {
	case "":
		list = sandbox.DefaultPackages
	case "none":
		list = ""
	}

	packages, err := sandbox.ParsePackages(list)
	if err != nil {
		return fmt.Errorf("SANDBOX_PACKAGES: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("Building the sandbox toolbox in %s\n", toolboxDir(cfg))

	opts := sandbox.InstallOptions{Out: os.Stdout, Dir: toolboxDir(cfg), Packages: packages}
	if err := sandbox.Install(ctx, opts); err != nil {
		return fmt.Errorf("install toolbox: %w", err)
	}

	fmt.Println("Done. Restart the server so it picks up the toolbox.")

	return nil
}

func toolboxStatus(cfg config.Config) error {
	marker, err := sandbox.ReadMarker(toolboxDir(cfg))
	if err != nil {
		fmt.Println("Toolbox: not installed. Run: agentic-ai-workspace sandbox install")
	} else {
		fmt.Printf("Toolbox: Alpine %s, installed %s\n  packages: %s\n", marker.Alpine,
			marker.InstalledAt.Format("2006-01-02 15:04"), strings.Join(marker.Packages, " "))
	}

	if _, err := sandbox.OpenCgroups(); err != nil {
		fmt.Printf("Resource limits (cgroups): NOT available: %v\n", err)
	} else {
		fmt.Println("Resource limits (cgroups): available")
	}

	return nil
}
