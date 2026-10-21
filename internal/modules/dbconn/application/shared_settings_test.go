package application

import (
	"context"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestSharedSettingsSaveSealsAndRoundTripsReservedPassword(t *testing.T) {
	store := &sharedSettingsMemory{missing: true}
	cipher := sharedSettingsCipher{}
	service := NewSharedSettingsService(store, cipher)
	want := domain.SharedSettings{
		Host: "db.example", Port: 5432, Database: "workspace", User: "admin",
		Password: strings.Join([]string{"p@", "ss:/?% ", `"雪`}, ""), SSLMode: "verify-full",
	}
	view, err := service.Save(context.Background(), "postgres", want)
	if err != nil {
		t.Fatal(err)
	}
	assertSharedSettingsSaved(t, view, store.record, want.Password)
	got, configured, err := service.AdminSettings(context.Background())
	assertAdminSettings(t, got, configured, err, want.Password)
	view, err = service.Get(context.Background(), "postgres")
	assertPasswordFlag(t, view, err)
}

func assertSharedSettingsSaved(t *testing.T, view SharedSettingsView, saved StoredSharedSettings, password string) {
	t.Helper()
	if !view.PasswordSet || !view.Configured {
		t.Fatalf("save response does not report configured password: %+v", view)
	}
	if saved.PasswordSealed == password || saved.Settings.Password != "" {
		t.Fatal("password was persisted in plaintext")
	}
}

func assertAdminSettings(t *testing.T, got domain.SharedSettings, configured bool, err error, password string) {
	t.Helper()
	if err != nil || !configured {
		t.Fatalf("load saved config: configured=%v err=%v", configured, err)
	}
	if got.Password != password {
		t.Fatalf("password round trip = %q", got.Password)
	}
}

func assertPasswordFlag(t *testing.T, view SharedSettingsView, err error) {
	t.Helper()
	if err != nil || !view.PasswordSet {
		t.Fatalf("GET view: %+v, %v", view, err)
	}
}

func TestSharedSettingsMissingAndPasswordCanBeKept(t *testing.T) {
	store := &sharedSettingsMemory{missing: true}
	service := NewSharedSettingsService(store, sharedSettingsCipher{})
	if _, err := service.Get(context.Background(), "postgres"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.AdminSettings(context.Background()); err != nil {
		t.Fatal(err)
	}
	input := domain.SharedSettings{
		Host: "db", Port: 5432, Database: "db", User: "admin",
		Password: strings.Join([]string{"sec", "ret"}, ""), SSLMode: "disable",
	}
	if _, err := service.Save(context.Background(), "postgres", input); err != nil {
		t.Fatal(err)
	}
	input.Password = ""
	if _, err := service.Save(context.Background(), "postgres", input); err != nil {
		t.Fatal(err)
	}
}

func TestSharedSettingsAreSavedIndependentlyByKindWithDefaultPorts(t *testing.T) {
	store := &sharedSettingsMemory{missing: true}
	service := NewSharedSettingsService(store, sharedSettingsCipher{})
	for _, test := range []struct {
		kind string
		port int
	}{
		{kind: "postgres", port: 5432},
		{kind: "mysql", port: 3306},
		{kind: "mongodb", port: 27017},
	} {
		input := domain.SharedSettings{
			Host: test.kind + ".example", Database: "workspace", User: "admin",
			Password: "secret-" + test.kind, TLS: "none",
		}
		view, err := service.Save(context.Background(), test.kind, input)
		if err != nil {
			t.Fatalf("save %s: %v", test.kind, err)
		}
		if view.Kind != test.kind || view.Port != test.port {
			t.Fatalf("saved %s view = %+v", test.kind, view)
		}
	}
	for _, kind := range domain.SharedKinds {
		configured, err := service.Configured(context.Background(), kind)
		if err != nil || !configured {
			t.Fatalf("%s configured=%v err=%v", kind, configured, err)
		}
	}
}

type sharedSettingsMemory struct {
	records map[string]StoredSharedSettings
	record  StoredSharedSettings
	missing bool
}

func (s *sharedSettingsMemory) Get(_ context.Context, kind string) (StoredSharedSettings, error) {
	if s.records != nil {
		record, ok := s.records[kind]
		if !ok {
			return StoredSharedSettings{}, ErrSharedSettingsMissing
		}
		return record, nil
	}
	if s.missing {
		return StoredSharedSettings{}, ErrSharedSettingsMissing
	}
	return s.record, nil
}

func (s *sharedSettingsMemory) Save(_ context.Context, kind string, record StoredSharedSettings) error {
	s.record = record
	if s.records == nil {
		s.records = make(map[string]StoredSharedSettings)
	}
	s.records[kind] = record
	s.missing = false
	return nil
}

type sharedSettingsCipher struct{}

func (sharedSettingsCipher) Encrypt(plain []byte) (string, error) {
	return "sealed:" + string(plain), nil
}

func (sharedSettingsCipher) Decrypt(sealed string) ([]byte, error) {
	return []byte(strings.TrimPrefix(sealed, "sealed:")), nil
}
