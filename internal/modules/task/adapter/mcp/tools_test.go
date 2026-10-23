package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type taskService struct {
	err                               error
	userID, projectID, taskID, prompt string
	task                              taskdomain.Task
}

func (s *taskService) ListOrchestratorProjects(context.Context, string) ([]taskapp.ProjectSummary, error) {
	return nil, s.err
}

func (s *taskService) ListOrchestratorTasks(context.Context, string, string) ([]taskdomain.Task, error) {
	return nil, s.err
}

var errDatabaseDetails = errors.New("database details")

func (s *taskService) StartOrchestrator(_ context.Context, userID, projectID, prompt string) (taskdomain.Task, error) {
	s.userID, s.projectID, s.prompt = userID, projectID, prompt
	return s.task, s.err
}

func (s *taskService) GetOrchestrator(_ context.Context, userID, projectID, taskID string) (taskdomain.Task, error) {
	s.userID, s.projectID, s.taskID = userID, projectID, taskID
	return s.task, s.err
}

func (s *taskService) SendOrchestratorMessage(context.Context, string, string, string, string) error {
	return s.err
}

func (s *taskService) RetryOrchestrator(context.Context, string, string, string) error { return s.err }

type gate struct{ allowed bool }

func (g gate) Allowed(context.Context, string, string) (bool, error) { return g.allowed, nil }

func TestStartUsesVerifiedUserAndBuildsTaskURL(t *testing.T) {
	service := &taskService{task: taskdomain.Task{
		ID: "task-1", ProjectID: "project-1", Status: taskdomain.StatusQueued,
	}}
	tool := tools{service: service, taskURLBase: "https://app.example/"}
	got, err := tool.start(context.Background(), "verified-user", startInput{ProjectID: "project-1", Prompt: "Build it"})
	if err != nil {
		t.Fatal(err)
	}
	if service.userID != "verified-user" || service.projectID != "project-1" || service.prompt != "Build it" {
		t.Fatalf("service call used unexpected identity or input: %#v", service)
	}
	if got.Status != "queued" || got.URL != "https://app.example/app/projects/project-1/tasks/task-1" {
		t.Fatalf("unexpected task response: %#v", got)
	}
}

func TestOrchestratorAdapterRejectsUnavailableOrMismatchedTasks(t *testing.T) {
	t.Run("team unavailable", func(t *testing.T) {
		assertAdapterError(t, taskdomain.ErrTeamUnavailable, "The team is not ready.", true)
	})
	t.Run("project mismatch", func(t *testing.T) {
		assertAdapterError(t, taskdomain.ErrNotFound, "The project or task was not found.", false)
	})
	t.Run("solo task", func(t *testing.T) {
		assertAdapterError(t, taskdomain.ErrWrongState, "This task cannot do that right now.", false)
	})
	t.Run("connected task source", func(t *testing.T) {
		assertAdapterError(t, taskdomain.ErrWrongState, "This task cannot do that right now.", false)
	})
}

func assertAdapterError(t *testing.T, cause error, want string, start bool) {
	t.Helper()
	tool := tools{service: &taskService{err: cause}}
	var err error
	if start {
		got, startErr := tool.start(context.Background(), "u1", startInput{ProjectID: "p1", Prompt: "Build it"})
		err = startErr
		if got != (taskOutput{}) {
			t.Fatalf("failed start returned task data: %#v", got)
		}
	} else {
		_, err = tool.get(context.Background(), "u1", taskInput{ProjectID: "p1", TaskID: "t1"})
	}
	if !errors.Is(err, cause) {
		t.Fatalf("tool error = %v, want %v", err, cause)
	}
	if message := errorMessage(err); !strings.Contains(message, want) {
		t.Fatalf("error message = %q, want it to contain %q", message, want)
	}
}

