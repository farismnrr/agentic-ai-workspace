package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/githubapi"
	githubhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/http"
	githubsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/sqlite"
	githubapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

const githubCallback = "/api/integrations/github/callback"

// newGitHub wires the GitHub connection: the GitHub App the server signs people in with, where their
// tokens are kept (sealed), and the endpoints behind Settings, Integrations.
func newGitHub(
	db *sql.DB, cfg config.Config, auth, sameOrigin fiber.Handler,
) (*githubapp.Service, httpserver.Mount, error) {
	sealer, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // already descriptive
	}

	conf := cfg.GitHub
	client := githubapi.New(conf.ClientID, conf.ClientSecret, cfg.PublicURL+githubCallback, time.Now)
	service := githubapp.NewService(
		githubsqlite.NewStore(db), client, sealer, opaque.Generator{}, time.Now, conf.Enabled())
	mount := githubhttp.NewHandler(service, conf.AppSlug).Mount(auth, sameOrigin, githubLimits())

	return service, mount, nil
}

// connectGitHub builds the GitHub connection and signs the commands of every project in to its owner.
func connectGitHub(
	db *sql.DB, cfg config.Config, tools *agentTools, projects projectOwners, auth, sameOrigin fiber.Handler,
) (httpserver.Mount, error) {
	service, mount, err := newGitHub(db, cfg, auth, sameOrigin)
	if err != nil {
		return nil, err
	}

	tools.useGitHub(service, projects)

	importing := newImport(service, tools, projects, auth, sameOrigin)

	return func(app *fiber.App) {
		mount(app)
		importing(app)
	}, nil
}

// projectOwners says who owns a project.
type projectOwners interface {
	Owner(ctx context.Context, id string) (userID, name string, err error)
}

// useGitHub signs a project's commands in to its owner's GitHub account. The owner is looked up
// from the project each time a command runs, never taken from the caller, so a command can only
// ever act as the person who owns the project it runs in.
func (a *agentTools) useGitHub(github *githubapp.Service, projects projectOwners) {
	a.github = github
	a.git = func(projectID string) codingrunner.GitSource {
		return func(ctx context.Context) (codingrunner.Git, bool) {
			owner, _, err := projects.Owner(ctx, projectID)
			if err != nil {
				log.Printf("github: project %s has no owner to sign in as: %v", projectID, err)

				return codingrunner.Git{}, false
			}

			cred, err := github.Credentials(ctx, owner)
			if err != nil {
				log.Printf("github: commands in project %s run without GitHub: %v", projectID, err)

				return codingrunner.Git{}, false
			}

			return codingrunner.Git{
				Token: cred.Token, Name: cred.Profile.DisplayName(), Email: cred.Profile.NoreplyEmail(),
			}, true
		}
	}
}
