package bootstrap

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestSharedAdminURLPreservesReservedPassword(t *testing.T) {
	want := domain.SharedSettings{
		Kind: "postgres", TLS: "full",
		Host: "db.example", Port: 5432, Database: "workspace", User: "admin user",
		Password: strings.Join([]string{"p@", "ss:/?% ", `"雪`}, ""), SSLMode: "verify-full",
	}
	raw, err := sharedAdminURL(want)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if config.Password != want.Password || config.User != want.User || config.Database != want.Database ||
		parsed.Query().Get("sslmode") != want.SSLMode || config.Port != 5432 {
		t.Fatalf("connection fields did not round trip: %+v", config)
	}
}

func TestProvisionerFallsBackToEnvironmentURLOrReportsUnconfigured(t *testing.T) {
	settings := application.NewSharedSettingsService(emptySharedSettingsStore{}, sharedSettingsTestCipher{})
	fallbackURL := (&url.URL{
		Scheme: "postgres", User: url.UserPassword("admin", "secret"), Host: "db", Path: "workspace",
	}).String()
	fallback := callTimeProvisioner{settings: settings, fallbackURL: fallbackURL, ownPort: "8081"}
	if configured, err := fallback.Configured(t.Context()); err != nil || !configured {
		t.Fatal("environment admin URL was not used as fallback")
	}
	current, err := fallback.current(t.Context())
	configured, configErr := current.Configured(t.Context())
	if err != nil || configErr != nil || !configured {
		t.Fatalf("fallback provisioner not configured: %v", err)
	}
	missing := callTimeProvisioner{settings: settings, ownPort: "8081"}
	if configured, err := missing.Configured(t.Context()); err != nil || configured {
		t.Fatal("missing saved settings and environment fallback reported configured")
	}
}

type emptySharedSettingsStore struct{}

func (emptySharedSettingsStore) Get(context.Context, string) (application.StoredSharedSettings, error) {
	return application.StoredSharedSettings{}, application.ErrSharedSettingsMissing
}

func (emptySharedSettingsStore) Save(context.Context, string, application.StoredSharedSettings) error {
	return nil
}

type sharedSettingsTestCipher struct{}

func (sharedSettingsTestCipher) Encrypt(value []byte) (string, error) { return string(value), nil }

func (sharedSettingsTestCipher) Decrypt(value string) ([]byte, error) { return []byte(value), nil }
