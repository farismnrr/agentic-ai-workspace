package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// LogSource reads the logs of GitHub Actions runs with a person's token.
type LogSource interface {
	// FailedJobLogs returns the logs of the jobs of a run that failed; none failed gives no logs.
	FailedJobLogs(ctx context.Context, access, repo string, runID int64) ([]domain.JobLog, error)
	JobLog(ctx context.Context, access, repo string, jobID int64) (domain.JobLog, error)
}

// WithLogs lets the service read workflow logs. Without it CILogs reports that GitHub is unavailable.
func (s *Service) WithLogs(source LogSource) *Service {
	s.logs = source

	return s
}

// CILogs reads the logs of a workflow run or job as the person who owns the connection. The token never leaves
// this call: the answer is text only.
func (s *Service) CILogs(ctx context.Context, userID string, ref domain.LogRef) ([]domain.JobLog, error) {
	if !ref.Valid() {
		return nil, domain.ErrBadLogRef
	}

	if s.logs == nil {
		return nil, domain.ErrNotConfigured
	}

	cred, err := s.Credentials(ctx, userID)
	if err != nil {
		return nil, err
	}

	if ref.JobID > 0 {
		log, readErr := s.logs.JobLog(ctx, cred.Token, ref.Repo, ref.JobID)
		if readErr != nil {
			return nil, fmt.Errorf("read job log: %w", readErr)
		}

		return []domain.JobLog{log}, nil
	}

	logs, err := s.logs.FailedJobLogs(ctx, cred.Token, ref.Repo, ref.RunID)
	if err != nil {
		return nil, fmt.Errorf("read run logs: %w", err)
	}

	return logs, nil
}
