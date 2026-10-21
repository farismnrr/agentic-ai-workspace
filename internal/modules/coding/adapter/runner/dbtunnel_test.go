package runner

import (
	"context"
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

type fixedTunnel struct {
	host string
	port int
	ok   bool
}

func (f fixedTunnel) Target(context.Context, string) (string, int, bool) { return f.host, f.port, f.ok }

func TestRunReachesItsDatabaseAndGitHubOnlyWhenEachIsGranted(t *testing.T) {
	ctx := context.Background()
	db := fixedTunnel{host: "db.internal", port: 5432, ok: true}

	hosts, tunnel := New("/w", nil).WithDBTunnel(db).egressHosts(ctx, true)
	if tunnel != "db.internal:5432" || !slices.Contains(hosts, tunnel) || !slices.Contains(hosts, "github.com") {
		t.Errorf("hosts %v tunnel %q", hosts, tunnel)
	}

	for name, run := range map[string]*Runner{
		"no tunnel wired":      New("/w", nil),
		"no database for this": New("/w", nil).WithDBTunnel(fixedTunnel{}),
	} {
		if hosts, tunnel = run.egressHosts(ctx, false); len(hosts) != 0 || tunnel != "" {
			t.Errorf("%s: hosts %v tunnel %q", name, hosts, tunnel)
		}
	}
}

func TestTunnelNeedsTheProxySocketAndKeepsNamedHostsAsSaved(t *testing.T) {
	values := map[string]string{"DATABASE_URL": "postgres://db.internal:5432/app", "OTHER": "x"}

	offline := sandbox.Spec{}
	got := UseTunnel(&offline, "db.internal:5432", values)
	if offline.Tunnel != "" || got["DATABASE_URL"] != values["DATABASE_URL"] {
		t.Errorf("without a proxy socket there is no door: %+v %v", offline, got)
	}

	online := sandbox.Spec{Egress: "/e"}
	got = UseTunnel(&online, "db.internal:5432", values)
	if online.Tunnel != "db.internal:5432" || got["DATABASE_URL"] != values["DATABASE_URL"] {
		t.Errorf("a name keeps its address, it resolves locally: %+v %v", online, got)
	}
}

func TestTunnelToAnIPAddressIsPointedAtLoopback(t *testing.T) {
	values := map[string]string{
		"DATABASE_URL": "postgres://10.0.0.5:5432/app?sslmode=require",
		"OTHER_DB":     "postgres://10.0.0.5:6000/app",
		"NOT_A_URL":    "plain",
	}
	spec := sandbox.Spec{Egress: "/e"}
	got := UseTunnel(&spec, "10.0.0.5:5432", values)

	if got["DATABASE_URL"] != "postgres://127.0.0.1:5432/app?sslmode=require" {
		t.Errorf("the saved database is reached locally: %q", got["DATABASE_URL"])
	}

	if got["OTHER_DB"] != values["OTHER_DB"] || got["NOT_A_URL"] != "plain" {
		t.Errorf("other values stay as they are: %v", got)
	}

	if values["DATABASE_URL"] != "postgres://10.0.0.5:5432/app?sslmode=require" {
		t.Error("the caller's values must not be changed")
	}
}
