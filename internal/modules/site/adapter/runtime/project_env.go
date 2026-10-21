package runtime

import (
	"context"
	"errors"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
)

var errProjectEnvironmentUnavailable = errors.New("project environment unavailable")

func (r *Runtime) projectEnvironment(
	ctx context.Context, projectID string, base []string,
) ([]string, error) {
	if r.env == nil {
		return base, nil
	}
	values, err := r.env.Values(ctx, projectID)
	if err != nil {
		return nil, errProjectEnvironmentUnavailable
	}
	return codingrunner.MergeProjectEnvironment(base, values), nil
}
