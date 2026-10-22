package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// SessionAction is a bounded browser form operation. Selectors identify only the required controls.
type SessionAction struct {
	Action           string `json:"action"`
	Path             string `json:"path"`
	EmailSelector    string `json:"emailSelector"`
	PasswordSelector string `json:"passwordSelector"`
	SubmitSelector   string `json:"submitSelector"`
	ReadySelector    string `json:"readySelector"`
	ErrorSelector    string `json:"errorSelector"`
	Email            string `json:"email"`
	Password         string `json:"password"`
}

// SessionResult reports submission and an explicit caller-selected readiness check.
type SessionResult struct{ Status string }

// SessionRuntime performs the native browser operation in the short-lived capture container.
type SessionRuntime interface {
	Session(ctx context.Context, req CaptureRequest) (string, error)
}

// RunSessionAction executes a constrained login or logout in the task's own cookie jar.
func (s *Service) RunSessionAction(
	ctx context.Context, userID, projectID, taskID string, action SessionAction,
) (SessionResult, error) {
	if err := validateSessionAction(&action); err != nil {
		return s.actionFailure(ctx, userID, projectID, taskID, action, err)
	}
	if action.Action == "reset" {
		if err := s.ResetAgentSession(ctx, userID, projectID, taskID); err != nil {
			return SessionResult{}, err
		}
		return SessionResult{Status: "local_session_cleared"}, nil
	}
	if s.deps.AgentSessions == nil {
		return s.actionFailure(ctx, userID, projectID, taskID, action, domain.ErrNoPicture)
	}
	site, err := s.runningSite(ctx, projectID)
	if err != nil || site.UserID != userID {
		return s.actionFailure(ctx, userID, projectID, taskID, action, domain.ErrNotRunning)
	}
	runner, ok := s.deps.Runtime.(SessionRuntime)
	if !ok {
		return s.actionFailure(ctx, userID, projectID, taskID, action, domain.ErrNoPicture)
	}
	result, err := s.performSessionAction(ctx, userID, projectID, taskID, site, runner, action)
	if action.Action != "logout" {
		return result, err
	}
	scope := AgentSessionScope{UserID: userID, ProjectID: projectID, TaskID: taskID, ContainerID: site.ContainerID}
	s.deps.AgentSessions.Reset(scope)
	if err != nil {
		return SessionResult{Status: "logout_failed_local_session_cleared"}, err
	}
	result.Status = logoutStatus(result.Status)
	return result, nil
}

func (s *Service) performSessionAction(
	ctx context.Context, userID, projectID, taskID string, site domain.Site,
	runner SessionRuntime, action SessionAction,
) (SessionResult, error) {
	var result SessionResult
	err := s.withAgentSession(ctx, site, userID, projectID, taskID, func(opCtx context.Context, jar http.CookieJar) error {
		request := CaptureRequest{
			ProjectID: projectID, Port: site.Port, Prefix: site.BasePath(),
			Spec: domain.CaptureSpec{Path: action.Path}, Session: &action,
		}
		if s.deps.View != nil {
			view, closeView, openErr := s.deps.View.Open(
				opCtx, s.deps.Runtime.Socket(projectID), site.BasePath(), site.BasePath(), jar,
			)
			if openErr != nil {
				return domain.ErrNoPicture
			}
			defer closeView()
			request.ViewSocket = view
		}
		output, runErr := runner.Session(opCtx, request)
		if runErr != nil {
			return sessionRuntimeFailure(opCtx, runErr)
		}
		result.Status = parseSessionStatus(output)
		return nil
	})
	if errors.Is(err, context.DeadlineExceeded) {
		result.Status = "operation_timed_out"
		return result, nil
	}
	if err != nil {
		return SessionResult{}, err
	}
	return result, nil
}

func sessionRuntimeFailure(ctx context.Context, cause error) error {
	if errors.Is(cause, context.DeadlineExceeded) || errors.Is(cause, context.Canceled) {
		return fmt.Errorf("run session action: %w", cause)
	}
	if ctx.Err() != nil {
		return fmt.Errorf("run session action: %w", ctx.Err())
	}
	return domain.ErrNoPicture
}

func (s *Service) actionFailure(
	ctx context.Context, userID, projectID, taskID string, action SessionAction, cause error,
) (SessionResult, error) {
	if action.Action == "logout" {
		return s.failedLogout(ctx, userID, projectID, taskID)
	}
	return SessionResult{}, cause
}

func (s *Service) failedLogout(ctx context.Context, userID, projectID, taskID string) (SessionResult, error) {
	if err := s.ResetAgentSession(ctx, userID, projectID, taskID); err != nil {
		return SessionResult{}, err
	}
	return SessionResult{Status: "logout_failed_local_session_cleared"}, nil
}

func logoutStatus(status string) string {
	switch status {
	case "indicator_found":
		return "logout_submitted_ready_local_session_cleared"
	case "indicator_not_found":
		return "logout_submitted_indicator_missing_local_session_cleared"
	case "operation_timed_out":
		return "logout_timed_out_local_session_cleared"
	default:
		return "logout_failed_local_session_cleared"
	}
}
