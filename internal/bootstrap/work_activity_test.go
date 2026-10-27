package bootstrap

import (
	"context"
	"testing"
	"time"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

func TestRecorderRedactsExplicitWorkActivityOnly(t *testing.T) {
	store := &captureActivities{}
	log := spaceapp.NewActivityLog(store, fixedActivityID{}, nil, time.Now)
	activityRecorder := recorder{log: log}
	if err := activityRecorder.Record(context.Background(), codingmcp.Entry{
		UserID: "u", WorkID: "work-1", Summary: "command token=secret-value",
		Detail: "DATABASE_PASSWORD=hunter2",
	}); err != nil {
		t.Fatal(err)
	}
	if len(store.entries) != 1 || store.entries[0].Summary == "command token=secret-value" ||
		store.entries[0].Detail == "DATABASE_PASSWORD=hunter2" {
		t.Fatalf("work activity was not redacted: %+v", store.entries)
	}

	if err := activityRecorder.Record(context.Background(), codingmcp.Entry{
		UserID: "u", Summary: "legacy token=secret-value",
	}); err != nil {
		t.Fatal(err)
	}
	if got := store.entries[1].Summary; got != "legacy token=secret-value" {
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
