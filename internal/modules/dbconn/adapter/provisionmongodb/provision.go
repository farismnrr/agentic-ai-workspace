package provisionmongodb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provision"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

const operationTimeout = 10 * time.Second

const generatedPasswordLength = 40

const (
	mongoAuthenticationFailed = 18
	mongoUnauthorized         = 13
)

type commandRunner interface {
	Run(context.Context, string, bson.D) error
	Close(context.Context) error
}

type clientFactory func(*options.ClientOptions) (commandRunner, error)

// Provisioner reconciles a readWrite MongoDB user for each project database.
type Provisioner struct {
	connect  clientFactory
	settings domain.SharedSettings
}

func New(settings domain.SharedSettings) *Provisioner {
	return &Provisioner{connect: connect, settings: settings}
}

func (p *Provisioner) Configured(context.Context) (bool, error) {
	return p != nil && p.settings.Kind == string(domain.MongoDB) && p.settings.Password != "", nil
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
	clientOptions, err := p.clientOptions()
	if err != nil {
		return application.ProvisionResult{}, provisionError(application.AdminURLInvalid)
	}
	client, err := p.connect(clientOptions)
	if err != nil {
		return application.ProvisionResult{}, classifyError(err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = client.Close(closeCtx)
	}()
	operationCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := reconcileUser(operationCtx, client, name, password); err != nil {
		return application.ProvisionResult{}, classifyError(err)
	}
	return application.ProvisionResult{
		Host: p.settings.Host, Port: p.settings.Port, Database: name, Role: name,
		Password: password, TLS: p.settings.TLS != "none", CACert: p.settings.CACert,
	}, nil
}

func (p *Provisioner) validate() error {
	if p == nil || p.settings.Kind != string(domain.MongoDB) || p.settings.Password == "" {
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

func (p *Provisioner) clientOptions() (*options.ClientOptions, error) {
	authSource := p.settings.Database
	if authSource == "" {
		authSource = "admin"
	}
	query := url.Values{"authSource": []string{authSource}}
	if p.settings.TLS != "none" {
		query.Set("tls", "true")
	}
	address := net.JoinHostPort(p.settings.Host, strconv.Itoa(p.settings.Port))
	uri := "mongodb://" + address + "/" + url.PathEscape(p.settings.Database) + "?" + query.Encode()
	tlsConfig, err := provision.BuildTLSConfig(p.settings.TLS, p.settings.Host, provision.TLSMaterial{
		CACert: p.settings.CACert, ClientCert: p.settings.ClientCert, ClientKey: p.settings.ClientKey,
	})
	if err != nil {
		return nil, fmt.Errorf("build MongoDB TLS settings: %w", err)
	}
	return options.Client().ApplyURI(uri).SetAuth(options.Credential{
		Username: p.settings.User, Password: p.settings.Password, AuthSource: authSource,
	}).SetTLSConfig(tlsConfig), nil
}

func connect(clientOptions *options.ClientOptions) (commandRunner, error) {
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect MongoDB admin client: %w", err)
	}
	return mongoCommands{client: client}, nil
}

type mongoCommands struct{ client *mongo.Client }

func (m mongoCommands) Run(ctx context.Context, database string, command bson.D) error {
	if err := m.client.Database(database).RunCommand(ctx, command).Err(); err != nil {
		return fmt.Errorf("run MongoDB project user command: %w", err)
	}
	return nil
}

func (m mongoCommands) Close(ctx context.Context) error {
	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect MongoDB admin client: %w", err)
	}
	return nil
}

func provisionError(category application.ProvisionFailure) error {
	return application.ProvisionError{Category: category, Remote: category == application.ProvisionFailed}
}

func classifyError(err error) error {
	var command mongo.CommandError
	if errors.As(err, &command) {
		switch command.Code {
		case mongoAuthenticationFailed:
			return provisionError(application.AdminRejected)
		case mongoUnauthorized:
			return provisionError(application.InsufficientPrivileges)
		}
	}
	return provisionError(application.ProvisionFailed)
}

var _ application.Provisioner = (*Provisioner)(nil)
