package application

import (
	"context"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func begunWork(t *testing.T) (*Works, *workFakeStore, BeginResult) {
	t.Helper()
	works, store, _ := newWorksTest(t)
	result, err := works.Begin(context.Background(), beginInput("Build a small page"))
	if err != nil {
		t.Fatal(err)
	}
	return works, store, result
}

func TestResearchAndRequirementsAreSeparateAndRedacted(t *testing.T) {
	works, store, begun := begunWork(t)
	saveResearchRound(t, works, begun.WorkID, "Found token=secret-value", []string{"https://example.test/source"})
	saveResearchRound(t, works, begun.WorkID, "Found token=secret-value", []string{"https://example.test/source"})
	saveResearchRound(t, works, begun.WorkID, "Second research round", nil)
	err := works.SaveRequirements(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID, "Requirements body")
	if err != nil {
		t.Fatal(err)
	}
	task := store.tasks[store.works[begun.WorkID].TaskID]
	events := store.events[task.ID]
	files := taskNoteFiles(task, events)
	assertResearchAndRequirements(t, task, files)
}

func saveResearchRound(t *testing.T, works *Works, workID, markdown string, sources []string) {
	t.Helper()
	if err := works.SaveResearch(context.Background(), "u1", "ChatGPT", "proj", workID, markdown, sources); err != nil {
		t.Fatal(err)
	}
}

func assertResearchAndRequirements(t *testing.T, task domain.Task, files []NoteFile) {
	t.Helper()
	if !strings.Contains(task.Research, "Second research round") || !strings.Contains(task.Research, "[redacted]") ||
		task.PRD != "Requirements body" || strings.Contains(task.Research, task.PRD) {
		t.Fatalf("documents were mixed or not redacted: %+v", task)
	}
	joined := ""
	for _, file := range files {
		joined += file.Path + "\n" + file.Content + "\n"
	}
	if !strings.Contains(joined, "/research/") || !strings.Contains(joined, "/prd/") ||
		!strings.Contains(joined, "external assistant, self-reported") || strings.Contains(joined, "secret-value") {
		t.Fatalf("external documents were not exported separately and safely: %s", joined)
	}
}

func TestProgressStoresRedactedClaimedEvidence(t *testing.T) {
	works, store, begun := begunWork(t)
	if err := works.Progress(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		"Changed token=secret-value", []string{"ran password=hunter2"}); err != nil {
		t.Fatal(err)
	}
	events := store.events[store.works[begun.WorkID].TaskID]
	event := events[len(events)-1]
	if !strings.Contains(event.Text, "[redacted]") || !strings.Contains(event.Technical, "[redacted]") ||
		event.Work.Evidence != domain.EvidenceClaimed {
		t.Fatalf("progress evidence was not safely attributed: %+v", event)
	}
}
