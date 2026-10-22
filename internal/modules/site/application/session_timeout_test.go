package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

type contextError uint8

const (
	noContextError contextError = iota
	deadlineAfterCallback
	cancelAfterCallback
)

func (s *sessionProbe) registryError() error {
	switch s.afterOp {
	case deadlineAfterCallback:
		return fmt.Errorf("registry wrapped: %w", context.DeadlineExceeded)
	case cancelAfterCallback:
		return fmt.Errorf("registry wrapped: %w", context.Canceled)
	case noContextError:
		return nil
	default:
		return nil
	}
}

func TestSessionLateCallbackDeadlineBecomesTimeout(t *testing.T) {
	service, sessions := newLateSessionService(t, deadlineAfterCallback)
	result, err := service.RunSessionAction(ctx, "u1", "p1", "task1", validSessionAction())
	if err != nil || result.Status != "operation_timed_out" {
		t.Fatalf("late deadline result = %+v, %v", result, err)
	}
	if sessions.jar == nil {
		t.Fatal("session operation did not run before the late deadline")
	}
}

func TestSessionCancelRemainsFailure(t *testing.T) {
	service, _ := newLateSessionService(t, cancelAfterCallback)
	result, err := service.RunSessionAction(ctx, "u1", "p1", "task1", validSessionAction())
	if !errors.Is(err, context.Canceled) || result.Status != "" {
		t.Fatalf("cancel result = %+v, %v", result, err)
	}
}

func newLateSessionService(t *testing.T, afterOp contextError) (*application.Service, *sessionProbe) {
	t.Helper()
	store := &memStore{sites: map[string]domain.Site{}}
	runtime, sessions := &fakeRuntime{}, &sessionProbe{afterOp: afterOp}
	runtime.sessionOutput = `SESSION_JSON:{"status":"indicator_found"}`
	service := application.NewService(application.Deps{
		Store: store, Runtime: runtime, Projects: fakeProjects{"p1": {"u1", "shop"}},
		Notify: &counter{}, IDs: &seq{}, Now: time.Now, AgentSessions: sessions,
	})
	if _, err := service.Start(ctx, "p1", "serve", 8000); err != nil {
		t.Fatal(err)
	}
	return service, sessions
}

func validSessionAction() application.SessionAction {
	return application.SessionAction{
		Action: "login", Path: "/login", EmailSelector: "#email", PasswordSelector: "#password",
		SubmitSelector: "button[type=submit]", ReadySelector: "[data-ready=account]",
		Email: "test@example.test", Password: "secret",
	}
}
