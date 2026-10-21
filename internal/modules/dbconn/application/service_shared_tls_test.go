package application

import (
	"context"
	"strings"
	"testing"
)

func TestProvisionSharedWritesSelectedSSLMaterial(t *testing.T) {
	service, _, env, _ := provisionService(t, nil, nil)
	if _, err := service.ProvisionShared(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	if env.values["project-a:DATABASE_SSL_CA"] != "shared-ca" {
		t.Fatalf("shared CA Env value missing: %#v", env.values)
	}
	for _, name := range []string{"DATABASE_SSL_CERT", "DATABASE_SSL_KEY"} {
		if _, exists := env.values["project-a:"+name]; exists {
			t.Fatalf("admin client credential %s was exposed to project Env", name)
		}
	}
	if !strings.Contains(env.values["project-a:DATABASE_URL"], "sslmode=verify-full") {
		t.Fatalf("selected sslmode missing from DATABASE_URL: %q", env.values["project-a:DATABASE_URL"])
	}
}

func TestProvisionSharedClearsStaleSSLEnvironmentWhenMaterialIsAbsent(t *testing.T) {
	service, _, env, provisioner := provisionService(t, nil, nil)
	if _, err := service.ProvisionShared(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	env.values["project-a:DATABASE_SSL_CA"] = "old-ca"
	env.values["project-a:DATABASE_SSL_CERT"] = "old-cert"
	env.values["project-a:DATABASE_SSL_KEY"] = "old-key"
	provisioner.withTLS = false
	if _, err := service.ProvisionShared(context.Background(), "user-a", "project-a"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DATABASE_SSL_CA", "DATABASE_SSL_CERT", "DATABASE_SSL_KEY"} {
		if _, exists := env.values["project-a:"+name]; exists {
			t.Fatalf("stale %s was retained", name)
		}
	}
}
