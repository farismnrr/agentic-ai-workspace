package mcpbridge_test

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/mcpbridge"
)

// connect serves one tool and returns a program's session with it.
func connect(t *testing.T) *sdk.ClientSession {
	t.Helper()

	dir := t.TempDir()
	tools := []mcpbridge.Tool{{
		Name: "delegate", Description: "Hand a job over.",
		Schema: json.RawMessage(`{"type":"object","properties":{"brief":{"type":"string"}}}`),
	}}

	answer := func(_ context.Context, name string, in json.RawMessage) (string, bool) {
		return name + " got " + string(in), strings.Contains(string(in), "bad")
	}

	stop, err := mcpbridge.Serve(t.Context(), dir, tools, answer)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(stop)

	conn, err := (&net.Dialer{}).DialContext(t.Context(), "unix", filepath.Join(dir, mcpbridge.SocketName))
	if err != nil {
		t.Fatal(err)
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "program", Version: "1"}, nil)

	session, err := client.Connect(t.Context(), &sdk.IOTransport{Reader: conn, Writer: conn}, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func TestAProgramCanListTheTools(t *testing.T) {
	t.Parallel()

	listed, err := connect(t).ListTools(t.Context(), nil)
	if err != nil || len(listed.Tools) != 1 || listed.Tools[0].Name != "delegate" {
		t.Fatalf("tools = %+v, %v", listed, err)
	}
}

func TestAProgramGetsWhatTheServerAnswersAndWhetherItFailed(t *testing.T) {
	t.Parallel()

	session := connect(t)

	for brief, wantFailed := range map[string]bool{"fine": false, "bad": true} {
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name: "delegate", Arguments: map[string]any{"brief": brief},
		})
		if err != nil || result.IsError != wantFailed {
			t.Fatalf("%s: %+v, %v", brief, result, err)
		}

		if text, _ := result.Content[0].(*sdk.TextContent); !strings.Contains(text.Text, brief) {
			t.Fatalf("answer %q lacks the arguments", text.Text)
		}
	}
}

func TestAFolderWithATooLongPathIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := mcpbridge.Serve(t.Context(), filepath.Join(t.TempDir(), strings.Repeat("x", 120)), nil, nil); err == nil {
		t.Fatal("want an error")
	}
}
