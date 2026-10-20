// Package mcpbridge lets a program in a sandbox use tools the server runs. The program speaks MCP to a Unix
// socket in a folder shared with it, so the sandbox needs no network and the server no extra port.
package mcpbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// SocketName is the socket's name inside the shared folder.
	SocketName = "mcp.sock"

	dirMode    = 0o750
	socketMode = 0o660
	// maxSocketPath is below the 108 bytes a Unix socket address can hold.
	maxSocketPath = 100
)

var errPathTooLong = errors.New("socket path is too long")

// Tool is one tool the program may call.
type Tool struct {
	Name        string
	Description string
	// Schema is the JSON Schema of the arguments, an object.
	Schema json.RawMessage
}

// Call runs a tool the program asked for. Its text goes back to the program; failed says the call did not work.
type Call func(ctx context.Context, name string, input json.RawMessage) (text string, failed bool)

// Serve answers the program on dir/mcp.sock until stop is called. Each program that connects gets its own session.
func Serve(ctx context.Context, dir string, tools []Tool, call Call) (stop func(), err error) {
	path := filepath.Join(dir, SocketName)
	if len(path) > maxSocketPath {
		return nil, fmt.Errorf("%w: %s", errPathTooLong, path)
	}

	if err = os.MkdirAll(dir, dirMode); err != nil {
		return nil, fmt.Errorf("create socket folder: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", path)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("listen on socket: %w", err)
	}

	if err = os.Chmod(path, socketMode); err != nil {
		cancel()
		_ = listener.Close()

		return nil, fmt.Errorf("protect socket: %w", err)
	}

	go accept(ctx, listener, tools, call)

	return func() {
		cancel()
		_ = listener.Close()
		_ = os.Remove(path)
	}, nil
}

func accept(ctx context.Context, listener net.Listener, tools []Tool, call Call) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go func() {
			defer func() { _ = conn.Close() }()

			server := serverOf(tools, call)
			_ = server.Run(ctx, &sdk.IOTransport{Reader: conn, Writer: conn})
		}()
	}
}

func serverOf(tools []Tool, call Call) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "team", Version: "1"}, nil)

	for _, tool := range tools {
		server.AddTool(
			&sdk.Tool{Name: tool.Name, Description: tool.Description, InputSchema: tool.Schema},
			func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				text, failed := call(ctx, tool.Name, req.Params.Arguments)

				return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}, IsError: failed}, nil
			},
		)
	}

	return server
}
