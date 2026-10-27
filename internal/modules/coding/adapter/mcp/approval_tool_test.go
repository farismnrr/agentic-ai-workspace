package mcp

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestApprovePlanRequiresItsSwitchAndDedicatedScope(t *testing.T) {
	for _, test := range []struct {
		name    string
		want    string
		scopes  []string
		allowed bool
	}{
		{name: "switch off", scopes: []string{domain.ScopeApprove}, want: "turned off"},
		{name: "read scope only", allowed: true, scopes: []string{domain.ScopeRead}, want: domain.ScopeApprove},
		{name: "approved", allowed: true, scopes: []string{domain.ScopeApprove}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, works := callApproveTool(t, test.allowed, test.scopes)
			if test.want != "" {
				assertApprovalRefused(t, result, works, test.want)
				return
			}
			assertApprovalAllowed(t, result, works)
		})
	}
}

func assertApprovalRefused(t *testing.T, result *sdk.CallToolResult, works *fakeWorkSessions, want string) {
	t.Helper()
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*sdk.TextContent).Text, want) {
		t.Fatalf("refused approval result = %+v, want %q", result, want)
	}
	if works.approvalCalls != 0 {
		t.Fatal("refused approval reached the use case")
	}
}

func assertApprovalAllowed(t *testing.T, result *sdk.CallToolResult, works *fakeWorkSessions) {
	t.Helper()
	if result == nil || result.IsError || works.approvalCalls != 1 || works.lastUser != "u1" ||
		works.lastClient != "ChatGPT" || works.lastProject != "project-1" {
		t.Fatalf("approval identity/result = %+v works=%+v", result, works)
	}
	body, ok := result.StructuredContent.(map[string]any)
	if !ok || body["decision"] != "approved" || body["handoff_id"] != "handoff-1" ||
		body["internal_task_id"] != "internal-1" {
		t.Fatalf("approval result omitted handoff reference: %#v", result.StructuredContent)
	}
}

func callApproveTool(t *testing.T, allowed bool, scopes []string) (*sdk.CallToolResult, *fakeWorkSessions) {
	t.Helper()
	works := &fakeWorkSessions{}
	server := NewServer(fakeWorkspaces{}, Options{
		Gate: testGate{allowed: allowed}, Recorder: testRecorder{}, Works: works,
	})
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	token := &auth.TokenInfo{UserID: "u1", Scopes: scopes, Extra: map[string]any{clientKey: "ChatGPT"}}
	transport := verifiedTransport{Transport: serverTransport, token: token}
	if _, err := server.Connect(t.Context(), transport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "test"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "approve_plan", Arguments: map[string]any{
		"work_id": "work-1", "revision": 2, "idempotency_key": "approval-key-123",
	}})
	if err != nil {
		t.Fatal(err)
	}
	return result, works
}
