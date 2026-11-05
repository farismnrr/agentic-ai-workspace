package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type createInput struct {
	Project        string `json:"project"`
	Summary        string `json:"summary"`
	IdempotencyKey string `json:"idempotency_key"`
}
type delegateInput struct {
	TaskID         string   `json:"task_id"`
	AgentKey       string   `json:"agent_key"`
	Brief          string   `json:"brief"`
	Context        string   `json:"context,omitempty"`
	FileScope      []string `json:"file_scope,omitempty"`
	Effort         string   `json:"effort,omitempty"`
	IdempotencyKey string   `json:"idempotency_key"`
}
type planInput struct {
	TaskID string                `json:"task_id"`
	Steps  []taskdomain.PlanStep `json:"steps"`
}
type documentInput struct {
	TaskID   string `json:"task_id"`
	Markdown string `json:"markdown"`
}
type stopInput struct {
	TaskID string `json:"task_id"`
}
type completeInput struct {
	TaskID  string `json:"task_id"`
	Summary string `json:"summary"`
	Outcome string `json:"outcome,omitempty"`
}
type doneOutput struct {
	Status string `json:"status"`
}
type approveInput struct {
	TaskID         string `json:"task_id"`
	EventID        string `json:"event_id"`
	IdempotencyKey string `json:"idempotency_key"`
}
type saveFileInput struct {
	TaskID       string `json:"task_id"`
	Project      string `json:"project"`
	Path         string `json:"path"`
	Content      string `json:"content"`
	ExpectedHash string `json:"expected_hash,omitempty" jsonschema:"empty means the file must not exist"`
}
type savedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

func (s Services) addWrite(server *sdk.Server) {
	sdk.AddTool(server, tool("create_task"), secured(s, "create_task", domain.ScopeTasks,
		func(ctx context.Context, user, client string, in createInput) (taskdomain.Task, error) {
			if err := s.ownProject(ctx, user, in.Project); err != nil {
				return taskdomain.Task{}, err
			}
			return s.Tasks.CreateExternalLeadTask(ctx, user, client, in.Project, in.Summary, in.IdempotencyKey)
		}))
	sdk.AddTool(server, tool("delegate_agent"), secured(s, "delegate_agent", domain.ScopeDelegate,
		func(ctx context.Context, user, _ string, in delegateInput) (taskapp.ExternalAgentJob, error) {
			return s.Tasks.DelegateExternal(ctx, user, taskapp.ExternalAgentInput{
				TaskID: in.TaskID, AgentKey: in.AgentKey, Brief: in.Brief, Context: in.Context,
				Scope: in.FileScope, Effort: in.Effort, RequestKey: in.IdempotencyKey,
			})
		}))
	sdk.AddTool(server, tool("cancel_delegation"), secured(s, "cancel_delegation", domain.ScopeDelegate,
		func(ctx context.Context, user, _ string, in delegationSelector) (taskapp.ExternalAgentJob, error) {
			started := time.Now()
			job, err := s.Tasks.CancelExternalJob(ctx, user, in.TaskID, in.JobID)
			outcome := "error"
			if err == nil {
				outcome = job.Status
			}
			slog.Info("orchestrator cancel handler", "outcome", outcome,
				"total_ms", time.Since(started).Milliseconds())
			return job, err
		}))
	sdk.AddTool(server, tool("stop_task"), secured(s, "stop_task", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in stopInput) (doneOutput, error) {
			err := s.Tasks.StopExternalLeadTask(ctx, user, in.TaskID)
			return doneOutput{Status: "stopped"}, err
		}))
	sdk.AddTool(server, tool("continue_task"), secured(s, "continue_task", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in stopInput) (doneOutput, error) {
			err := s.Tasks.ContinueExternalLeadTask(ctx, user, in.TaskID)
			return doneOutput{Status: "working"}, err
		}))
	sdk.AddTool(server, tool("complete_task"), secured(s, "complete_task", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in completeInput) (doneOutput, error) {
			err := s.Tasks.CompleteExternalLeadTask(ctx, user, in.TaskID, in.Summary, in.Outcome)
			return doneOutput{Status: "done"}, err
		}))
	planTool := tool("update_task_plan")
	planTool.InputSchema = json.RawMessage(`{
		"type": "object",
		"required": ["task_id", "steps"],
		"properties": {
			"task_id": {"type": "string"},
			"steps": {
				"type": "array",
				"description": "Provide 1 to 40 plan steps.",
				"minItems": 1,
				"maxItems": 40,
				"items": {
					"type": "object",
					"required": ["id", "title", "state"],
					"properties": {
						"id": {"type": "string", "minLength": 1},
						"title": {"type": "string", "minLength": 1},
						"state": {
							"type": "string",
						"enum": ["pending", "active", "done", "failed"]
						}
					}
				}
			}
		}
	}`)
	sdk.AddTool(server, planTool, secured(s, "update_task_plan", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in planInput) (doneOutput, error) {
			err := s.Tasks.UpdateExternalPlan(ctx, user, in.TaskID, in.Steps)
			return doneOutput{Status: "saved"}, err
		}))
	sdk.AddTool(server, tool("save_task_research"), secured(s, "save_task_research", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in documentInput) (doneOutput, error) {
			err := s.Tasks.SaveExternalResearch(ctx, user, in.TaskID, in.Markdown)
			return doneOutput{Status: "saved"}, err
		}))
	sdk.AddTool(server, tool("save_task_requirements"), secured(s, "save_task_requirements", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in documentInput) (doneOutput, error) {
			err := s.Tasks.SaveExternalPRD(ctx, user, in.TaskID, in.Markdown)
			return doneOutput{Status: "saved"}, err
		}))
	sdk.AddTool(server, tool("submit_plan"), secured(s, "submit_plan", domain.ScopeTasks,
		func(ctx context.Context, user, _ string, in stopInput) (taskapp.ExternalPlanApproval, error) {
			return s.Tasks.SubmitExternalPlan(ctx, user, in.TaskID)
		}))
	sdk.AddTool(server, tool("approve_plan"), secured(s, "approve_plan", domain.ScopeApprove,
		func(ctx context.Context, user, client string, in approveInput) (taskapp.ExternalPlanApproval, error) {
			return s.Tasks.ApproveExternalPlan(ctx, user, client, in.TaskID, in.EventID, in.IdempotencyKey)
		}))
	sdk.AddTool(server, tool("save_file"), secured(s, "save_file", domain.ScopeFiles,
		func(ctx context.Context, user, _ string, in saveFileInput) (savedFile, error) {
			if err := s.Tasks.AuthorizeExternalFileWrite(ctx, user, in.Project, in.TaskID); err != nil {
				return savedFile{}, err
			}
			path, err := s.Files.SaveText(ctx, user, in.Project, in.Path, in.Content, in.ExpectedHash)
			return savedFile{Path: path, Status: "saved"}, err
		}))
}
