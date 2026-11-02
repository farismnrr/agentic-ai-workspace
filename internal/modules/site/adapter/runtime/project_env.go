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

// WithReach lets a project's site reach what the server granted it, such as its database and model gateway.
func (r *Runtime) WithReach(reach codingrunner.Reach) *Runtime {
	r.reach = reach

	return r
}

// siteSpec describes the container for a site: the shared sandbox, the app's port, the project's
// values, and what the project was granted to reach.
func (r *Runtime) siteSpec(
	ctx context.Context, docker *sandbox.Docker, project, dir string, start application.StartSpec,
) (sandbox.Spec, error) {
	port := strconv.Itoa(start.Port)
	granted := codingrunner.GrantedFor(ctx, r.reach, start.ProjectID)

	argv, err := application.SiteArgv(start.Command)
	if err != nil {
		return sandbox.Spec{}, err
	}
	writable, err := WorkspaceMounts(project)
	if err != nil {
		return sandbox.Spec{}, err
	}
	spec := r.engine.SiteSpec(ctx, project, docker, argv,
		[]string{"SITE_PORT=" + port, "PORT=" + port, "HOST=127.0.0.1", "BASE_PATH=" + start.BasePath,
			"APP_DATA_DIR=" + sandbox.SiteDir + "/data", "XDG_DATA_HOME=" + sandbox.SiteDir + "/data"}, granted.Targets...)

	if err := r.projectEnvironment(ctx, start.ProjectID, &spec, granted); err != nil {
		return sandbox.Spec{}, err
	}

	spec.Site = dir
	spec.WorkspaceWritable = writable

	return spec, nil
}

func (r *Runtime) projectEnvironment(
	ctx context.Context, projectID string, spec *sandbox.Spec, granted codingrunner.Granted,
) error {
	var values map[string]string

	if r.env != nil {
		var err error
		if values, err = r.env.Values(ctx, projectID); err != nil {
			return errProjectEnvironmentUnavailable
		}
	}

	values, granting := granted.Apply(spec, values)
	spec.Env = codingrunner.MergeProjectEnvironment(append(spec.Env, granting...), values)

	return nil
}
