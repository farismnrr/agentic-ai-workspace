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

func TestWorkToolsRunThroughInMemoryMCPClients(t *testing.T) {
	for _, clientName := range []string{"ChatGPT", "Gemini"} {
		t.Run(clientName, func(t *testing.T) {
			runWorkToolsForClient(t, clientName)
		})
	}
}

//nolint:gocyclo,cyclop // setup, calls, and ordered assertions form one readable MCP lifecycle scenario.
func runWorkToolsForClient(t *testing.T, clientName string) {
	t.Helper()
	works := &fakeWorkSessions{}
	server := NewServer(fakeWorkspaces{}, Options{
		Gate: testGate{allowed: true}, Recorder: testRecorder{}, Works: works,
	})
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	token := &auth.TokenInfo{
		UserID: "u1", Scopes: []string{domain.ScopeRead}, Extra: map[string]any{clientKey: clientName},
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
	callResearchAndPlanning(t, session, workID)
	callRolesAndState(t, session, workID)
	if completeWork(t, session, workID)["task_status"] != "done" {
		t.Fatal("completed work task status is not done")
	}
	callLifecycleTool(t, session, "get_work_state", map[string]any{"work_id": workID})
	if works.lastClient != clientName || works.lastUser != "u1" ||
		fmt.Sprint(works.roles) != "[Researcher Planner Developer QA]" || works.research != "Finding" ||
		works.requirements != "Requirement" || works.state.State != "completed" {
		t.Fatalf("lifecycle identity, roles, documents, or state incorrect: %+v", works)
	}
}

func completeWork(t *testing.T, session *sdk.ClientSession, workID string) map[string]any {
	t.Helper()
	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "complete_work", Arguments: map[string]any{
		"work_id": workID, "outcome": "completed", "summary": "Ready",
	}})
	requireToolSuccess(t, result, err)
	requireWorkFields(t, result)
	content, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("complete response has no structured content: %#v", result.StructuredContent)
	}
	return content
}

func callResearchAndPlanning(t *testing.T, session *sdk.ClientSession, workID string) {
	t.Helper()
	callLifecycleTool(t, session, "set_role", map[string]any{"work_id": workID, "role": "Researcher"})
	callLifecycleTool(t, session, "save_research", map[string]any{"work_id": workID, "markdown": "Finding"})
	callLifecycleTool(t, session, "set_role", map[string]any{"work_id": workID, "role": "Planner"})
	callLifecycleTool(t, session, "update_plan", map[string]any{
		"work_id": workID, "base_revision": 0, "request_id": "request-123",
		"steps": []any{map[string]any{"title": "Build", "state": "pending"}},
	})
	callLifecycleTool(t, session, "save_requirements", map[string]any{"work_id": workID, "markdown": "Requirement"})
}

func callRolesAndState(t *testing.T, session *sdk.ClientSession, workID string) {
	t.Helper()
	callLifecycleTool(t, session, "set_role", map[string]any{"work_id": workID, "role": "Developer"})
	callLifecycleTool(t, session, "set_role", map[string]any{"work_id": workID, "role": "QA"})
	callLifecycleTool(t, session, "report_progress", map[string]any{"work_id": workID, "message": "Done"})
}

func callLifecycleTool(t *testing.T, session *sdk.ClientSession, name string, args map[string]any) {
	t.Helper()
	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	requireToolSuccess(t, result, err)
	requireWorkFields(t, result)
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
