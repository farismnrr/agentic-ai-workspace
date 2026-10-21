package provisionmysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provision"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

const operationTimeout = 10 * time.Second

const generatedPasswordLength = 40

const (
	mysqlAccessDenied    = 1045
	mysqlUserRejected    = 1698
	mysqlDatabaseDenied  = 1044
	mysqlPrivilegeDenied = 1227
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	Close() error
}

type connector func(context.Context, *mysqldriver.Config) (executor, error)

// Provisioner reconciles a database-scoped MySQL account for each project.
type Provisioner struct {
	connect  connector
	settings domain.SharedSettings
}

func New(settings domain.SharedSettings) *Provisioner {
	return &Provisioner{connect: connect, settings: settings}
}

func (p *Provisioner) Configured(context.Context) (bool, error) {
	return p != nil && p.settings.Kind == string(domain.MySQL) && p.settings.Password != "", nil
}

func (p *Provisioner) Provision(ctx context.Context, projectID string) (application.ProvisionResult, error) {
	password, err := generatedPassword()
	if err != nil {
		return application.ProvisionResult{}, provisionError(application.ProvisionFailed)
	}
	return p.provisionPassword(ctx, projectID, password)
}

func (p *Provisioner) Restore(ctx context.Context, projectID, password string) error {
	_, err := p.provisionPassword(ctx, projectID, password)
	return err
}

func (p *Provisioner) provisionPassword(
	ctx context.Context, projectID, password string,
) (application.ProvisionResult, error) {
	if err := p.validate(); err != nil {
		return application.ProvisionResult{}, err
	}
	name := domain.ProjectDatabaseIdentifier(projectID)
	if !domain.ValidDatabaseIdentifier(name) {
		return application.ProvisionResult{}, provisionError(application.AdminURLInvalid)
	}
	config, err := p.adminConfig()
	if err != nil {
		return application.ProvisionResult{}, provisionError(application.AdminURLInvalid)
	}
	admin, err := p.connect(ctx, config)
	if err != nil {
		return application.ProvisionResult{}, classifyError(err)
	}
	defer func() { _ = admin.Close() }()
	operationCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	queries, err := statements(name, password)
	if err != nil {
		return application.ProvisionResult{}, provisionError(application.AdminURLInvalid)
	}
	for _, statement := range queries {
		if _, err := admin.ExecContext(operationCtx, statement); err != nil {
			return application.ProvisionResult{}, classifyError(err)
		}
	}
	return application.ProvisionResult{
		Host: p.settings.Host, Port: p.settings.Port, Database: name, Role: name,
		Password: password, TLS: p.settings.TLS != "none", CACert: p.settings.CACert,
	}, nil
}

func (p *Provisioner) validate() error {
	if p == nil || p.settings.Kind != string(domain.MySQL) || p.settings.Password == "" {
		return provisionError(application.AdminNotConfigured)
	}
	if err := p.settings.Validate(); err != nil {
		return provisionError(application.AdminURLInvalid)
	}
	if err := p.settings.ValidateCertificates(); err != nil {
		return provisionError(application.AdminURLInvalid)
	}
	return nil
}

func (p *Provisioner) adminConfig() (*mysqldriver.Config, error) {
	config := mysqldriver.NewConfig()
	config.User, config.Passwd, config.Net = p.settings.User, p.settings.Password, "tcp"
	config.Addr = net.JoinHostPort(p.settings.Host, fmt.Sprint(p.settings.Port))
	config.DBName = p.settings.Database
	config.ParseTime = true
	var err error
	config.TLS, err = provision.BuildTLSConfig(p.settings.TLS, p.settings.Host, provision.TLSMaterial{
		CACert: p.settings.CACert, ClientCert: p.settings.ClientCert, ClientKey: p.settings.ClientKey,
	})
	if err != nil {
		return nil, fmt.Errorf("build MySQL TLS settings: %w", err)
	}
	return config, nil
}

func connect(ctx context.Context, config *mysqldriver.Config) (executor, error) {
	dsn := config.FormatDSN()
	parsed, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse MySQL admin connection settings: %w", err)
	}
	parsed.TLS = config.TLS
	connector, err := mysqldriver.NewConnector(parsed)
	if err != nil {
		return nil, fmt.Errorf("create MySQL admin connector: %w", err)
	}
	return sql.OpenDB(connector), nil
}

func generatedPassword() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var password strings.Builder
	password.Grow(generatedPasswordLength)
	for range generatedPasswordLength {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate MySQL project password: %w", err)
		}
		password.WriteByte(alphabet[index.Int64()])
	}
	return password.String(), nil
}

func provisionError(category application.ProvisionFailure) error {
	return application.ProvisionError{Category: category, Remote: category == application.ProvisionFailed}
}

func classifyError(err error) error {
	var server *mysqldriver.MySQLError
	if errors.As(err, &server) {
		switch server.Number {
		case mysqlAccessDenied, mysqlUserRejected:
			return provisionError(application.AdminRejected)
		case mysqlDatabaseDenied, mysqlPrivilegeDenied:
			return provisionError(application.InsufficientPrivileges)
		}
	}
	return provisionError(application.ProvisionFailed)
}

var _ application.Provisioner = (*Provisioner)(nil)
