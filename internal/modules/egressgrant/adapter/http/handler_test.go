package http

import (
	"net/http"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func TestApprovalSessionCSRFOriginReplayAndTTL(t *testing.T) {
	app, s := testApp(t)
	path := "/api/projects/p1/egress-grants/g1/approve"
	token := issue(t, app, "owner")
	for _, tc := range []struct {
		user, origin, token, body string
		code                      int
	}{
		{"owner", "https://app.example", "", "{}", 403},
		{"owner", "https://evil.example", token, "{}", 403},
		{"other", "https://app.example", token, "{}", 404},
		{"owner", "https://app.example", token, `{"ttlHours":169}`, 400},
		{"owner", "https://app.example", token, `{"ttlHours":24}`, 200},
		{"owner", "https://app.example", token, "{}", 409},
	} {
		resp := send(t, app, "POST", path, tc.user, tc.origin, tc.token, tc.body)
		if resp.code != tc.code {
			t.Fatalf("got status %d want %d", resp.code, tc.code)
		}
	}
	if s.grant.Status != domain.StatusApproved {
		t.Fatal("grant was not approved")
	}
	if resp := send(t, app, "POST", "/api/projects/p1/egress-grants/g1/revoke", "owner",
		"https://app.example", token, ""); resp.code != http.StatusOK {
		t.Fatalf("revoke %d", resp.code)
	}
}

func TestApprovalAdminAndManualRequestScope(t *testing.T) {
	app, s := testApp(t)
	token := issue(t, app, "admin")
	if send(t, app, "POST", "/api/projects/p1/egress-grants/g1/deny", "admin",
		"https://app.example", token, "").code != http.StatusOK {
		t.Fatal("admin denied")
	}
	if s.grant.DecidedBy != "admin" {
		t.Fatal("actor identity not server derived")
	}
	token = issue(t, app, "owner")
	resp := send(t, app, "POST", "/api/projects/p1/egress-grants/", "owner", "https://app.example", token,
		`{"taskId":"t1","host":"safe.example.com","reason":"manual","actor":"admin","role":"admin","project_id":"p2"}`)
	if resp.code != http.StatusCreated || s.grant.ProjectID != "p1" || s.grant.RequestedBy.ID != "owner" {
		t.Fatalf("request status %d scope %+v", resp.code, s.grant)
	}
}
