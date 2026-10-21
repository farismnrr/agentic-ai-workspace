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
}

type Prober interface {
	Probe(context.Context, domain.Connection) error
}
