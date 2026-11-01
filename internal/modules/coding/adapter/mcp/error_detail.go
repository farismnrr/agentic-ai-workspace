package mcp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func cleanErrorDetail(err, known error) string {
	msg := err.Error()
	knownMsg := known.Error()
	if !strings.HasSuffix(msg, knownMsg) {
		return ""
	}
	prefix := strings.TrimSuffix(msg, knownMsg)
	prefix = strings.TrimSuffix(strings.TrimSpace(prefix), ":")
	if idx := strings.LastIndex(prefix, ": "); idx >= 0 {
		prefix = prefix[idx+2:]
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || strings.ContainsAny(prefix, "\r\n") || len(prefix) > 120 {
		return ""
	}
	return prefix
}

var errInternal = errors.New("internal error")

// UserFacing keeps domain errors readable for the model and hides everything else.
func UserFacing(err error) error {
	for _, known := range knownErrors {
		if errors.Is(err, known) {
			if errors.Is(err, taskdomain.ErrBadRequest) || errors.Is(err, taskdomain.ErrTooLarge) {
				if detail := cleanErrorDetail(err, known); detail != "" {
					return fmt.Errorf("%w: %s", known, detail)
				}
			}
			return known
		}
	}

	return fmt.Errorf("the workspace could not be used: %w", errInternal)
}

var knownErrors = []error{
	domain.ErrDenied, domain.ErrNotFound, domain.ErrIsDir, domain.ErrNotDir, domain.ErrBinary, domain.ErrEmptyQuery,
	domain.ErrTooLarge, domain.ErrExists, domain.ErrNotUnique, domain.ErrNoMatch, domain.ErrBadRange,
	domain.ErrBadPattern, domain.ErrNoTasks, domain.ErrUnknownTask, domain.ErrBadTasks, domain.ErrTaskBusy,
	domain.ErrServerBusy, domain.ErrUnknownProject, domain.ErrBadGitArgs, domain.ErrBadVerifyArgs,
	domain.ErrBadQuestion, errBadMessage, taskdomain.ErrUnknownWork,
	taskdomain.ErrWorkClosed, taskdomain.ErrStaleRevision, taskdomain.ErrKeyReused, taskdomain.ErrBadRole, taskdomain.ErrRoleForbidden,
	errMoLabDeveloperRequired,
	taskdomain.ErrTooMany, taskdomain.ErrRateLimited, taskdomain.ErrTooLarge, taskdomain.ErrBadRequest,
	taskdomain.ErrNoProject, taskdomain.ErrWrongState, taskdomain.ErrAwaitingPerson, taskdomain.ErrNoResearch,
	taskdomain.ErrOpenQuestions, taskdomain.ErrTryAgain, taskdomain.ErrUnverifiedUser,
}
