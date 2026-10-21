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

// SharedSettingsStore persists the one server-wide PostgreSQL provisioning config.
type SharedSettingsStore interface {
	Get(context.Context) (StoredSharedSettings, error)
	Save(context.Context, StoredSharedSettings) error
}

type SharedSettingsService struct {
	store  SharedSettingsStore
	cipher Cipher
}

func NewSharedSettingsService(store SharedSettingsStore, cipher Cipher) *SharedSettingsService {
	return &SharedSettingsService{store: store, cipher: cipher}
}

type SharedSettingsView struct {
	Host, Database, User, SSLMode          string
	Port                                   int
	PasswordSet, Configured                bool
	CACertSet, ClientCertSet, ClientKeySet bool
}

func (s *SharedSettingsService) Get(ctx context.Context) (SharedSettingsView, error) {
	stored, err := s.store.Get(ctx)
	if errors.Is(err, ErrSharedSettingsMissing) {
		return SharedSettingsView{}, nil
	}
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("load shared database settings: %w", err)
	}
	return sharedSettingsView(stored, true), nil
}

func (s *SharedSettingsService) AdminSettings(ctx context.Context) (domain.SharedSettings, bool, error) {
	stored, err := s.store.Get(ctx)
	if errors.Is(err, ErrSharedSettingsMissing) {
		return domain.SharedSettings{}, false, nil
	}
	if err != nil {
		return domain.SharedSettings{}, false, fmt.Errorf("load shared database settings: %w", err)
	}
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

func (s *SharedSettingsService) Save(ctx context.Context, input domain.SharedSettings) (SharedSettingsView, error) {
	return s.SaveWithRemovals(ctx, input, false, false)
}

// SaveWithRemovals keeps blank certificate fields unless their explicit remove flag is set.
func (s *SharedSettingsService) SaveWithRemovals(
	ctx context.Context, input domain.SharedSettings, removeCA, removeClient bool,
) (SharedSettingsView, error) {
	previous, err := s.previous(ctx)
	if err != nil && !errors.Is(err, ErrSharedSettingsMissing) {
		return SharedSettingsView{}, err
	}
	input, err = s.retainCertificateMaterial(input, previous, removeCA, removeClient)
	if err != nil {
		return SharedSettingsView{}, err
	}
	if input.Password == "" {
		input.Password, err = s.savedPassword(ctx)
		if err != nil {
			return SharedSettingsView{}, err
		}
	}
	if err := input.Validate(); err != nil {
		return SharedSettingsView{}, fmt.Errorf("validate shared database settings: %w", err)
	}
	return s.persistSettings(ctx, input)
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

func (s *SharedSettingsService) persistSettings(
	ctx context.Context, input domain.SharedSettings,
) (SharedSettingsView, error) {
	sealed, err := s.cipher.Encrypt([]byte(input.Password))
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("seal shared database password: %w", err)
	}
	ca, err := sealOptional(s.cipher, input.CACert)
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("seal database CA: %w", err)
	}
	cert, err := sealOptional(s.cipher, input.ClientCert)
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("seal database client certificate: %w", err)
	}
	key, err := sealOptional(s.cipher, input.ClientKey)
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("seal database client key: %w", err)
	}
	stored := StoredSharedSettings{
		Settings: input, PasswordSealed: sealed,
		CACertSealed: ca, ClientCertSealed: cert, ClientKeySealed: key,
	}
	stored.Settings.Password = ""
	stored.Settings.CACert, stored.Settings.ClientCert, stored.Settings.ClientKey = "", "", ""
	if err := s.store.Save(ctx, stored); err != nil {
		return SharedSettingsView{}, fmt.Errorf("save shared database settings: %w", err)
	}
	return sharedSettingsView(stored, true), nil
}

func (s *SharedSettingsService) savedPassword(ctx context.Context) (string, error) {
	stored, err := s.store.Get(ctx)
	if errors.Is(err, ErrSharedSettingsMissing) {
		return "", ErrSharedPasswordRequired
	}
	if err != nil {
		return "", fmt.Errorf("load shared database settings: %w", err)
	}
	password, err := s.cipher.Decrypt(stored.PasswordSealed)
	if err != nil {
		return "", fmt.Errorf("open shared database password: %w", err)
	}
	return string(password), nil
}

func sharedSettingsView(stored StoredSharedSettings, configured bool) SharedSettingsView {
	settings := stored.Settings
	return SharedSettingsView{
		Host: settings.Host, Port: settings.Port, Database: settings.Database,
		User: settings.User, SSLMode: settings.SSLMode,
		PasswordSet: stored.PasswordSealed != "", Configured: configured,
		CACertSet: stored.CACertSealed != "", ClientCertSet: stored.ClientCertSealed != "",
		ClientKeySet: stored.ClientKeySealed != "",
	}
}
