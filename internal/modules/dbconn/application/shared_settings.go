package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

var (
	ErrSharedSettingsMissing  = errors.New("shared database settings missing")
	ErrSharedPasswordRequired = errors.New("shared database password required")
)

type StoredSharedSettings struct {
	PasswordSealed, CACertSealed, ClientCertSealed, ClientKeySealed string
	Settings                                                        domain.SharedSettings
}

// SharedSettingsStore persists one encrypted server-wide config per database kind.
type SharedSettingsStore interface {
	Get(context.Context, string) (StoredSharedSettings, error)
	Save(context.Context, string, StoredSharedSettings) error
}

type SharedSettingsService struct {
	store  SharedSettingsStore
	cipher Cipher
}

func NewSharedSettingsService(store SharedSettingsStore, cipher Cipher) *SharedSettingsService {
	return &SharedSettingsService{store: store, cipher: cipher}
}

type SharedSettingsView struct {
	Kind, Host, Database, User, SSLMode, TLS string
	Port                                     int
	PasswordSet, Configured                  bool
	CACertSet, ClientCertSet, ClientKeySet   bool
}

func (s *SharedSettingsService) Get(ctx context.Context, kind string) (SharedSettingsView, error) {
	if !domain.IsSharedKind(kind) {
		return SharedSettingsView{}, domain.ErrInvalidSharedSettings
	}
	stored, err := s.store.Get(ctx, kind)
	if errors.Is(err, ErrSharedSettingsMissing) {
		return SharedSettingsView{
			Kind: kind, Port: domain.DefaultSharedPort(kind), TLS: "none",
		}, nil
	}
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("load shared database settings: %w", err)
	}
	stored.Settings.Kind = kind
	return sharedSettingsView(stored, true), nil
}

func (s *SharedSettingsService) Configured(ctx context.Context, kind string) (bool, error) {
	if !domain.IsSharedKind(kind) {
		return false, domain.ErrInvalidSharedSettings
	}
	stored, err := s.store.Get(ctx, kind)
	if errors.Is(err, ErrSharedSettingsMissing) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load shared database settings: %w", err)
	}
	return stored.PasswordSealed != "", nil
}

func (s *SharedSettingsService) AdminSettings(ctx context.Context) (domain.SharedSettings, bool, error) {
	return s.AdminSettingsForKind(ctx, "postgres")
}

// AdminSettingsForKind opens the saved settings needed by a database provisioner.
func (s *SharedSettingsService) AdminSettingsForKind(
	ctx context.Context, kind string,
) (domain.SharedSettings, bool, error) {
	if !domain.IsSharedKind(kind) {
		return domain.SharedSettings{}, false, domain.ErrInvalidSharedSettings
	}
	stored, err := s.store.Get(ctx, kind)
	if errors.Is(err, ErrSharedSettingsMissing) {
		return domain.SharedSettings{}, false, nil
	}
	if err != nil {
		return domain.SharedSettings{}, false, fmt.Errorf("load shared database settings: %w", err)
	}
	stored.Settings.Kind = kind
	password, err := s.cipher.Decrypt(stored.PasswordSealed)
	if err != nil {
		return domain.SharedSettings{}, false, fmt.Errorf("open shared database password: %w", err)
	}
	stored.Settings.Password = string(password)
	if err := s.openCertificates(&stored); err != nil {
		return domain.SharedSettings{}, false, err
	}
	return stored.Settings, true, nil
}

func (s *SharedSettingsService) Save(
	ctx context.Context, kind string, input domain.SharedSettings,
) (SharedSettingsView, error) {
	return s.SaveWithRemovals(ctx, kind, input, false, false)
}

// SaveWithRemovals keeps blank certificate fields unless their explicit remove flag is set.
func (s *SharedSettingsService) SaveWithRemovals(
	ctx context.Context, kind string, input domain.SharedSettings, removeCA, removeClient bool,
) (SharedSettingsView, error) {
	if !domain.IsSharedKind(kind) {
		return SharedSettingsView{}, domain.ErrInvalidSharedSettings
	}
	input = normalizeSharedSettings(kind, input)
	previous, err := s.previous(ctx, kind)
	if err != nil && !errors.Is(err, ErrSharedSettingsMissing) {
		return SharedSettingsView{}, err
	}
	input, err = s.retainCertificateMaterial(input, previous, removeCA, removeClient)
	if err != nil {
		return SharedSettingsView{}, err
	}
	if input.Password == "" {
		input.Password, err = s.savedPassword(ctx, kind)
		if err != nil {
			return SharedSettingsView{}, err
		}
	}
	if err := input.Validate(); err != nil {
		return SharedSettingsView{}, fmt.Errorf("validate shared database settings: %w", err)
	}
	return s.persistSettings(ctx, input)
}

func normalizeSharedSettings(kind string, input domain.SharedSettings) domain.SharedSettings {
	input.Kind = kind
	if input.Port == 0 {
		input.Port = domain.DefaultSharedPort(kind)
	}
	if input.TLS == "" {
		input.TLS = "none"
	}
	if kind == "postgres" {
		input.SSLMode = "verify-full"
		if input.TLS == "none" {
			input.SSLMode = "disable"
		}
	}
	return input
}

func (s *SharedSettingsService) retainCertificateMaterial(
	input domain.SharedSettings, previous StoredSharedSettings, removeCA, removeClient bool,
) (domain.SharedSettings, error) {
	var err error
	input.CACert, err = retainedPEM(s, input.CACert, previous.CACertSealed, removeCA)
	if err != nil {
		return domain.SharedSettings{}, err
	}
	input.ClientCert, err = retainedPEM(s, input.ClientCert, previous.ClientCertSealed, removeClient)
	if err != nil {
		return domain.SharedSettings{}, err
	}
	input.ClientKey, err = retainedPEM(s, input.ClientKey, previous.ClientKeySealed, removeClient)
	if err != nil {
		return domain.SharedSettings{}, err
	}
	if err := input.ValidateCertificates(); err != nil {
		return domain.SharedSettings{}, fmt.Errorf("validate database certificates: %w", err)
	}
	return input, nil
}
