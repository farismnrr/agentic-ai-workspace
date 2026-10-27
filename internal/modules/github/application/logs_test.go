package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// logSpy records which token read the logs.
type logSpy struct {
	err   error
	token string
}

func (s *logSpy) FailedJobLogs(_ context.Context, access, _ string, _ int64) ([]domain.JobLog, error) {
	s.token = access

	return []domain.JobLog{{Name: "Deploy", Text: "boom"}}, s.err
}

func (s *logSpy) JobLog(_ context.Context, access, _ string, _ int64) (domain.JobLog, error) {
	s.token = access

	return domain.JobLog{Text: "one job"}, s.err
}

func TestCILogsReadAsTheAskedPersonAndNoOneElse(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")
	r.connect(t, "bob", "bob")

	spy := &logSpy{}
	r.svc.WithLogs(spy)

	logs, err := r.svc.CILogs(context.Background(), "bob", domain.LogRef{Repo: "bob/app", RunID: 5})
	if err != nil || len(logs) != 1 || logs[0].Text != "boom" {
		t.Fatalf("logs = %+v, err = %v", logs, err)
	}

	if len(spy.token) < 7 || spy.token[:7] != "ghu_bob" {
		t.Errorf("bob's logs were read with %q, want bob's token", spy.token)
	}

	never := domain.LogRef{Repo: "a/b", RunID: 1}
	if _, err = r.svc.CILogs(context.Background(), "carol", never); !errors.Is(err, domain.ErrNotConnected) {
		t.Errorf("a person who never connected: %v, want ErrNotConnected", err)
	}
}

func TestCILogsRefuseABadReferenceBeforeTouchingGitHub(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")

	spy := &logSpy{}
	r.svc.WithLogs(spy)

	bad := domain.LogRef{Repo: "a/../b", RunID: 1}
	if _, err := r.svc.CILogs(context.Background(), "alice", bad); !errors.Is(err, domain.ErrBadLogRef) {
		t.Errorf("a bad repository gave %v, want ErrBadLogRef", err)
	}

	if spy.token != "" {
		t.Errorf("GitHub was called with %q for a bad reference", spy.token)
	}
}

func TestCILogsPassGitHubsRefusalOn(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")
	r.svc.WithLogs(&logSpy{err: domain.ErrLogsDenied})

	job := domain.LogRef{Repo: "a/b", JobID: 3}
	if _, err := r.svc.CILogs(context.Background(), "alice", job); !errors.Is(err, domain.ErrLogsDenied) {
		t.Errorf("err = %v, want ErrLogsDenied", err)
	}
}
