package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

type StoredConnection struct {
	PasswordSealed string
	Connection     domain.Connection
}

type Store interface {
	Get(context.Context, string) (StoredConnection, error)
	Save(context.Context, string, StoredConnection) error
	Delete(context.Context, string) error
}

type Cipher interface {
	Encrypt([]byte) (string, error)
	Decrypt(string) ([]byte, error)
}

type ProjectOwner interface {
	Owner(context.Context, string) (string, string, error)
}

// Environment lets this feature write its generated URL as a project secret.
type Environment interface {
	SetSecret(context.Context, string, string, string) error
	DeleteSecret(context.Context, string, string) error
	Values(context.Context, string) (map[string]string, error)
}

type Prober interface {
	Probe(context.Context, domain.Connection) error
}

// Provisioner provisions a shared database identity for a project.
type Provisioner interface {
	Provision(context.Context, string) (ProvisionResult, error)
	Restore(context.Context, string, string) error
	Configured() bool
}

type ProvisionResult struct {
	Host, Database, Schema, Role, Password string
	Port                                   int
	TLS                                    bool
}

type ProvisionFailure string

const (
	AdminNotConfigured     ProvisionFailure = "admin_not_configured"
	AdminRejected          ProvisionFailure = "admin_rejected"
	InsufficientPrivileges ProvisionFailure = "insufficient_privileges"
	ManualConnectionExists ProvisionFailure = "manual_connection_exists"
	EnvironmentNameInUse   ProvisionFailure = "env_name_in_use"
	AdminURLInvalid        ProvisionFailure = "admin_url_invalid"
	CompensationFailed     ProvisionFailure = "compensation_failed"
	ProvisionFailed        ProvisionFailure = "provision_failed"
	ProvisionInProgress    ProvisionFailure = "provision_in_progress"
	LocalSaveFailed        ProvisionFailure = "local_save_failed"
	EnvironmentWriteFailed ProvisionFailure = "environment_write_failed"
)

// ProvisionError carries only a safe category; adapter details must stay private.
type ProvisionError struct {
	Category ProvisionFailure
	Remote   bool
}

func (e ProvisionError) Error() string { return string(e.Category) }
