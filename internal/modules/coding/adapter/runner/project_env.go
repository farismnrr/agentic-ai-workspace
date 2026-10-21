package runner

import (
	"context"
	"errors"
)

var errProjectEnvironmentUnavailable = errors.New("project environment unavailable")

func (r *Runner) projectValues(ctx context.Context) (map[string]string, error) {
	if r.env == nil || r.projectID == "" {
		return nil, nil
	}

	values, err := r.env.Values(ctx, r.projectID)
	if err != nil {
		return nil, errProjectEnvironmentUnavailable
	}
	return values, nil
}
