package bootstrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"

	cliagenthttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/http"
	cliagentapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/application"
	cliagentdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const homeMode = 0o700

var safeUser = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// plans is everything for using a person's own plan with a vendor's coding program: signing in to it, and
// doing a task with it. The programs run in the toolbox image, each person with a folder of their own.
type plans struct {
	service *cliagentapp.Service
	engine  *codingrunner.Engine
	folders *folder.Folders
	hosts   map[string][]string
	box     cliSandbox
}

func newPlans(cfg config.Config, engine *codingrunner.Engine, folders *folder.Folders) *plans {
	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		dataDir = cfg.DataDir
	}

	box := cliSandbox{engine: engine, base: filepath.Join(dataDir, "cli")}
	hosts := map[string][]string{
		string(cliagentdomain.Claude): cfg.ClaudeEgress, string(cliagentdomain.ChatGPT): cfg.ChatGPTEgress,
	}
	signIn := map[cliagentdomain.Runtime][]string{
		cliagentdomain.Claude: cfg.ClaudeEgress, cliagentdomain.ChatGPT: cfg.ChatGPTEgress,
	}

	return &plans{
		service: cliagentapp.NewService(box, signIn, time.Now), engine: engine, folders: folders, hosts: hosts, box: box,
	}
}

// mount serves the sign-in endpoints.
func (p *plans) mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return cliagenthttp.NewHandler(p.service).Mount(auth, sameOrigin, signInLimit())
}

// erase removes everything the programs kept for a person.
func (p *plans) erase(ctx context.Context, userID string) error { return p.box.erase(ctx, userID) }

// cliSandbox runs a vendor's program in the toolbox image with the person's own folder as its home.
// The server never opens that folder: only the program reads and writes the sign-in in it.
type cliSandbox struct {
	engine *codingrunner.Engine
	base   string
}

// Run implements the use cases' sandbox port.
func (b cliSandbox) Run(ctx context.Context, job cliagentapp.Job, out io.Writer) (int, error) {
	home, err := b.home(job.UserID)
	if err != nil {
		return 0, err
	}

	docker, egress, release, err := b.engine.Slot(ctx, "cli-"+job.UserID, job.Hosts...)
	if err != nil {
		return 0, fmt.Errorf("start program: %w", err)
	}

	defer release()

	env := append([]string{
		"LANG=C.UTF-8", "PATH=" + sandbox.Path, "TMPDIR=/tmp", "HOME=" + sandbox.Home,
		"CODEX_HOME=" + sandbox.Home + "/.codex",
	}, job.Env...)
	if egress != "" {
		env = append(env, codingrunner.ProxyEnv()...)
	}

	res, err := docker.Run(ctx, sandbox.Spec{Home: home, Egress: egress, Stdin: job.Stdin, Argv: job.Argv, Env: env}, out)
	if err != nil {
		return 0, fmt.Errorf("run program: %w", err)
	}

	return res.ExitCode, nil
}

// home is the person's folder, made on first use. The names are opaque ids, checked all the same.
func (b cliSandbox) home(userID string) (string, error) {
	if !safeUser.MatchString(userID) {
		return "", fmt.Errorf("%w: bad person id", cliagentdomain.ErrUnavailable)
	}

	dir := filepath.Join(b.base, userID)

	for _, sub := range []string{"", ".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), homeMode); err != nil {
			return "", fmt.Errorf("make home: %w", err)
		}
	}

	return dir, nil
}

// erase removes everything the programs kept for a person.
func (b cliSandbox) erase(_ context.Context, userID string) error {
	if !safeUser.MatchString(userID) {
		return nil
	}

	b.engine.Forget(filepath.Join(b.base, "cli-"+userID))

	if err := os.RemoveAll(filepath.Join(b.base, userID)); err != nil {
		return fmt.Errorf("remove programs' folder: %w", err)
	}

	return nil
}
