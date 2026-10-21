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
	PasswordSealed string
	Settings       domain.SharedSettings
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
	Host, Database, User, SSLMode string
	Port                          int
	PasswordSet, Configured       bool
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
	return stored.Settings, true, nil
}

func (s *SharedSettingsService) Save(ctx context.Context, input domain.SharedSettings) (SharedSettingsView, error) {
	if input.Password == "" {
		password, err := s.savedPassword(ctx)
		if err != nil {
			return SharedSettingsView{}, err
		}
		input.Password = password
	}
	if err := input.Validate(); err != nil {
		return SharedSettingsView{}, fmt.Errorf("validate shared database settings: %w", err)
	}
	sealed, err := s.cipher.Encrypt([]byte(input.Password))
	if err != nil {
		return SharedSettingsView{}, fmt.Errorf("seal shared database password: %w", err)
	}
	stored := StoredSharedSettings{Settings: input, PasswordSealed: sealed}
	stored.Settings.Password = ""
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
	}
}
