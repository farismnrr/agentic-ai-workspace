package application_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// memoryShots keeps screenshots in memory.
type memoryShots struct{ saved map[string][]byte }

func (m *memoryShots) Save(_ context.Context, taskID string, png []byte) (string, error) {
	id := "s" + strconv.Itoa(len(m.saved))
	m.saved[taskID+"/"+id] = png

	return id, nil
}

func (m *memoryShots) Load(_ context.Context, taskID, id string) ([]byte, error) {
	if data, ok := m.saved[taskID+"/"+id]; ok {
		return data, nil
	}

	return nil, domain.ErrNotFound
}

func (m *memoryShots) Remove(context.Context, string) error { return nil }

func shotTool() *fakeTools {
	return &fakeTools{results: map[string]application.ToolResult{
		"screenshot": {Output: "Screenshot of /", Image: []byte("PNG")},
	}}
}

func anyImage(request application.Request) bool {
	for _, msg := range request.Messages {
		for _, block := range msg.Blocks {
			if len(block.Data) > 0 {
				return true
			}
		}
	}

	return false
}

func TestAScreenshotIsSavedShownAndGivenToTheModel(t *testing.T) {
	t.Parallel()

	shots := &memoryShots{saved: map[string][]byte{}}
	r := newRigWith(nil, func(d *application.Deps, _ *application.Limits) { d.Shots = shots; d.Tools = shotTool() },
		call("c1", "screenshot", map[string]any{}), finish("Done"))
	task := r.start(t, "Look at the page")
	r.run(task.ID)

	event := r.lastEvent(task.ID, domain.EventShot)
	if event.Image == "" || string(shots.saved[task.ID+"/"+event.Image]) != "PNG" {
		t.Fatalf("the person sees the screenshot in the thread: %+v", event)
	}

	last := r.model.requests[len(r.model.requests)-1]
	got := resultFor(last, "c1")
	if got.Image == "" || string(got.Data) != "PNG" || !strings.Contains(got.Text, "Screenshot") {
		t.Fatalf("the model gets the picture with the report: %+v", got)
	}
}

func TestAModelThatCannotReadImagesGetsTheTextReportInstead(t *testing.T) {
	t.Parallel()

	shots := &memoryShots{saved: map[string][]byte{}}
	r := newRigWith(nil, func(d *application.Deps, _ *application.Limits) { d.Shots = shots; d.Tools = shotTool() },
		call("c1", "screenshot", map[string]any{}),
		failing("this model does not accept images", false),
		finish("Done"))
	task := r.start(t, "Look at the page")
	r.run(task.ID)

	if got := r.task(t, task.ID); got.Status == domain.StatusFailed {
		t.Fatalf("a refused picture must not fail the task: %+v", got)
	}

	if len(r.model.requests) != 3 || !anyImage(r.model.requests[1]) || anyImage(r.model.requests[2]) {
		t.Fatalf("the second try has no pictures: %d requests", len(r.model.requests))
	}

	if resultFor(r.model.requests[2], "c1").Text == "" {
		t.Fatal("the text report is still there")
	}
}
