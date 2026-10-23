package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxMessageCharacters = 12_000

var errInputTooLong = errors.New("orchestrator input is too long")

type tools struct {
	service     Service
	gate        Gate
	activity    Activity
	metadataURL string
	taskURLBase string
}

type startInput struct {
	ProjectID string `json:"projectId" jsonschema:"ID of the project where the team should work"`
	Prompt    string `json:"prompt" jsonschema:"what the team should do"`
}

type taskInput struct {
	ProjectID string `json:"projectId" jsonschema:"ID of the project that contains the task"`
	TaskID    string `json:"taskId" jsonschema:"ID of the Orchestrator task"`
}

type projectsInput struct{}

type projectTasksInput struct {
	ProjectID string `json:"projectId" jsonschema:"ID of the project whose team tasks should be listed"`
}

type projectOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type taskSummary struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

type taskListOutput struct {
	ProjectID string        `json:"projectId"`
	Tasks     []taskSummary `json:"tasks"`
}

type messageInput struct {
	ProjectID string `json:"projectId" jsonschema:"ID of the project that contains the task"`
	TaskID    string `json:"taskId" jsonschema:"ID of the Orchestrator task"`
	Text      string `json:"text" jsonschema:"the message to add to the task"`
}

type taskOutput struct {
	ProjectID string `json:"projectId"`
	TaskID    string `json:"taskId"`
	Status    string `json:"status"`
	URL       string `json:"url"`
}

func (t tools) addTools(server *sdk.Server) {
	sdk.AddTool(server, t.definition("list_projects", "List your projects", "orchestrator:read", true),
		guarded(t, "list_projects", "orchestrator:read", func(projectsInput) string { return "list your projects" },
			func(projectsInput) string { return "" }, func(projectsInput) string { return "" }, t.listProjects))
	sdk.AddTool(server, t.definition("list_tasks", "List team tasks", "orchestrator:read", true),
		guarded(t, "list_tasks", "orchestrator:read", func(projectTasksInput) string { return "list team tasks" },
			func(in projectTasksInput) string { return in.ProjectID },
			func(projectTasksInput) string { return "" }, t.listTasks))
	sdk.AddTool(server, t.definition("start_task", "Start a team task", "orchestrator:run", false),
		guarded(t, "start_task", "orchestrator:run", func(startInput) string { return "start a team task" },
			projectOf, func(startInput) string { return "" }, t.start))
	sdk.AddTool(server, t.definition("get_task", "Get task status", "orchestrator:read", true),
		guarded(t, "get_task", "orchestrator:read", func(taskInput) string { return "read task status" },
			projectOfTask, taskOfTask, t.get))
	sdk.AddTool(server, t.definition("send_message", "Send a task message", "orchestrator:run", false),
		guarded(t, "send_message", "orchestrator:run", func(messageInput) string { return "send a task message" },
			projectOfMessage, taskOfMessage, t.send))
	sdk.AddTool(server, t.definition("retry_task", "Retry a failed task", "orchestrator:run", false),
		guarded(t, "retry_task", "orchestrator:run", func(taskInput) string { return "retry a failed task" },
			projectOfTask, taskOfTask, t.retry))
}

func (t tools) listProjects(ctx context.Context, userID string, _ projectsInput) ([]projectOutput, error) {
	projects, err := t.service.ListOrchestratorProjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list Orchestrator projects: %w", err)
	}
	result := make([]projectOutput, 0, len(projects))
	for _, project := range projects {
		result = append(result, projectOutput{ID: project.ID, Name: project.Name})
	}
	return result, nil
}

func (t tools) listTasks(ctx context.Context, userID string, in projectTasksInput) (taskListOutput, error) {
	tasks, err := t.service.ListOrchestratorTasks(ctx, userID, in.ProjectID)
	if err != nil {
		return taskListOutput{}, fmt.Errorf("list Orchestrator tasks: %w", err)
	}
	result := taskListOutput{ProjectID: in.ProjectID, Tasks: make([]taskSummary, 0, len(tasks))}
	for _, task := range tasks {
		result.Tasks = append(result.Tasks, taskSummary{
			ID: task.ID, Title: task.Title, Status: string(task.Status), URL: t.taskURL(task.ProjectID, task.ID),
		})
	}
	return result, nil
}

func (t tools) definition(name, title, scope string, readOnly bool) *sdk.Tool {
	return &sdk.Tool{
		Name: name, Title: title, Description: title + " using the selected project and task.",
		Meta:        sdk.Meta{"securitySchemes": []map[string]any{{"type": "oauth2", "scopes": []string{scope}}}},
		Annotations: &sdk.ToolAnnotations{Title: title, ReadOnlyHint: readOnly},
	}
}

func (t tools) output(task taskdomain.Task) taskOutput {
	return taskOutput{
		ProjectID: task.ProjectID, TaskID: task.ID, Status: string(task.Status),
		URL: t.taskURL(task.ProjectID, task.ID),
	}
}

func (t tools) taskURL(projectID, taskID string) string {
	return taskLink(t.taskURLBase, projectID, taskID)
}

func (t tools) start(ctx context.Context, userID string, in startInput) (taskOutput, error) {
	if utf8.RuneCountInString(in.Prompt) > maxMessageCharacters {
		return taskOutput{}, errInputTooLong
	}

	task, err := t.service.StartOrchestrator(ctx, userID, in.ProjectID, in.Prompt)
	if err != nil {
		return taskOutput{}, fmt.Errorf("start Orchestrator task: %w", err)
	}
	return t.output(task), nil
}

func taskLink(base, projectID, taskID string) string {
	return strings.TrimRight(base, "/") + "/app/projects/" + url.PathEscape(projectID) + "/tasks/" + url.PathEscape(taskID)
}

func (t tools) get(ctx context.Context, userID string, in taskInput) (taskOutput, error) {
	task, err := t.service.GetOrchestrator(ctx, userID, in.ProjectID, in.TaskID)
	return t.output(task), err
}

func (t tools) send(ctx context.Context, userID string, in messageInput) (taskOutput, error) {
	if utf8.RuneCountInString(in.Text) > maxMessageCharacters {
		return taskOutput{}, errInputTooLong
	}

	if err := t.service.SendOrchestratorMessage(ctx, userID, in.ProjectID, in.TaskID, in.Text); err != nil {
		return taskOutput{}, fmt.Errorf("send Orchestrator message: %w", err)
	}

	return t.queued(in.ProjectID, in.TaskID), nil
}

func (t tools) retry(ctx context.Context, userID string, in taskInput) (taskOutput, error) {
	if err := t.service.RetryOrchestrator(ctx, userID, in.ProjectID, in.TaskID); err != nil {
		return taskOutput{}, fmt.Errorf("retry Orchestrator task: %w", err)
	}

	return t.queued(in.ProjectID, in.TaskID), nil
}

func (t tools) queued(projectID, taskID string) taskOutput {
	return taskOutput{ProjectID: projectID, TaskID: taskID, Status: "queued", URL: t.taskURL(projectID, taskID)}
}
