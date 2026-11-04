// Package mcp serves the external-only Orchestrator Space without touching Coding.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

type FileSaver interface {
	SaveText(ctx context.Context, userID, projectID, path, content, baseHash string) (string, error)
	List(ctx context.Context, userID, projectID, path string) (domain.FileListing, error)
}

type UploadManager interface {
	Start(context.Context, string, string, string, string, int64, string) (domain.UploadSession, error)
	Append(context.Context, string, string, string, int64, string) (domain.UploadSession, error)
	Status(context.Context, string, string, string) (domain.UploadSession, error)
	Finish(context.Context, string, string, string) (domain.UploadSession, error)
	Cancel(context.Context, string, string, string) error
}

type Services struct {
	Projects    *projectapp.Service
	Tasks       *taskapp.Service
	Permissions *spaceapp.Permissions
	Files       FileSaver
	Uploads     UploadManager
	MetadataURL string
}

func NewServer(s Services) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "masih-awam-orchestrator", Version: "1.0.0"},
		&sdk.ServerOptions{Instructions: "You are the sole external lead. Select an existing project and create one primary task per issue. Delegate directly to enabled internal worker agents; do not delegate to the internal orchestrator or create extra tasks per worker. After every worker has finished and your final report is ready, call complete_task once with that report, including when work is partial or not verified. Do not call complete_task after each worker. There is no conversation-ended signal, so the task stays Working until you explicitly call complete_task or the owner stops it."})
	s.addRead(server)
	s.addWrite(server)
	s.addUploads(server)
	return server
}

func tool(name string) *sdk.Tool {
	for _, info := range domain.Space().Tools {
		if info.Name == name {
			closed := true
			return &sdk.Tool{
				Name: name, Title: info.Title, Description: info.Description,
				Meta: sdk.Meta{"securitySchemes": []map[string]any{{"type": "oauth2", "scopes": []string{info.Scope}}}},
				Annotations: &sdk.ToolAnnotations{
					Title: info.Title, ReadOnlyHint: info.ReadOnly, OpenWorldHint: &closed,
				},
			}
		}
	}
	panic("unknown orchestrator tool " + name)
}

func problem(message string, meta sdk.Meta) *sdk.CallToolResult {
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: message}}, Meta: meta}
}

func (s Services) identity(ctx context.Context, req *sdk.CallToolRequest, name, scope string) (string, string, *sdk.CallToolResult) {
	token := req.GetExtra().TokenInfo
	if token == nil || token.UserID == "" {
		return "", "", problem("Sign in to use Orchestrator Space.", nil)
	}
	enabled, err := s.Permissions.Enabled(ctx, token.UserID, "orchestrator", name)
	if err != nil {
		return "", "", problem("Could not check Orchestrator Space permissions.", nil)
	}
	if !enabled {
		return "", "", problem("This Orchestrator tool is off. Enable it under Settings > Spaces.", nil)
	}
	if !slices.Contains(token.Scopes, scope) {
		header := fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q, resource_metadata=%q`, scope, s.MetadataURL)
		return "", "", problem("Reconnect with the "+scope+" permission.", sdk.Meta{"mcp/www_authenticate": []string{header}})
	}
	client, _ := token.Extra["client"].(string)
	if client == "" {
		return "", "", problem("Connected app identity is missing.", nil)
	}
	return token.UserID, client, nil
}

func secured[In, Out any](s Services, name, scope string, fn func(context.Context, string, string, In) (Out, error)) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *sdk.CallToolRequest, input In) (*sdk.CallToolResult, Out, error) {
		var empty Out
		started := time.Now()
		user, client, denied := s.identity(ctx, req, name, scope)
		if name == "cancel_delegation" {
			outcome := "allowed"
			if denied != nil {
				outcome = "denied"
			}
			slog.Info("orchestrator cancel identity", "stage", "identity",
				"outcome", outcome, "total_ms", time.Since(started).Milliseconds())
		}
		if denied != nil {
			return denied, empty, nil
		}
		result, err := fn(ctx, user, client, input)
		if err != nil {
			return problem("Orchestrator "+name+": "+err.Error(), nil), empty, nil
		}
		if recordErr := s.recordLeadAction(ctx, user, client, name, input, result); recordErr != nil {
			// The operation itself may have succeeded. Do not turn it into
			// an apparent failure that would encourage unsafe retries.
			slog.Warn("orchestrator activity recording failed", "tool", name, "error", recordErr)
			return &sdk.CallToolResult{
				Meta: sdk.Meta{"masih-awam/activity": "record_failed"},
			}, result, nil
		}
		return nil, result, nil
	}
}
