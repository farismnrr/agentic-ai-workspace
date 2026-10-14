package bootstrap

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

var codingScopes = []spacedomain.Scope{
	{ID: codingdomain.ScopeRead, Description: "Read files and search code in your coding workspace"},
	{
		ID: codingdomain.ScopeWrite,
		Description: "Create, change, move and delete files in your coding workspace " +
			"(every change can be undone in Settings)",
	},
	{ID: codingdomain.ScopeExec, Description: "Run the tasks you set up, such as tests"},
}

func codingSpace() spacedomain.Space {
	tools := []spacedomain.Tool{}

	for _, tool := range codingdomain.Tools() {
		tools = append(tools, spacedomain.Tool{
			Name: tool.Name, Title: tool.Title, Description: tool.Description, Scope: tool.Scope,
			Group: string(tool.Group), ReadOnly: tool.ReadOnly, DefaultOn: tool.DefaultOn,
		})
	}

	return spacedomain.Space{
		Slug: "coding", Name: "Coding", Entry: codingdomain.ScopeRead, Scopes: codingScopes, Tools: tools,
	}
}

// codingHandler serves the coding tools over stateless streamable HTTP with JSON
// responses. Localhost DNS-rebinding protection is off because the endpoint is
// public (often behind a local tunnel) and every request needs a bearer token.
func codingHandler(workspaces codingmcp.Workspaces, opts codingmcp.Options) http.Handler {
	server := codingmcp.NewServer(workspaces, opts)

	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, DisableLocalhostProtection: true,
	})
}
