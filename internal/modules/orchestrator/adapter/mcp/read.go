package mcp

import (
	"context"
	"errors"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"
	projectdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type projectItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
type projectList struct {
	Projects []projectItem `json:"projects"`
}
type agentItem struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Purpose     string   `json:"purpose"`
	Tools       []string `json:"tools,omitempty"`
	CanWrite    bool     `json:"can_write"`
	CanParallel bool     `json:"can_parallel"`
}
type agentList struct {
	Agents []agentItem `json:"agents"`
}
type taskSelector struct {
	Project string `json:"project" jsonschema:"required; project ID from list_projects"`
}
type taskDetail struct {
	Project string `json:"project" jsonschema:"required; project ID"`
	TaskID  string `json:"task_id" jsonschema:"required; task ID"`
}
type tasksList struct {
	Tasks []taskdomain.Task `json:"tasks"`
}
type taskEventsInput struct {
	TaskID string `json:"task_id"`
	After  int64  `json:"after,omitempty"`
}
type taskEvents struct {
	Events []taskdomain.Event `json:"events"`
}
type delegationSelector struct {
	TaskID string `json:"task_id"`
	JobID  string `json:"job_id"`
}
type delegationList struct {
	Jobs []taskapp.ExternalAgentJob `json:"jobs"`
}

func (s Services) ownProject(ctx context.Context, user, project string) error {
	if project == "" {
		return errors.New("choose an existing project")
	}
	_, err := s.Projects.Get(ctx, user, project)
	if err != nil {
		return projectdomain.ErrNotFound
	}
	return nil
}

func (s Services) ownTask(ctx context.Context, user, project, taskID string) (taskdomain.Task, error) {
	if err := s.ownProject(ctx, user, project); err != nil {
		return taskdomain.Task{}, err
	}
	task, err := s.Tasks.Get(ctx, user, taskID)
	if err != nil {
		return taskdomain.Task{}, err
	}
	if task.ProjectID != project {
		return taskdomain.Task{}, taskdomain.ErrNotFound
	}
	return task, nil
}

func (s Services) addRead(server *sdk.Server) {
	sdk.AddTool(server, tool("list_projects"), secured(s, "list_projects", domain.ScopeRead,
		func(ctx context.Context, user, _ string, _ struct{}) (projectList, error) {
			found, err := s.Projects.List(ctx, user)
			if err != nil {
				return projectList{}, err
			}
			out := projectList{Projects: make([]projectItem, 0, len(found))}
			for _, project := range found {
				out.Projects = append(out.Projects, projectItem{ID: project.ID, Name: project.Name, Description: project.Description})
			}
			return out, nil
		}))
	sdk.AddTool(server, tool("list_agents"), secured(s, "list_agents", domain.ScopeRead,
		func(ctx context.Context, user, _ string, _ struct{}) (agentList, error) {
			agents, err := s.Tasks.ExternalAgents(ctx, user)
			if err != nil {
				return agentList{}, err
			}
			out := agentList{Agents: make([]agentItem, 0, len(agents))}
			for _, role := range agents {
				if role.Lead {
					continue
				}
				out.Agents = append(out.Agents, agentItem{
					Key:         role.Key,
					Name:        role.Name,
					Purpose:     role.Purpose,
					Tools:       role.Tools,
					CanWrite:    taskapp.RoleCanWrite(role),
					CanParallel: taskapp.CanParallel(role),
				})
			}
			return out, nil
		}))
	sdk.AddTool(server, tool("list_tasks"), secured(s, "list_tasks", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in taskSelector) (tasksList, error) {
			if err := s.ownProject(ctx, user, in.Project); err != nil {
				return tasksList{}, err
			}
			found, err := s.Tasks.List(ctx, user, in.Project)
			return tasksList{Tasks: found}, err
		}))
	sdk.AddTool(server, tool("get_task"), secured(s, "get_task", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in taskDetail) (taskdomain.Task, error) {
			return s.ownTask(ctx, user, in.Project, in.TaskID)
		}))
	sdk.AddTool(server, tool("get_task_activity"), secured(s, "get_task_activity", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in taskEventsInput) (taskEvents, error) {
			if strings.TrimSpace(in.TaskID) == "" {
				return taskEvents{}, taskdomain.ErrBadRequest
			}
			found, err := s.Tasks.Events(ctx, user, in.TaskID, in.After)
			return taskEvents{Events: found}, err
		}))
	sdk.AddTool(server, tool("get_delegation"), secured(s, "get_delegation", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in delegationSelector) (taskapp.ExternalAgentJob, error) {
			return s.Tasks.ExternalJob(ctx, user, in.TaskID, in.JobID)
		}))
	sdk.AddTool(server, tool("list_files"), secured(s, "list_files", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in struct {
			Project string `json:"project"`
			Path    string `json:"path,omitempty"`
		}) (domain.FileListing, error) {
			return s.Files.List(ctx, user, in.Project, in.Path)
		}))
	sdk.AddTool(server, tool("get_approval_state"), secured(s, "get_approval_state", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in struct {
			TaskID string `json:"task_id"`
		}) (taskapp.ExternalPlanApproval, error) {
			return s.Tasks.ExternalApproval(ctx, user, in.TaskID)
		}))
	sdk.AddTool(server, tool("list_delegations"), secured(s, "list_delegations", domain.ScopeRead,
		func(ctx context.Context, user, _ string, in struct {
			TaskID string `json:"task_id"`
		}) (delegationList, error) {
			found, err := s.Tasks.ExternalJobs(ctx, user, in.TaskID)
			return delegationList{Jobs: found}, err
		}))
}
