package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestProvisionSharedCreatesAndReconcilesManagedConnection(t *testing.T) {
	service, store, env, provisioner := provisionService(t, nil, nil)
	first, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	stored := store.records["project-a"]
	assertReconciledConnection(t, first, second, stored, provisioner.calls)
	assertMetadataAndEnvironmentAreSafe(t, second, env.values["project-a:DATABASE_URL"])
}

func assertReconciledConnection(
	t *testing.T, first, second domain.Metadata, stored StoredConnection, calls int,
) {
	t.Helper()
	if !first.Managed || !second.Managed || stored.Connection.Password != "" {
		t.Fatalf("shared connection was not reconciled safely: %#v", stored)
	}
	if calls != 2 {
		t.Fatalf("provisioner calls = %d, want 2", calls)
	}
	if stored.PasswordSealed != "sealed:generated-2" ||
		!stored.Connection.TLS {
		t.Fatalf("connection was not updated: %#v", stored.Connection)
	}
}

func assertMetadataAndEnvironmentAreSafe(t *testing.T, second domain.Metadata, databaseURL string) {
	t.Helper()
	if !strings.Contains(databaseURL, "generated-2") {
		t.Fatal("environment did not receive the rotated password")
	}
	encoded, _ := json.Marshal(second)
	if strings.Contains(string(encoded), "generated-") || strings.Contains(databaseURL, "admin") {
		t.Fatal("metadata exposed a password or admin URL")
	}
}

func TestProvisionSharedOwnerOnlyAndNotConfigured(t *testing.T) {
	service, _, _, _ := provisionService(t, nil, nil)
	if _, err := service.ProvisionShared(context.Background(), "user-b", "project-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-owner provision error = %v", err)
	}
	service = NewService(nil, nil, fixedOwner{"project-a": "user-a"}, nil, nil)
	_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	var failure ProvisionError
	if !errors.As(err, &failure) || failure.Category != AdminNotConfigured {
		t.Fatalf("unconfigured admin error = %v", err)
	}
}

func TestProvisionSharedRefusesToReplaceLegacyConnection(t *testing.T) {
	manual := StoredConnection{Connection: domain.Connection{
		Kind: domain.Postgres, Host: "db.local", Port: 5432, Database: "records",
		User: "manual", Password: "", EnvName: "DATABASE_URL",
	}, PasswordSealed: "sealed:manual"}
	store := &memoryStore{records: map[string]StoredConnection{"project-a": manual}}
	service, _, _, provisioner := provisionService(t, store, &memoryEnvironment{values: map[string]string{}})
	_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	var failure ProvisionError
	if !errors.As(err, &failure) || failure.Category != ManualConnectionExists || provisioner.calls != 0 {
		t.Fatalf("manual connection was not preserved: err=%v calls=%d", err, provisioner.calls)
	}
}

func TestProvisionSharedRefusesExistingDatabaseURLVariableWithoutManagedConnection(t *testing.T) {
	env := &memoryEnvironment{values: map[string]string{"project-a:DATABASE_URL": "postgres://manual"}}
	service, _, _, provisioner := provisionService(t, &memoryStore{records: map[string]StoredConnection{}}, env)
	_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	var failure ProvisionError
	if !errors.As(err, &failure) || failure.Category != EnvironmentNameInUse || provisioner.calls != 0 {
		t.Fatalf("existing DATABASE_URL was not preserved: err=%v calls=%d", err, provisioner.calls)
	}
}

func TestProvisionSharedRefusesAndPreservesUserOwnedSSLKey(t *testing.T) {
	env := &memoryEnvironment{values: map[string]string{"project-a:DATABASE_SSL_KEY": "user-key"}}
	service, _, _, provisioner := provisionService(t, &memoryStore{records: map[string]StoredConnection{}}, env)
	_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	var failure ProvisionError
	if !errors.As(err, &failure) || failure.Category != EnvironmentNameInUse || provisioner.calls != 0 {
		t.Fatalf("existing DATABASE_SSL_KEY was not protected: err=%v calls=%d", err, provisioner.calls)
	}
	if got := env.values["project-a:DATABASE_SSL_KEY"]; got != "user-key" {
		t.Fatalf("user-owned DATABASE_SSL_KEY = %q, want preserved value", got)
	}
}

