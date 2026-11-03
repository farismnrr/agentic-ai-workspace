package mcpbridge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/mcpbridge"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestScreenshotToolReturnsImageContentThroughMCP(t *testing.T) {
	const png = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0bIDAT\x08\xd7c`\x00\x02\x00\x00\x05\x00\x01\xa5\xf6E\x40\x00\x00\x00\x00IEND\xaeB`\x82"
	image := []byte(png)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop, err := mcpbridge.Serve(ctx, dir, []mcpbridge.Tool{{Name: "read_browser_screenshot", Schema: json.RawMessage(`{"type":"object"}`)}}, func(context.Context, string, json.RawMessage) mcpbridge.Result {
		return mcpbridge.Result{Text: "Screenshot ready", Media: []mcpbridge.Media{{MIMEType: "image/png", Data: image}}}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	conn, err := net.DialTimeout("unix", filepath.Join(dir, mcpbridge.SocketName), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.IOTransport{Reader: conn, Writer: conn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "read_browser_screenshot", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content) != 2 {
		t.Fatalf("got %d content blocks, want text and image", len(result.Content))
	}
	if got := result.Content[0].(*sdk.TextContent).Text; got != "Screenshot ready" {
		t.Fatalf("text = %q", got)
	}
	got, ok := result.Content[1].(*sdk.ImageContent)
	if !ok {
		t.Fatalf("image content has type %T", result.Content[1])
	}
	if got.MIMEType != "image/png" || !bytes.Equal(got.Data, image) {
		t.Fatalf("image metadata or bytes differ: mime=%q length=%d", got.MIMEType, len(got.Data))
	}
}
