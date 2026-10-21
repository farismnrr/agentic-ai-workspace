package sandbox

import (
	"slices"
	"testing"
)

func TestTunnelTargetAcceptsOnlyAHostAndAPort(t *testing.T) {
	good := map[string]string{
		"db.internal:5432": "db.internal", "DB.Internal:5432": "db.internal", "10.0.0.5:3306": "10.0.0.5",
	}
	for in, host := range good {
		if got, _, ok := tunnelTarget(Spec{Tunnel: in}); !ok || got != host {
			t.Errorf("%q: host %q ok %v, want %q", in, got, ok, host)
		}
	}

	// Anything that could add a socat option or reach another place is refused.
	for _, in := range []string{
		"", "db", "db:0", "db:65536", "db:x", ":5432", "db,fork:5432", "db:5432,fork", "a b:5432", "[::1]:5432", "-x:5432",
	} {
		if _, _, ok := tunnelTarget(Spec{Tunnel: in}); ok {
			t.Errorf("%q must be refused", in)
		}
	}
}

func TestTunnelNamesResolveToTheLocalDoorOnly(t *testing.T) {
	cases := map[string][]string{
		"db.internal:5432":  {"db.internal:127.0.0.1"},
		"localhost:5432":    nil,
		"192.168.1.5:5432":  nil,
		"db.internal:5432,": nil,
	}
	for in, want := range cases {
		if got := tunnelHosts(Spec{Tunnel: in}); !slices.Equal(got, want) {
			t.Errorf("%q: %v, want %v", in, got, want)
		}
	}

	if got := containerEnv(Spec{Tunnel: "db.internal:5432"}); !slices.Equal(got, []string{"DB_TUNNEL=db.internal:5432"}) {
		t.Errorf("env: %v", got)
	}

	if got := containerEnv(Spec{Env: []string{"DB_TUNNEL=evil:1", "A=1"}}); !slices.Equal(got, []string{"A=1"}) {
		t.Errorf("a value set from outside must not open a door: %v", got)
	}
}

func TestATunnelKeepsTheContainerWithoutANetwork(t *testing.T) {
	d := &Docker{}
	spec := Spec{Workspace: "/w", Egress: "/e", Tunnel: "db.internal:5432", Env: []string{"A=1"}, Argv: []string{"true"}}
	opts := d.createOptions(spec, nil)

	if opts.HostConfig.NetworkMode != "none" || !opts.Config.NetworkDisabled || !opts.HostConfig.ReadonlyRootfs {
		t.Fatalf("the container must stay offline and read-only: %+v", opts.HostConfig)
	}

	if !slices.Equal(opts.HostConfig.ExtraHosts, []string{"db.internal:127.0.0.1"}) {
		t.Errorf("hosts: %v", opts.HostConfig.ExtraHosts)
	}

	if !slices.Contains(opts.Config.Env, "DB_TUNNEL=db.internal:5432") || !slices.Contains(opts.Config.Env, "A=1") {
		t.Errorf("env: %v", opts.Config.Env)
	}

	if len(spec.Env) != 1 {
		t.Errorf("the caller's environment must not be changed: %v", spec.Env)
	}

	if opts.Config.Cmd[0] != Entry {
		t.Errorf("the start script must open the door: %v", opts.Config.Cmd)
	}
}
