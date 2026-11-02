package mcp

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type startUploadInput struct {
	TaskID  string `json:"task_id"`
	Project string `json:"project"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}
type uploadIDInput struct {
	Project  string `json:"project"`
	UploadID string `json:"upload_id"`
}
type uploadChunkInput struct {
	Project    string `json:"project"`
	UploadID   string `json:"upload_id"`
	Offset     int64  `json:"offset"`
	DataBase64 string `json:"data_base64" jsonschema:"at most 512 KiB raw binary, Base64 encoded"`
}

func (s Services) addUploads(server *sdk.Server) {
	sdk.AddTool(server, tool("start_upload"), secured(s, "start_upload", domain.ScopeFiles,
		func(ctx context.Context, user, _ string, in startUploadInput) (domain.UploadSession, error) {
			if err := s.Tasks.AuthorizeExternalFileWrite(ctx, user, in.Project, in.TaskID); err != nil {
				return domain.UploadSession{}, err
			}
			out, err := s.Uploads.Start(ctx, user, in.Project, in.TaskID, in.Name, in.Size, in.SHA256)
			out.Owner = ""
			return out, err
		}))
	sdk.AddTool(server, tool("upload_chunk"), secured(s, "upload_chunk", domain.ScopeFiles,
		func(ctx context.Context, user, _ string, in uploadChunkInput) (domain.UploadSession, error) {
			out, err := s.Uploads.Append(ctx, user, in.Project, in.UploadID, in.Offset, in.DataBase64)
			out.Owner = ""
			return out, err
		}))
	sdk.AddTool(server, tool("upload_status"), secured(s, "upload_status", domain.ScopeFiles,
		func(ctx context.Context, user, _ string, in uploadIDInput) (domain.UploadSession, error) {
			out, err := s.Uploads.Status(ctx, user, in.Project, in.UploadID)
			out.Owner = ""
			return out, err
		}))
	sdk.AddTool(server, tool("finish_upload"), secured(s, "finish_upload", domain.ScopeFiles,
		func(ctx context.Context, user, _ string, in uploadIDInput) (domain.UploadSession, error) {
			current, err := s.Uploads.Status(ctx, user, in.Project, in.UploadID)
			if err != nil {
				return domain.UploadSession{}, err
			}
			if err := s.Tasks.AuthorizeExternalFileWrite(ctx, user, in.Project, current.TaskID); err != nil {
				return domain.UploadSession{}, err
			}
			out, err := s.Uploads.Finish(ctx, user, in.Project, in.UploadID)
			out.Owner = ""
			return out, err
		}))
	sdk.AddTool(server, tool("cancel_upload"), secured(s, "cancel_upload", domain.ScopeFiles,
		func(ctx context.Context, user, _ string, in uploadIDInput) (doneOutput, error) {
			err := s.Uploads.Cancel(ctx, user, in.Project, in.UploadID)
			return doneOutput{Status: "cancelled"}, err
		}))
}
