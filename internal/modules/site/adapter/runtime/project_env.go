package runtime

import (
	"context"
	"errors"
	"strconv"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

var errProjectEnvironmentUnavailable = errors.New("project environment unavailable")

// WithDBTunnel lets a project's site reach the one database its connection was made from.
func (r *Runtime) WithDBTunnel(tunnel codingrunner.DBTunnel) *Runtime {
	r.tunnel = tunnel

	return r
}

// siteSpec describes the container for a site: the shared sandbox, the app's port, the project's
// values, and the project's database when it has one.
func (r *Runtime) siteSpec(
	ctx context.Context, docker *sandbox.Docker, project, dir string, start application.StartSpec,
) (sandbox.Spec, error) {
	port := strconv.Itoa(start.Port)
	tunnel := codingrunner.TunnelTarget(ctx, r.tunnel, start.ProjectID)

	var hosts []string
	if tunnel != "" {
		hosts = []string{tunnel}
	}

	spec := r.engine.SiteSpec(ctx, project, docker, []string{"bash", "-c", start.Command},
		[]string{"SITE_PORT=" + port, "PORT=" + port, "HOST=127.0.0.1", "BASE_PATH=" + start.BasePath}, hosts...)

	if err := r.projectEnvironment(ctx, start.ProjectID, &spec, tunnel); err != nil {
		return sandbox.Spec{}, err
	}

	spec.Site = dir

	return spec, nil
}

func (r *Runtime) projectEnvironment(ctx context.Context, projectID string, spec *sandbox.Spec, tunnel string) error {
	var values map[string]string

	if r.env != nil {
		var err error
		if values, err = r.env.Values(ctx, projectID); err != nil {
			return errProjectEnvironmentUnavailable
		}
	}

	values = codingrunner.UseTunnel(spec, tunnel, values)
	spec.Env = codingrunner.MergeProjectEnvironment(spec.Env, values)

	return nil
}
