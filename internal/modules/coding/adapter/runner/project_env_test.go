package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
)

var errPrivateEnvValue = errors.New("failed with private-value")

type projectEnvStub struct {
	values map[string]map[string]string
	err    error
}

func (s projectEnvStub) Values(_ context.Context, projectID string) (map[string]string, error) {
	return s.values[projectID], s.err
}

func TestProjectEnvironmentIsScopedAndFailuresAreGeneric(t *testing.T) {
	values := projectEnvStub{values: map[string]map[string]string{
		"project-a": {"PROJECT_TOKEN": "a-only"},
		"project-b": {"PROJECT_TOKEN": "b-only"},
	}}
	for projectID, want := range map[string]string{"project-a": "a-only", "project-b": "b-only"} {
		run := (&Runner{projectID: projectID, env: values})
		loaded, err := run.projectValues(context.Background())
		if err != nil || loaded["PROJECT_TOKEN"] != want || len(loaded) != 1 {
			t.Fatalf("project %s values = %#v, %v", projectID, loaded, err)
		}
	}

	failed := &Runner{projectID: "project-a", env: projectEnvStub{err: errPrivateEnvValue}}
	_, err := failed.projectValues(context.Background())
	if err == nil || strings.Contains(err.Error(), "private-value") {
		t.Fatalf("project environment failure exposed a value: %v", err)
	}
}
