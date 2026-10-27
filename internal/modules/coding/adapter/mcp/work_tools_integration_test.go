package mcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

//nolint:gocognit // the test is intentionally an ordered MCP tool lifecycle scenario.
func TestWorkToolsRunThroughInMemoryMCPClients(t *testing.T) {
	for _, clientName := range []string{"ChatGPT", "Gemini"} {
		t.Run(clientName, func(t *testing.T) {
			works := &fakeWorkSessions{}
			server := NewServer(fakeWorkspaces{}, Options{
				Gate: testGate{allowed: true}, Recorder: testRecorder{}, Works: works,
			})
			clientTransport, serverTransport := sdk.NewInMemoryTransports()
			token := &auth.TokenInfo{
				UserID: "u1", Scopes: []string{domain.ScopeRead},
				Extra: map[string]any{clientKey: clientName},
			}
			verified := verifiedTransport{Transport: serverTransport, token: token}
			if _, err := server.Connect(t.Context(), verified, nil); err != nil {
				t.Fatal(err)
			}
			client := sdk.NewClient(&sdk.Implementation{Name: clientName, Version: "test"}, nil)
			session, err := client.Connect(t.Context(), clientTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if closeErr := session.Close(); closeErr != nil {
					t.Errorf("close MCP client session: %v", closeErr)
				}
			})

			begin, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "begin_work", Arguments: map[string]any{
				"summary": "Build feature", "idempotency_key": "client-key-123",
			}})
			requireToolSuccess(t, begin, err)
			requireWorkFields(t, begin)
			workID := works.state.WorkID
			arguments := map[string]any{"work_id": workID}
			calls := []struct {
				args map[string]any
				name string
			}{
				{name: "set_role", args: map[string]any{"work_id": workID, "role": "Developer", "brief": "Implement"}},
				{name: "update_plan", args: map[string]any{
					"work_id": workID, "base_revision": 0,
					"request_id": "request-123", "steps": []any{map[string]any{"title": "Build", "state": "pending"}},
				}},
				{name: "save_research", args: map[string]any{"work_id": workID, "markdown": "Finding"}},
				{name: "save_requirements", args: map[string]any{"work_id": workID, "markdown": "Requirement"}},
				{name: "report_progress", args: map[string]any{"work_id": workID, "message": "Done"}},
				{name: "complete_work", args: map[string]any{"work_id": workID, "outcome": "completed", "summary": "Ready"}},
				{name: "get_work_state", args: arguments},
			}
			for _, call := range calls {
				t.Run(call.name, func(t *testing.T) {
					result, callErr := session.CallTool(t.Context(), &sdk.CallToolParams{Name: call.name, Arguments: call.args})
					requireToolSuccess(t, result, callErr)
					requireWorkFields(t, result)
				})
			}
			if works.lastClient != clientName || works.lastUser != "u1" {
				t.Fatalf("verified token identity = %q/%q", works.lastUser, works.lastClient)
			}
		})
	}
}

func requireWorkFields(t *testing.T, result *sdk.CallToolResult) {
	t.Helper()
	content, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("MCP response has no structured content: %#v", result.StructuredContent)
	}
	for _, key := range []string{"work_id", "revision", "state", "warnings"} {
		if _, ok := content[key]; !ok {
			t.Errorf("MCP response lacks %q: %#v", key, content)
		}
	}
}

type verifiedTransport struct {
	sdk.Transport
	token *auth.TokenInfo
}

func (t verifiedTransport) Connect(ctx context.Context) (sdk.Connection, error) {
	connection, err := t.Transport.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect verified transport: %w", err)
	}
	return verifiedConnection{Connection: connection, token: t.token}, nil
}

type verifiedConnection struct {
	sdk.Connection
	token *auth.TokenInfo
}

func (c verifiedConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	message, err := c.Connection.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("read verified MCP connection: %w", err)
	}
	if request, ok := message.(*jsonrpc.Request); ok {
		request.Extra = &sdk.RequestExtra{TokenInfo: c.token}
	}
	return message, nil
}

func requireToolSuccess(t *testing.T, result *sdk.CallToolResult, err error) {
	t.Helper()
	if err != nil || result == nil || result.IsError {
		message := ""
		if result != nil && len(result.Content) > 0 {
			if content, ok := result.Content[0].(*sdk.TextContent); ok {
				message = content.Text
			}
		}
		t.Fatalf("MCP call failed: %s (%+v), %v", message, result, err)
	}
}
