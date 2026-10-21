package application

import (
	"context"
	"strings"
)

// SharedTargets tells which database server an admin saved for a kind. It never carries a secret.
type SharedTargets interface {
	Get(ctx context.Context, kind string) (SharedSettingsView, error)
}

// Tunnels decides which database a project's sandbox may reach: the one server an admin saved in
// Settings, and only for a project whose own connection was made from it. A person who types that
// server's address into their own Env gets nothing, and neither does a project whose saved server
// has since been changed or removed.
type Tunnels struct {
	store    Store
	settings SharedTargets
}

// NewTunnels builds the check over the project connections and the admin's saved servers.
func NewTunnels(store Store, settings SharedTargets) *Tunnels {
	return &Tunnels{store: store, settings: settings}
}

// Target returns the host and port the project's sandbox may reach, and whether there is one.
func (t *Tunnels) Target(ctx context.Context, projectID string) (host string, port int, ok bool) {
	stored, err := t.store.Get(ctx, projectID)
	if err != nil || !stored.Connection.Managed {
		return "", 0, false
	}

	conn := stored.Connection

	saved, err := t.settings.Get(ctx, string(conn.Kind))
	if err != nil || !saved.Configured || !strings.EqualFold(saved.Host, conn.Host) || saved.Port != conn.Port {
		return "", 0, false
	}

	return strings.ToLower(conn.Host), conn.Port, true
}
