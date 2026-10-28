package activity_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

func TestRecorderRedactsExternalWorkToolActivityWithoutWorkID(t *testing.T) {
	store := &captureActivities{}
	log := spaceapp.NewActivityLog(store, fixedActivityID{}, nil, time.Now)
	activityRecorder := bootstrap.NewRecorder(log)
	workTools := []string{
		codingdomain.ToolBeginWork.Name, codingdomain.ToolSetRole.Name, codingdomain.ToolUpdatePlan.Name,
		codingdomain.ToolSaveResearch.Name, codingdomain.ToolSaveRequirements.Name,
		codingdomain.ToolReportProgress.Name, codingdomain.ToolCompleteWork.Name, codingdomain.ToolGetWorkState.Name,
		codingdomain.ToolSubmitPlan.Name,
	}
	for _, name := range workTools {
		if err := activityRecorder.Record(context.Background(), codingmcp.Entry{
			UserID: "u", Tool: name, Summary: "token=secret-value", Detail: "DATABASE_PASSWORD=hunter2",
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i, entry := range store.entries {
		if entry.Summary == "token=secret-value" || entry.Detail == "DATABASE_PASSWORD=hunter2" {
			t.Errorf("%s activity was not redacted: %+v", workTools[i], entry)
		}
	}

	if err := activityRecorder.Record(context.Background(), codingmcp.Entry{
		UserID: "u", Tool: "read_file", Summary: "legacy token=secret-value",
	}); err != nil {
		t.Fatal(err)
	}
	if got := store.entries[len(workTools)].Summary; got != "legacy token=secret-value" {
		t.Fatalf("legacy activity changed: %q", got)
	}
}

type captureActivities struct{ entries []spacedomain.Activity }

func (s *captureActivities) AddActivity(_ context.Context, entry spacedomain.Activity) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (*captureActivities) ListActivity(context.Context, string, int) ([]spacedomain.Activity, error) {
	return nil, nil
}

func (*captureActivities) FindActivity(context.Context, string) (spacedomain.Activity, error) {
	return spacedomain.Activity{}, spacedomain.ErrNoActivity
}

func (*captureActivities) MarkUndone(context.Context, string, time.Time) error { return nil }

type fixedActivityID struct{}

func (fixedActivityID) ID() (string, error) { return "activity-id", nil }
