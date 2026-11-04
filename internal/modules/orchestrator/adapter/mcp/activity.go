package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	orchestratordomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// recordLeadAction covers MCP work not already fully represented by the
// app's plan, research, approval, message and internal-worker events.
// It is independent of which connected app (Claude, ChatGPT, etc.) called.
func (s Services) recordLeadAction(
	ctx context.Context, user, client, name string, input, output any,
) error {
	var taskID, key, description string

	switch name {
	case "create_task":
		in, inOK := input.(createInput)
		out, outOK := output.(taskdomain.Task)
		if !inOK || !outOK {
			return nil
		}
		taskID, key, description = out.ID, in.IdempotencyKey, "External lead created this task"
	case "delegate_agent":
		_, inOK := input.(delegateInput)
		out, outOK := output.(taskapp.ExternalAgentJob)
		if !inOK || !outOK {
			return nil
		}
		taskID = input.(delegateInput).TaskID
		key, description = out.ID, "External lead delegated to "+out.Role
	case "cancel_delegation":
		in, inOK := input.(delegationSelector)
		out, outOK := output.(taskapp.ExternalAgentJob)
		if !inOK || !outOK || out.Status != taskapp.DelegationCancelled {
			return nil
		}
		taskID, key, description = in.TaskID, in.JobID, "External lead cancelled a worker"
	case "save_file":
		in, inOK := input.(saveFileInput)
		if !inOK {
			return nil
		}
		// Do not store project paths, file text or a plaintext hash as activity.
		digest := sha256.Sum256([]byte(in.Project + "\x00" + in.Path + "\x00" +
			in.ExpectedHash + "\x00" + in.Content))
		taskID, key = in.TaskID, hex.EncodeToString(digest[:])
		description = "External lead saved a project file"
	case "finish_upload":
		in, inOK := input.(uploadIDInput)
		out, outOK := output.(orchestratordomain.UploadSession)
		if !inOK || !outOK {
			return nil
		}
		taskID, key, description = out.TaskID, in.UploadID, "External lead uploaded a project file"
	default:
		return nil // The handler already records a semantic task event.
	}
	if taskID == "" || key == "" {
		return nil
	}
	return s.Tasks.RecordExternalLeadAction(ctx, user, client, taskID, name, key, description)
}
