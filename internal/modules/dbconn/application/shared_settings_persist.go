package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

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
	if err := s.store.Save(ctx, input.Kind, stored); err != nil {
		return SharedSettingsView{}, fmt.Errorf("save shared database settings: %w", err)
	}
	return sharedSettingsView(stored, true), nil
}

func (s *SharedSettingsService) savedPassword(ctx context.Context, kind string) (string, error) {
	stored, err := s.store.Get(ctx, kind)
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
		Kind: settings.Kind, Host: settings.Host, Port: settings.Port, Database: settings.Database,
		User: settings.User, SSLMode: settings.SSLMode, TLS: settings.TLS,
		PasswordSet: stored.PasswordSealed != "", Configured: configured,
		CACertSet: stored.CACertSealed != "", ClientCertSet: stored.ClientCertSealed != "",
		ClientKeySet: stored.ClientKeySealed != "",
	}
}
