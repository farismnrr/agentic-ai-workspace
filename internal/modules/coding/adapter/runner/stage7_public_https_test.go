package runner_test

import (
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

// Docker-backed security gate: one real public HTTPS destination is permitted
// only while its own task-scoped grant is active. The host is a reserved RFC 2606
// example domain, never the production marimo or app host.
func TestStage7RealPublicHTTPSNeedsTaskGrant(t *testing.T) {
	if toolboxImage() == "" {
		t.Skip("Docker toolbox image required")
	}

	engine, grants := grantEngine(t)
	dir := t.TempDir()
	const host = "example.com"
	scope := egressproxy.Scope{Project: "p1", Task: "b"}
	if err := grants.Allow("stage7-https", scope, host, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	probe := "curl -sS -m 25 -o /dev/null -w 'STAGE7_CODE=%{http_code}' https://" + host + " 2>&1; true"
	if out := runAs(t, engine, dir, "a", probe); !strings.Contains(out, "403") {
		t.Fatalf("unapproved task must receive 403: %q", out)
	}

	out := runAs(t, engine, dir, "b", probe)
	if !strings.Contains(out, "STAGE7_CODE=200") {
		t.Fatalf("approved task must complete public TLS/HTTPS request: %q", out)
	}

	grants.Revoke("stage7-https")
	if out = runAs(t, engine, dir, "b", probe); !strings.Contains(out, "403") {
		t.Fatalf("revoke must refuse the next CONNECT: %q", out)
	}
}
