package application

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

type provisionerMap map[domain.Kind]Provisioner

func (m provisionerMap) For(kind domain.Kind) Provisioner { return m[kind] }

func TestProvisionSwitchesKindsAndReconcilesOnlyManagedEnvironment(t *testing.T) {
	store := &memoryStore{records: map[string]StoredConnection{}}
	env := &memoryEnvironment{values: map[string]string{}}
	postgres, mysql, mongo := &fakeProvisioner{withTLS: true},
		&fakeProvisioner{withTLS: true}, &fakeProvisioner{withTLS: true}
	service := NewServiceWithProvisioners(store, testCipher{}, fixedOwner{"project-a": "user-a"}, env,
		healthyProbe{}, provisionerMap{
			domain.Postgres: postgres, domain.MySQL: mysql, domain.MongoDB: mongo,
		})
	for _, kind := range []domain.Kind{domain.Postgres, domain.MySQL, domain.MongoDB} {
		assertKindProvisioned(t, service, env, kind)
	}
	assertFinalMongoEnvironment(t, env)
}

func assertKindProvisioned(t *testing.T, service *Service, env *memoryEnvironment, kind domain.Kind) {
	t.Helper()
	metadata, err := service.Provision(context.Background(), "user-a", "project-a", kind)
	if err != nil {
		t.Fatalf("provision %s: %v", kind, err)
	}
	if metadata.Choice != string(kind) || !metadata.Managed {
		t.Fatalf("metadata for %s = %#v", kind, metadata)
	}
	urlName := domain.DefaultEnvName(kind)
	value := env.values["project-a:"+urlName]
	assertManagedDatabaseURL(t, kind, value)
	if value == "" || env.values["project-a:DATABASE_SSL_CA"] != "shared-ca" {
		t.Fatalf("managed environment missing for %s: %#v", kind, env.values)
	}
}

func assertManagedDatabaseURL(t *testing.T, kind domain.Kind, value string) {
	t.Helper()
	parsed, parseErr := url.Parse(value)
	if parseErr != nil || parsed.Scheme != string(kind) || parsed.User == nil || parsed.User.Username() != "project" {
		t.Fatalf("managed %s URL is incorrect: %q", kind, value)
	}
	if kind == domain.MongoDB && parsed.Query().Get("authSource") != "records" {
		t.Fatalf("MongoDB authSource is incorrect: %q", value)
	}
}

func assertFinalMongoEnvironment(t *testing.T, env *memoryEnvironment) {
	t.Helper()
	if _, exists := env.values["project-a:DATABASE_URL"]; exists {
		t.Fatal("switch to MongoDB retained the SQL URL")
	}
	if env.values["project-a:MONGODB_URL"] == "" {
		t.Fatal("MongoDB URL was not written")
	}
	for _, name := range []string{"DATABASE_SSL_CERT", "DATABASE_SSL_KEY"} {
		if _, exists := env.values["project-a:"+name]; exists {
			t.Fatalf("managed project received administrator %s", name)
		}
	}
}

func TestProvisionKindsProtectUserURLsAndRefuseUnconfiguredKind(t *testing.T) {
	for _, name := range []string{"DATABASE_URL", "MONGODB_URL"} {
		t.Run(name, func(t *testing.T) {
			env := &memoryEnvironment{values: map[string]string{"project-a:" + name: "user-owned"}}
			service := NewServiceWithProvisioners(&memoryStore{records: map[string]StoredConnection{}}, testCipher{},
				fixedOwner{"project-a": "user-a"}, env, healthyProbe{}, provisionerMap{
					domain.Postgres: &fakeProvisioner{}, domain.MySQL: &fakeProvisioner{},
					domain.MongoDB: &fakeProvisioner{},
				})
			_, err := service.Provision(context.Background(), "user-a", "project-a", domain.MongoDB)
			var failure ProvisionError
			if !errors.As(err, &failure) || failure.Category != EnvironmentNameInUse {
				t.Fatalf("user-owned %s was not protected: %v", name, err)
			}
		})
	}
	service := NewServiceWithProvisioners(&memoryStore{records: map[string]StoredConnection{}}, testCipher{},
		fixedOwner{"project-a": "user-a"}, &memoryEnvironment{values: map[string]string{}},
		healthyProbe{}, provisionerMap{})
	_, err := service.Provision(context.Background(), "user-a", "project-a", domain.MongoDB)
	var failure ProvisionError
	if !errors.As(err, &failure) || failure.Category != KindNotConfigured {
		t.Fatalf("unconfigured kind error = %v", err)
	}
}

func TestProvisionSwitchFailureRestoresConnectionAndEnvironment(t *testing.T) {
	store := &memoryStore{records: map[string]StoredConnection{}}
	env := &memoryEnvironment{values: map[string]string{}}
	postgres, mysql := &fakeProvisioner{withTLS: true}, &fakeProvisioner{withTLS: true}
	service := NewServiceWithProvisioners(store, testCipher{}, fixedOwner{"project-a": "user-a"}, env,
		healthyProbe{}, provisionerMap{domain.Postgres: postgres, domain.MySQL: mysql})
	if _, err := service.Provision(context.Background(), "user-a", "project-a", domain.Postgres); err != nil {
		t.Fatal(err)
	}
	oldURL := env.values["project-a:DATABASE_URL"]
	oldSecret := store.records["project-a"].PasswordSealed
	env.failSet = true
	if _, err := service.Provision(context.Background(), "user-a", "project-a", domain.MySQL); err == nil {
		t.Fatal("expected environment write failure")
	}
	if store.records["project-a"].Connection.Kind != domain.Postgres ||
		store.records["project-a"].PasswordSealed != oldSecret ||
		env.values["project-a:DATABASE_URL"] != oldURL {
		t.Fatal("failed kind switch did not restore previous local state")
	}
	if _, exists := env.values["project-a:MONGODB_URL"]; exists {
		t.Fatal("rollback changed an unrelated MongoDB URL")
	}
}

func TestGetReportsSQLiteAndLegacyChoices(t *testing.T) {
	service := NewServiceWithProvisioners(&memoryStore{records: map[string]StoredConnection{}}, testCipher{},
		fixedOwner{"project-a": "user-a"}, &memoryEnvironment{values: map[string]string{}}, healthyProbe{}, nil)
	metadata, err := service.Get(context.Background(), "user-a", "project-a")
	if err != nil || metadata.Choice != "sqlite" {
		t.Fatalf("SQLite choice = %#v, %v", metadata, err)
	}
	service.store = &memoryStore{records: map[string]StoredConnection{"project-a": {
		Connection: domain.Connection{Kind: domain.MySQL, Managed: false, EnvName: "DATABASE_URL"},
	}}}
	metadata, err = service.Get(context.Background(), "user-a", "project-a")
	if err != nil || metadata.Choice != "custom" || !metadata.ReadOnly || metadata.Managed {
		t.Fatalf("legacy choice = %#v, %v", metadata, err)
	}
}
