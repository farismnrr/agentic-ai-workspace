package application

import (
	"context"
	"sync"
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

func TestConcurrentBeginKeepsRoleAndEventsOnTheirOwnSessions(t *testing.T) {
	works, store, _ := newWorksTest(t)
	inputs := []BeginInput{beginInput("developer request"), beginInput("research request")}
	inputs[1].Role = "researcher"
	start := make(chan struct{})
	results := make([]BeginResult, len(inputs))
	errs := make([]error, len(inputs))
	var group sync.WaitGroup
	for i := range inputs {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			results[index], errs[index] = works.Begin(context.Background(), inputs[index])
		}(i)
	}
	close(start)
	group.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
		work := store.works[results[i].WorkID]
		roleEvent := store.events[work.TaskID][1]
		if work.RoleKey != inputs[i].Role || roleEvent.Work.Role != inputs[i].Role {
			t.Fatalf("session %d has mixed role attribution: work=%+v event=%+v", i, work, roleEvent)
		}
	}
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
