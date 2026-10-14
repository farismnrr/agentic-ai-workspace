package bootstrap

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	codingbackup "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/backup"
	codingfs "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

var errDataInsideWorkspace = errors.New("DATA_DIR must not be inside CODING_WORKSPACE_ROOT")

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

// codingServices builds the coding use cases on top of the workspace folder.
func codingServices(cfg config.Config) (codingmcp.Services, error) {
	if inside(cfg.CodingRoot, cfg.DataDir) {
		return codingmcp.Services{}, fmt.Errorf("%w: %s", errDataInsideWorkspace, cfg.DataDir)
	}

	files, err := codingfs.Open(cfg.CodingRoot)
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("open coding workspace: %w", err)
	}

	backups, err := codingbackup.Open(cfg.DataDir)
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("open backups: %w", err)
	}

	dir, err := filepath.Abs(cfg.CodingRoot)
	if err != nil {
		return codingmcp.Services{}, fmt.Errorf("resolve workspace: %w", err)
	}

	return codingmcp.Services{
		Reader: codingapp.NewService(files),
		Editor: codingapp.NewEditor(files, files, backups, opaque.Generator{}),
		Tasks:  codingapp.NewTasks(files, codingrunner.New(dir)),
	}, nil
}

// codingHandler serves the coding tools over stateless streamable HTTP with JSON
// responses. Localhost DNS-rebinding protection is off because the endpoint is
// public (often behind a local tunnel) and every request needs a bearer token.
func codingHandler(services codingmcp.Services, opts codingmcp.Options) http.Handler {
	server := codingmcp.NewServer(services, opts)

	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, DisableLocalhostProtection: true,
	})
}

// inside reports whether path lies within (or equals) root.
func inside(root, path string) bool {
	absRoot, errRoot := filepath.Abs(root)
	absPath, errPath := filepath.Abs(path)

	if errRoot != nil || errPath != nil {
		return true
	}

	rel, err := filepath.Rel(absRoot, absPath)

	return err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
