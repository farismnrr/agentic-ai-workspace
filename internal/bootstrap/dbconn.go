package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v3"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	dbconnhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/http"
	dbconnprobe "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/probe"
	dbconnprovision "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provision"
	dbconnsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/sqlite"
	dbconnapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	dbconndomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	envapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/env/application"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

func newDBConnMount(
	db *sql.DB, cfg config.Config, projects dbconnapp.ProjectOwner, env *envapp.Service,
	account *accountapp.Service,
	auth, sameOrigin fiber.Handler,
) ([]httpserver.Mount, error) {
	cipher, err := secretbox.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open database connection secret key: %w", err)
	}
	settings := dbconnapp.NewSharedSettingsService(dbconnsqlite.NewSharedSettingsStore(db), cipher)
	provisioner := callTimeProvisioner{settings: settings, fallbackURL: cfg.PostgresAdminURL, ownPort: cfg.Port}
	service := dbconnapp.NewService(dbconnsqlite.NewStore(db), cipher, projects, env,
		dbconnprobe.New(cfg.Port), provisioner)
	return []httpserver.Mount{
		dbconnhttp.NewHandler(service).Mount(auth, sameOrigin, databaseTestLimit(), databaseSharedLimit()),
		dbconnhttp.NewSharedSettingsHandler(settings).Mount(auth, sameOrigin, accountRole(account), adminOnly(account)),
	}, nil
}

type callTimeProvisioner struct {
	settings    *dbconnapp.SharedSettingsService
	fallbackURL string
	ownPort     string
}

func (p callTimeProvisioner) Configured(ctx context.Context) (bool, error) {
	_, saved, err := p.settings.AdminSettings(ctx)
	if err != nil {
		return false, fmt.Errorf("load shared database settings: %w", err)
	}
	return saved || p.fallbackURL != "", nil
}

func (p callTimeProvisioner) Provision(ctx context.Context, projectID string) (dbconnapp.ProvisionResult, error) {
	provisioner, err := p.current(ctx)
	if err != nil {
		return dbconnapp.ProvisionResult{}, err
	}
	result, err := provisioner.Provision(ctx, projectID)
	if err != nil {
		return dbconnapp.ProvisionResult{}, fmt.Errorf("provision shared database: %w", err)
	}
	return result, nil
}

func (p callTimeProvisioner) Restore(ctx context.Context, projectID, password string) error {
	provisioner, err := p.current(ctx)
	if err != nil {
		return err
	}
	if err := provisioner.Restore(ctx, projectID, password); err != nil {
		return fmt.Errorf("restore shared database password: %w", err)
	}
	return nil
}

func (p callTimeProvisioner) current(ctx context.Context) (*dbconnprovision.Provisioner, error) {
	settings, saved, err := p.settings.AdminSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load shared database settings: %w", err)
	}
	adminURL := p.fallbackURL
	if saved {
		adminURL, err = sharedAdminURL(settings)
		if err != nil {
			return nil, fmt.Errorf("build shared database connection: %w", err)
		}
	}
	return dbconnprovision.New(adminURL, p.ownPort), nil
}

func sharedAdminURL(settings dbconndomain.SharedSettings) (string, error) {
	if err := settings.Validate(); err != nil {
		return "", fmt.Errorf("validate shared database settings: %w", err)
	}
	query := url.Values{"sslmode": []string{settings.SSLMode}}
	connection := url.URL{
		Scheme: "postgres", User: url.UserPassword(settings.User, settings.Password),
		Host: net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port)), Path: settings.Database,
		RawQuery: query.Encode(),
	}
	return connection.String(), nil
}

var (
	_ dbconnapp.ProjectOwner = (*projectapp.Service)(nil)
	_ dbconnapp.Environment  = (*envapp.Service)(nil)
	_ dbconnapp.Provisioner  = (*dbconnprovision.Provisioner)(nil)
)
