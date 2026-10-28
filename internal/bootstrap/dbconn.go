package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v3"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	dbconnhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/http"
	dbconnprobe "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/probe"
	dbconnprovision "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provision"
	dbconnmongoprovision "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provisionmongodb"
	dbconnmysqlprovision "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provisionmysql"
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
	provisioners := callTimeProvisioners{settings: settings, fallbackURL: cfg.PostgresAdminURL, ownPort: cfg.Port}
	service := dbconnapp.NewServiceWithProvisioners(dbconnsqlite.NewStore(db), cipher, projects, env,
		dbconnprobe.New(cfg.Port), provisioners)
	return []httpserver.Mount{
		dbconnhttp.NewHandler(service).Mount(auth, sameOrigin,
			dbconnhttp.TestRateLimit(), dbconnhttp.SharedRateLimit()),
		dbconnhttp.NewSharedSettingsHandler(settings, service).Mount(
			auth, sameOrigin, accountRole(account), adminOnly(account)),
	}, nil
}

type callTimeProvisioners struct {
	settings    *dbconnapp.SharedSettingsService
	fallbackURL string
	ownPort     string
}

func (p callTimeProvisioners) For(kind dbconndomain.Kind) dbconnapp.Provisioner {
	return callTimeProvisioner{settings: p.settings, fallbackURL: p.fallbackURL, ownPort: p.ownPort, kind: kind}
}

type callTimeProvisioner struct {
	settings    *dbconnapp.SharedSettingsService
	fallbackURL string
	ownPort     string
	kind        dbconndomain.Kind
}

func (p callTimeProvisioner) databaseKind() dbconndomain.Kind {
	if p.kind == "" {
		return dbconndomain.Postgres
	}
	return p.kind
}

func (p callTimeProvisioner) Configured(ctx context.Context) (bool, error) {
	kind := p.databaseKind()
	saved, err := p.settings.Configured(ctx, string(kind))
	if err != nil {
		return false, fmt.Errorf("load shared database settings: %w", err)
	}
	return saved || kind == dbconndomain.Postgres && p.fallbackURL != "", nil
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

func (p callTimeProvisioner) current(ctx context.Context) (dbconnapp.Provisioner, error) {
	kind := p.databaseKind()
	settings, saved, err := p.settings.AdminSettingsForKind(ctx, string(kind))
	if err != nil {
		return nil, fmt.Errorf("load shared database settings: %w", err)
	}
	switch kind {
	case dbconndomain.Postgres:
		return p.postgres(settings, saved)
	case dbconndomain.MySQL:
		if !saved {
			return nil, dbconnapp.ProvisionError{Category: dbconnapp.KindNotConfigured}
		}
		return dbconnmysqlprovision.New(settings), nil
	case dbconndomain.MongoDB:
		if !saved {
			return nil, dbconnapp.ProvisionError{Category: dbconnapp.KindNotConfigured}
		}
		return dbconnmongoprovision.New(settings), nil
	default:
		return nil, dbconnapp.ProvisionError{Category: dbconnapp.KindNotConfigured}
	}
}

func (p callTimeProvisioner) postgres(
	settings dbconndomain.SharedSettings, saved bool,
) (*dbconnprovision.Provisioner, error) {
	if !saved {
		return dbconnprovision.New(p.fallbackURL, p.ownPort), nil
	}
	adminURL, err := dbconnprovision.SharedAdminURL(settings)
	if err != nil {
		return nil, fmt.Errorf("build shared database connection: %w", err)
	}
	material := dbconnprovision.TLSMaterial{
		CACert: settings.CACert, ClientCert: settings.ClientCert, ClientKey: settings.ClientKey,
	}
	return dbconnprovision.New(adminURL, p.ownPort, material), nil
}

var (
	_ dbconnapp.ProjectOwner = (*projectapp.Service)(nil)
	_ dbconnapp.Environment  = (*envapp.Service)(nil)
	_ dbconnapp.Provisioner  = (*dbconnprovision.Provisioner)(nil)
)
