package application

import (
	"context"
	"testing"
)

func TestIndependentWorksDoNotMixAcrossProjectsOrWithinOneProject(t *testing.T) {
	works, store, _ := newWorksTest(t)
	inputs := []BeginInput{beginInput("first"), beginInput("same project")}
	otherProject := beginInput("other project")
	otherProject.ProjectID = "other"
	inputs = append(inputs, otherProject)
	results := make([]BeginResult, len(inputs))
	for i, input := range inputs {
		result, err := works.Begin(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		results[i] = result
	}
	for i, result := range results {
		role := "Developer"
		if i == 1 {
			role = "Researcher"
		}
		if _, err := works.SetRole(context.Background(), "u1", "ChatGPT",
			inputs[i].ProjectID, result.WorkID, role); err != nil {
			t.Fatal(err)
		}
		if err := works.Progress(context.Background(), "u1", "ChatGPT", inputs[i].ProjectID, result.WorkID,
			"progress-"+result.WorkID, nil); err != nil {
			t.Fatal(err)
		}
	}
	assertIndependentWorkEvents(t, store, inputs, results)
}

func assertIndependentWorkEvents(t *testing.T, store *workFakeStore, inputs []BeginInput, results []BeginResult) {
	t.Helper()
	seenTasks := map[string]bool{}
	for i, result := range results {
		work := store.works[result.WorkID]
		task := store.tasks[work.TaskID]
		events := store.events[work.TaskID]
		wantRole := "Developer"
		if i == 1 {
			wantRole = "Researcher"
		}
		if seenTasks[task.ID] || work.ProjectID != inputs[i].ProjectID || work.RoleName != wantRole ||
			events[len(events)-1].Text != "progress-"+work.ID {
			t.Fatalf("work context mixed: work=%+v task=%+v events=%+v", work, task, events)
		}
		seenTasks[task.ID] = true
	}
}
