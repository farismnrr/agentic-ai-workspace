package mcp

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestExternalWorkToolsAreDiscoverableByConnectedClients(t *testing.T) {
	server := NewServer(nil, Options{})
	want := []string{
		"begin_work", "set_role", "update_plan", "save_research", "save_requirements",
		"report_progress", "complete_work", "get_work_state",
	}
	for _, clientName := range []string{"ChatGPT", "Gemini"} {
		clientTransport, serverTransport := sdk.NewInMemoryTransports()
		if _, err := server.Connect(t.Context(), serverTransport, nil); err != nil {
			t.Fatal(err)
		}
		client := sdk.NewClient(&sdk.Implementation{Name: clientName, Version: "test"}, nil)
		session, err := client.Connect(t.Context(), clientTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		listed, err := session.ListTools(t.Context(), nil)
		_ = session.Close()
		if err != nil {
			t.Fatalf("%s list tools: %v", clientName, err)
		}
		for _, name := range want {
			if !slices.ContainsFunc(listed.Tools, func(tool *sdk.Tool) bool { return tool.Name == name }) {
				t.Errorf("%s did not receive %s", clientName, name)
			}
		}
	}
}

func TestWorkLifecycleGuardUsesExistingSwitchThenReadScope(t *testing.T) {
	for _, tc := range []workGuardCase{
		{name: "off", info: domain.ToolGetWorkState, scopes: []string{domain.ScopeRead}, wantError: "turned off"},
		{
			name: "scope", info: domain.ToolGetWorkState, allowed: true,
			scopes: []string{domain.ScopeWrite}, wantError: domain.ScopeRead,
		},
		{
			name: "role does not permit edits", info: domain.ToolEditFile, allowed: true,
			scopes: []string{domain.ScopeRead}, wantError: domain.ScopeWrite,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := testGate{allowed: tc.allowed}
			g := &guard{gate: gate, recorder: testRecorder{}}
			handler := guarded[workIDInput, WorkState](g, tc.info,
				func(workIDInput) string { return "state" }, func(context.Context, workIDInput) (WorkState, report, error) {
					t.Fatal("refused call ran")
					return WorkState{}, report{}, nil
				})
			result, _, err := handler(t.Context(), &sdk.CallToolRequest{
				Extra: &sdk.RequestExtra{TokenInfo: testToken(tc.scopes)},
			}, workIDInput{})
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("guard result = %+v, %v", result, err)
			}
			message := result.Content[0].(*sdk.TextContent).Text
			if !strings.Contains(message, tc.wantError) {
				t.Fatalf("error %q does not mention %q", message, tc.wantError)
			}
		})
	}
}

//nolint:govet // fieldalignment: keep test cases grouped by their purpose.
type workGuardCase struct {
	name      string
	info      domain.ToolInfo
	allowed   bool
	scopes    []string
	wantError string
}

type testGate struct{ allowed bool }

func (g testGate) Allowed(context.Context, string, string) (bool, error) { return g.allowed, nil }

type testRecorder struct{}

func (testRecorder) Record(context.Context, Entry) error { return nil }

func testToken(scopes []string) *auth.TokenInfo {
	return &auth.TokenInfo{UserID: "user-1", Scopes: scopes, Extra: map[string]any{clientKey: "ChatGPT"}}
}