func TestOrchestratorInputsAreLengthLimited(t *testing.T) {
	service := &taskService{}
	tool := tools{service: service}
	long := strings.Repeat("x", maxMessageCharacters+1)

	if _, err := tool.start(context.Background(), "u1", startInput{Prompt: long}); !errors.Is(err, errInputTooLong) {
		t.Fatalf("oversized prompt was accepted: %v", err)
	}
	if _, err := tool.send(context.Background(), "u1", messageInput{Text: long}); !errors.Is(err, errInputTooLong) {
		t.Fatalf("oversized message was accepted: %v", err)
	}
	if service.prompt != "" {
		t.Fatal("oversized prompt reached the task service")
	}
}

type activityRecorder struct{ entry Entry }

func (a *activityRecorder) Record(_ context.Context, entry Entry) error {
	a.entry = entry
	return nil
}

func TestActivityIDsAreTruncatedBeforeRecording(t *testing.T) {
	activity := &activityRecorder{}
	tool := tools{activity: activity}
	tool.record(context.Background(), Entry{
		Project: strings.Repeat("p", 70), Task: strings.Repeat("t", 70),
	})

	if len([]rune(activity.entry.Project)) != 64 || len([]rune(activity.entry.Task)) != 64 {
		t.Fatalf("activity IDs were not capped at 64 characters: %#v", activity.entry)
	}
}

func TestRefuseChecksSwitchAndScope(t *testing.T) {
	entry := Entry{UserID: "user-1"}
	if result := refuse(context.Background(), tools{gate: gate{}}, "start_task", "orchestrator:run",
		[]string{"orchestrator:run"}, &entry); result == nil || entry.Outcome != "off" {
		t.Fatalf("disabled tool was not refused: result=%v entry=%#v", result, entry)
	}
	entry = Entry{UserID: "user-1"}
	if result := refuse(context.Background(), tools{gate: gate{allowed: true}}, "start_task", "orchestrator:run",
		[]string{"orchestrator:read"}, &entry); result == nil || entry.Outcome != "forbidden" {
		t.Fatalf("missing scope was not refused: result=%v entry=%#v", result, entry)
	}
}

func TestReadToolsRequireReadSwitchAndScope(t *testing.T) {
	for _, name := range []string{"list_projects", "list_tasks"} {
		entry := Entry{UserID: "user-1"}
		if result := refuse(context.Background(), tools{gate: gate{}}, name, "orchestrator:read",
			[]string{"orchestrator:read"}, &entry); result == nil || entry.Outcome != "off" {
			t.Fatalf("%s did not honor disabled switch: result=%v entry=%#v", name, result, entry)
		}
		entry = Entry{UserID: "user-1"}
		if result := refuse(context.Background(), tools{gate: gate{allowed: true}}, name, "orchestrator:read",
			[]string{"orchestrator:run"}, &entry); result == nil || entry.Outcome != "forbidden" {
			t.Fatalf("%s did not honor missing read scope: result=%v entry=%#v", name, result, entry)
		}
	}
}

func TestApplicationErrorsAreMappedToSafeMessages(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{taskdomain.ErrNotFound, "The project or task was not found."},
		{taskdomain.ErrNeedsAnswer, "This task cannot do that right now."},
		{taskdomain.ErrTeamUnavailable, "The team is not ready."},
		{errDatabaseDetails, "The request could not be completed."},
	} {
		if got := errorMessage(test.err); !strings.Contains(got, test.want) {
			t.Errorf("errorMessage(%v) = %q, want it to contain %q", test.err, got, test.want)
		}
	}
}

func TestHandlerRejectsMissingToken(t *testing.T) {
	// Authentication is normally supplied by the Space mount; the tool guard still fails closed.
	service := &taskService{}
	handler := guarded(tools{service: service, gate: gate{allowed: true}}, "get_task", "orchestrator:read",
		func(taskInput) string { return "read task status" }, projectOfTask, taskOfTask,
		func(ctx context.Context, userID string, in taskInput) (taskOutput, error) {
			return taskOutput{}, nil
		})
	result, _, err := handler(context.Background(), &sdk.CallToolRequest{}, taskInput{})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("missing token should be rejected with a tool error: result=%v err=%v", result, err)
	}
}

var _ Service = (*taskService)(nil)
