package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/mcpbridge"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// bridgeWait is how long, in seconds, a program waits for a tool: a teammate's job can take a long time.
const bridgeWait = 3600

// bridgeWaitMillis is the same wait, for the program that counts in milliseconds.
const bridgeWaitMillis = bridgeWait * 1000

// serveBridge starts the socket a program reaches the server's tools by, in a folder only this turn can see.
// stop removes the folder.
func (p *plans) serveBridge(ctx context.Context, bridge *taskapp.CLIBridge) (stop func(), dir string, err error) {
	base := filepath.Join(p.box.base, "..", "br")
	if err = os.MkdirAll(base, homeMode); err != nil {
		return nil, "", fmt.Errorf("make bridge folder: %w", err)
	}

	dir, err = os.MkdirTemp(base, "b")
	if err != nil {
		return nil, "", fmt.Errorf("make bridge folder: %w", err)
	}

	tools := make([]mcpbridge.Tool, 0, len(bridge.Tools))
	for _, tool := range bridge.Tools {
		tools = append(tools, mcpbridge.Tool{Name: tool.Name, Description: tool.Description, Schema: tool.Schema})
	}

	closeSocket, err := mcpbridge.Serve(ctx, dir, tools, bridge.Call)
	if err != nil {
		_ = os.RemoveAll(dir)

		return nil, "", fmt.Errorf("serve tools: %w", err)
	}

	return func() { closeSocket(); _ = os.RemoveAll(dir) }, dir, nil
}

// bridgeArgs tell the program where the server's tools are: a socket it reaches through socat, with no network.
func bridgeArgs(rt domain.Runtime) []string {
	socket := "UNIX-CONNECT:" + sandbox.BridgeDir + "/" + mcpbridge.SocketName
	if rt == domain.ChatGPT {
		return []string{
			"-c", `mcp_servers.team.command="socat"`,
			"-c", `mcp_servers.team.args=["STDIO","` + socket + `"]`,
			"-c", fmt.Sprintf("mcp_servers.team.tool_timeout_sec=%d", bridgeWait),
		}
	}

	config := `{"mcpServers":{"team":{"command":"socat","args":["STDIO","` + socket + `"]}}}`

	return []string{"--mcp-config", config, "--strict-mcp-config"}
}