func TestProvisionSharedCompensatesSaveAndEnvironmentFailures(t *testing.T) {
	for _, fail := range []string{"store", "env"} {
		t.Run(fail, func(t *testing.T) {
			store := &memoryStore{records: map[string]StoredConnection{}}
			env := &memoryEnvironment{values: map[string]string{}}
			service, _, _, provisioner := provisionService(t, store, env)
			if _, err := service.ProvisionShared(context.Background(), "user-a", "project-a"); err != nil {
				t.Fatal(err)
			}
			old := store.records["project-a"]
			oldEnv := env.values["project-a:DATABASE_URL"]
			store.failSave = fail == "store"
			env.failSet = fail == "env"
			_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
			if err == nil || strings.Contains(err.Error(), "generated-") {
				t.Fatalf("unsafe or missing failure: %v", err)
			}
			if store.records["project-a"].PasswordSealed != old.PasswordSealed ||
				env.values["project-a:DATABASE_URL"] != oldEnv || provisioner.restored != "generated-1" {
				t.Fatal("failed rotation did not restore the previous connection, secret and remote password")
			}
		})
	}
}

func TestProvisionSharedCompensatesCipherFailureAndStopsOnSnapshotFailure(t *testing.T) {
	service, store, env, provisioner := provisionService(t, nil, nil)
	if _, err := service.ProvisionShared(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	old := store.records["project-a"]
	oldEnv := env.values["project-a:DATABASE_URL"]
	service.cipher = failedCipher{}
	_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	if err == nil || store.records["project-a"].PasswordSealed != old.PasswordSealed ||
		env.values["project-a:DATABASE_URL"] != oldEnv || provisioner.restored != "generated-1" {
		t.Fatal("cipher failure did not compensate the remote and local state")
	}

	env.failValues = true
	provisioner.calls = 0
	provisioner.restored = ""
	_, err = service.ProvisionShared(context.Background(), "user-a", "project-a")
	if err == nil || provisioner.calls != 0 {
		t.Fatal("failed Env snapshot should stop before remote provisioning")
	}
}

func TestDeleteNeverCallsSharedProvisioner(t *testing.T) {
	legacy := StoredConnection{Connection: domain.Connection{
		Kind: domain.Postgres, Host: "db.local", Port: 5432, Database: "records",
		User: "manual", EnvName: "DATABASE_URL",
	}, PasswordSealed: "sealed:pw"}
	store := &memoryStore{records: map[string]StoredConnection{"project-a": legacy}}
	service, _, _, provisioner := provisionService(t, store, &memoryEnvironment{
		values: map[string]string{"project-a:DATABASE_URL": legacy.Connection.URL()},
	})
	if err := service.Delete(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	if provisioner.calls != 0 || provisioner.restored != "" {
		t.Fatal("deleting a connection called the shared provisioner")
	}
}

func TestProvisionSharedRejectsConcurrentRunForProject(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	provisioner := &blockingProvisioner{entered: entered, release: release}
	service := NewService(&memoryStore{records: map[string]StoredConnection{}}, testCipher{},
		fixedOwner{"project-a": "user-a"}, &memoryEnvironment{values: map[string]string{}},
		healthyProbe{}, provisioner)
	result := make(chan error, 1)
	go func() {
		_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
		result <- err
	}()
	<-entered
	_, err := service.ProvisionShared(context.Background(), "user-a", "project-a")
	var failure ProvisionError
	if !errors.As(err, &failure) || failure.Category != ProvisionInProgress {
		t.Fatalf("concurrent run error = %v", err)
	}
	err = service.Delete(context.Background(), "user-a", "project-a")
	if !errors.As(err, &failure) || failure.Category != ProvisionInProgress {
		t.Fatalf("concurrent delete error = %v", err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
