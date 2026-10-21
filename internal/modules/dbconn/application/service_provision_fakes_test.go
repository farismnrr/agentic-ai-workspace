package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func provisionService(
	t *testing.T, storeOverride *memoryStore, envOverride *memoryEnvironment,
) (*Service, *memoryStore, *memoryEnvironment, *fakeProvisioner) {
	t.Helper()
	store := &memoryStore{records: map[string]StoredConnection{}}
	env := &memoryEnvironment{values: map[string]string{}}
	var actualStore Store = store
	var actualEnv Environment = env
	if storeOverride != nil {
		actualStore = storeOverride
	}
	if envOverride != nil {
		actualEnv = envOverride
	}
	provisioner := &fakeProvisioner{withTLS: true}
	service := NewService(actualStore, testCipher{}, fixedOwner{"project-a": "user-a"}, actualEnv,
		healthyProbe{}, provisioner)
	return service, store, env, provisioner
}

type fakeProvisioner struct {
	restored string
	calls    int
	withTLS  bool
}

func (*fakeProvisioner) Configured(context.Context) (bool, error) { return true, nil }
func (f *fakeProvisioner) Provision(context.Context, string) (ProvisionResult, error) {
	f.calls++
	result := ProvisionResult{
		Host: "db.local", Port: 5432, Database: "records", Role: "project",
		Password: fmt.Sprintf("generated-%d", f.calls),
	}
	if f.withTLS {
		result.TLS, result.SSLMode = true, "verify-full"
		result.CACert, result.ClientCert, result.ClientKey = "shared-ca", "shared-cert", "shared-key"
	}
	return result, nil
}

type failedCipher struct{}

var errCipherFailure = errors.New("cipher failed with generated-2")

func (failedCipher) Encrypt([]byte) (string, error) { return "", errCipherFailure }
func (failedCipher) Decrypt(value string) ([]byte, error) {
	return testCipher{}.Decrypt(value)
}

type blockingProvisioner struct {
	entered chan struct{}
	release chan struct{}
}

func (*blockingProvisioner) Configured(context.Context) (bool, error) { return true, nil }
func (p *blockingProvisioner) Provision(context.Context, string) (ProvisionResult, error) {
	close(p.entered)
	<-p.release
	return ProvisionResult{Host: "db.local", Port: 5432, Database: "records", Role: "project", Password: "one"}, nil
}
func (*blockingProvisioner) Restore(context.Context, string, string) error { return nil }
func (f *fakeProvisioner) Restore(_ context.Context, _, password string) error {
	f.restored = password
	return nil
}
