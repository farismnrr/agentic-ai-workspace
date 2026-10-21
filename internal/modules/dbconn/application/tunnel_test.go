package application

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

var errUnavailable = errors.New("unavailable")

type tunnelStore struct{ conns map[string]StoredConnection }

func (s tunnelStore) Get(_ context.Context, id string) (StoredConnection, error) {
	if c, ok := s.conns[id]; ok {
		return c, nil
	}

	return StoredConnection{}, ErrNotFound
}
func (tunnelStore) Save(context.Context, string, StoredConnection) error { return nil }
func (tunnelStore) Delete(context.Context, string) error                 { return nil }

type tunnelSettings map[string]SharedSettingsView

func (s tunnelSettings) Get(_ context.Context, kind string) (SharedSettingsView, error) {
	if v, ok := s[kind]; ok {
		return v, nil
	}

	return SharedSettingsView{}, errUnavailable
}

func managed(kind domain.Kind, host string, port int) StoredConnection {
	return StoredConnection{Connection: domain.Connection{Kind: kind, Host: host, Port: port, Managed: true}}
}

func TestTunnelOnlyForAManagedConnectionToTheSavedServer(t *testing.T) {
	saved := tunnelSettings{"postgres": {Configured: true, Host: "DB.internal", Port: 5432}}
	unmanaged := managed(domain.Postgres, "db.internal", 5432)
	unmanaged.Connection.Managed = false

	store := tunnelStore{conns: map[string]StoredConnection{
		"ok":        managed(domain.Postgres, "db.internal", 5432),
		"typed":     unmanaged,
		"moved":     managed(domain.Postgres, "old.internal", 5432),
		"otherport": managed(domain.Postgres, "db.internal", 6432),
		"mysql":     managed(domain.MySQL, "db.internal", 5432),
	}}
	tunnels := NewTunnels(store, saved)

	if host, port, ok := tunnels.Target(context.Background(), "ok"); !ok || host != "db.internal" || port != 5432 {
		t.Errorf("a managed connection gets its server: %q %d %v", host, port, ok)
	}

	for _, project := range []string{"typed", "moved", "otherport", "mysql", "none"} {
		if _, _, ok := tunnels.Target(context.Background(), project); ok {
			t.Errorf("%s must get no tunnel", project)
		}
	}

	if _, _, ok := NewTunnels(store, tunnelSettings{}).Target(context.Background(), "ok"); ok {
		t.Error("without saved settings there is no tunnel")
	}

	removed := tunnelSettings{"postgres": {Configured: false, Host: "db.internal", Port: 5432}}
	if _, _, ok := NewTunnels(store, removed).Target(context.Background(), "ok"); ok {
		t.Error("a removed server leaves no tunnel")
	}
}
