package mcp

import (
	"context"
	"testing"
)

func TestLegacyTargetWithoutWorkIDSkipsWorkValidation(t *testing.T) {
	works := &fakeWorkSessions{}
	tools := &tools{workspaces: fakeWorkspaces{}, works: works}
	ctx := context.WithValue(t.Context(), identityKey{}, identity{User: "u1", Client: "ChatGPT"})
	services, err := tools.services(ctx)
	if err != nil || services.ProjectID != "project-1" || works.validations != 0 {
		t.Fatalf("legacy service resolution = %+v, validations %d, error %v", services, works.validations, err)
	}
}
