package application

import (
	"context"
	"fmt"
)

func (s *SharedSettingsService) previous(ctx context.Context) (StoredSharedSettings, error) {
	stored, err := s.store.Get(ctx)
	if err != nil {
		return StoredSharedSettings{}, fmt.Errorf("load previous shared database settings: %w", err)
	}
	return stored, nil
}

func retainedPEM(s *SharedSettingsService, submitted, saved string, remove bool) (string, error) {
	if remove {
		return "", nil
	}
	if submitted != "" {
		return submitted, nil
	}
	if saved == "" {
		return "", nil
	}
	plain, err := s.cipher.Decrypt(saved)
	if err != nil {
		return "", fmt.Errorf("open saved database certificate: %w", err)
	}
	return string(plain), nil
}

func sealOptional(cipher Cipher, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	sealed, err := cipher.Encrypt([]byte(value))
	if err != nil {
		return "", fmt.Errorf("seal database certificate: %w", err)
	}
	return sealed, nil
}

func (s *SharedSettingsService) openCertificates(stored *StoredSharedSettings) error {
	fields := []struct {
		target *string
		sealed string
	}{
		{&stored.Settings.CACert, stored.CACertSealed},
		{&stored.Settings.ClientCert, stored.ClientCertSealed},
		{&stored.Settings.ClientKey, stored.ClientKeySealed},
	}
	for _, field := range fields {
		if field.sealed == "" {
			continue
		}
		plain, err := s.cipher.Decrypt(field.sealed)
		if err != nil {
			return fmt.Errorf("open saved database certificate: %w", err)
		}
		*field.target = string(plain)
	}
	return nil
}
