package runner

import (
	"context"
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

type fixedReach Granted

func (f fixedReach) For(context.Context, string) Granted { return Granted(f) }

func TestRunReachesWhatWasGrantedAndGitHubOnlyWhenConnected(t *testing.T) {
	ctx := context.Background()
	grant := fixedReach{Targets: []string{"db.internal:5432", "localhost:20128"}, Env: []string{"OPENAI_BASE_URL=x"}}

	hosts, granted := New("/w", nil).WithReach(grant).egressHosts(ctx, true)
	if !slices.Contains(hosts, "github.com") || !slices.Contains(hosts, "db.internal:5432") ||
		!slices.Contains(hosts, "localhost:20128") || len(granted.Env) != 1 {
		t.Errorf("hosts %v granted %+v", hosts, granted)
	}

	for name, run := range map[string]*Runner{
		"no reach wired":  New("/w", nil),
		"nothing granted": New("/w", nil).WithReach(fixedReach{}),
	} {
		if hosts, granted = run.egressHosts(ctx, false); len(hosts) != 0 || len(granted.Targets) != 0 {
			t.Errorf("%s: hosts %v granted %+v", name, hosts, granted)
		}
	}
}

func TestTargetsNeedTheProxySocketAndNamedHostsKeepTheirAddress(t *testing.T) {
	values := map[string]string{"DATABASE_URL": "postgres://db.internal:5432/app", "OTHER": "x"}
	grant := Granted{Targets: []string{"db.internal:5432"}, Env: []string{"OPENAI_BASE_URL=http://localhost:20128/v1"}}

	offline := sandbox.Spec{}
	got, env := grant.Apply(&offline, values)

	if len(offline.Tunnels) != 0 || env != nil || got["DATABASE_URL"] != values["DATABASE_URL"] {
		t.Errorf("without a proxy socket there is no door and no variable: %+v %v %v", offline, env, got)
	}

	online := sandbox.Spec{Egress: "/e"}
	got, env = grant.Apply(&online, values)

	if !slices.Equal(online.Tunnels, grant.Targets) || got["DATABASE_URL"] != values["DATABASE_URL"] || len(env) != 1 {
		t.Errorf("a name keeps its address, it resolves locally: %+v %v %v", online, env, got)
	}
}

func TestTargetsGivenAsIPAddressesArePointedAtLoopback(t *testing.T) {
	values := map[string]string{
		"DATABASE_URL": "postgres://10.0.0.5:5432/app?sslmode=require",
		"GATEWAY":      "http://10.0.0.9:20128/v1",
		"OTHER_DB":     "postgres://10.0.0.5:6000/app",
		"NOT_A_URL":    "plain",
	}
	spec := sandbox.Spec{Egress: "/e"}
	got, _ := Granted{Targets: []string{"10.0.0.5:5432", "10.0.0.9:20128"}}.Apply(&spec, values)

	wantDB, wantGateway := "postgres://127.0.0.1:5432/app?sslmode=require", "http://127.0.0.1:20128/v1"
	if got["DATABASE_URL"] != wantDB || got["GATEWAY"] != wantGateway {
		t.Errorf("each target is reached locally: %v", got)
	}

	if got["OTHER_DB"] != values["OTHER_DB"] || got["NOT_A_URL"] != "plain" {
		t.Errorf("other values stay as they are: %v", got)
	}

	if values["DATABASE_URL"] != "postgres://10.0.0.5:5432/app?sslmode=require" {
		t.Error("the caller's values must not be changed")
	}
}
