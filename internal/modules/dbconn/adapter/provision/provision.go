package provision

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
)

const (
	provisionTimeout   = 10 * time.Second
	passwordByteLength = 32
)

type Provisioner struct {
	connect  connector
	adminURL string
	ownPort  string
}

type connector func(context.Context, string, string) (session, error)

type session interface {
	Begin(context.Context) (transaction, error)
	Close(context.Context) error
}

type transaction interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) row
	Commit(context.Context) error
	Rollback(context.Context) error
}

type row interface{ Scan(...any) error }

func New(adminURL, ownPort string) *Provisioner {
	return &Provisioner{adminURL: strings.TrimSpace(adminURL), ownPort: ownPort, connect: connect}
}

func (p *Provisioner) Configured() bool { return p != nil && p.adminURL != "" }

func (p *Provisioner) Provision(ctx context.Context, projectID string) (application.ProvisionResult, error) {
	if !p.Configured() {
		return application.ProvisionResult{}, provisionError(application.AdminNotConfigured)
	}
	cfg, err := parseAdminURL(p.adminURL)
	if err != nil {
		return application.ProvisionResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, provisionTimeout)
	defer cancel()
	password, err := generatedPassword()
	if err != nil {
		return application.ProvisionResult{}, provisionError(application.ProvisionFailed)
	}
	return p.provisionPassword(ctx, projectID, cfg, password)
}

func (p *Provisioner) Restore(ctx context.Context, projectID, password string) error {
	if !p.Configured() {
		return provisionError(application.AdminNotConfigured)
	}
	cfg, err := parseAdminURL(p.adminURL)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, provisionTimeout)
	defer cancel()
	_, err = p.provisionPassword(ctx, projectID, cfg, password)
	return err
}

func parseAdminURL(rawURL string) (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig(rawURL)
	parsed, urlErr := url.Parse(rawURL)
	if err != nil || urlErr != nil || strings.Trim(strings.TrimSpace(parsed.Path), "/") == "" ||
		strings.TrimSpace(cfg.Database) == "" {
		return nil, provisionError(application.AdminURLInvalid)
	}
	return cfg, nil
}

func (p *Provisioner) provisionPassword(
	ctx context.Context, projectID string, cfg *pgx.ConnConfig, password string,
) (application.ProvisionResult, error) {
	name := projectName(projectID)
	s, err := p.connect(ctx, p.adminURL, p.ownPort)
	if err != nil {
		return application.ProvisionResult{}, classifyError(err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = s.Close(closeCtx)
	}()
	if err := apply(ctx, s, cfg.Database, name, password); err != nil {
		return application.ProvisionResult{}, classifyError(err)
	}
	return application.ProvisionResult{
		Host: cfg.Host, Port: int(cfg.Port), Database: cfg.Database, Schema: name, Role: name,
		Password: password, TLS: adminTLS(p.adminURL),
	}, nil
}

func adminTLS(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	mode := parsed.Query().Get("sslmode")
	return mode != "" && mode != "disable"
}

func projectName(projectID string) string {
	digest := sha256.Sum256([]byte(projectID))
	return "p_" + fmt.Sprintf("%x", digest[:12])
}

func generatedPassword() (string, error) {
	secret := make([]byte, passwordByteLength)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate project password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func provisionError(category application.ProvisionFailure) error {
	return application.ProvisionError{Category: category, Remote: category == application.ProvisionFailed}
}

func classifyError(err error) error {
	if errors.Is(err, errPublicPrivilegesOpen) {
		return provisionError(application.InsufficientPrivileges)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if strings.HasPrefix(pgErr.Code, "28") {
			return provisionError(application.AdminRejected)
		}
		if pgErr.Code == "42501" {
			return provisionError(application.InsufficientPrivileges)
		}
	}
	return provisionError(application.ProvisionFailed)
}
