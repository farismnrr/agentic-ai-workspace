package application

import (
	"time"
)

// Lifetimes of the credentials the server mints.
const (
	requestTTL      = 10 * time.Minute
	codeTTL         = time.Minute
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
	touchInterval   = time.Minute
)

// Deps are the collaborators of the OAuth use cases.
type Deps struct {
	Clients     ClientStore
	Requests    RequestStore
	Grants      GrantStore
	Credentials CredentialStore
	Metadata    MetadataFetcher
	Signer      IDTokenSigner
	Users       UserDirectory
	Resources   ResourceCatalog
	Tokens      Tokens
	Now         func() time.Time
	Issuer      string
}

// Service holds the OAuth authorization server use cases.
type Service struct {
	Deps
}

// NewService wires the use cases.
func NewService(deps Deps) *Service {
	return &Service{Deps: deps}
}
